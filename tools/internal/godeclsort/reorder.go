package godeclsort

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
)

type declKind int

const (
	declKindOther declKind = iota
	declKindType
	declKindConst
	declKindFunc
)

type declBlock struct {
	kind declKind
	text []byte
}

// ReorderSource formats a Go source file after reordering top-level type,
// const, and non-init func declarations. Top-level vars and init functions are
// left in place to avoid changing package initialization order.
func ReorderSource(src []byte) ([]byte, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	if len(file.Decls) == 0 {
		return format.Source(src)
	}

	starts, err := declStarts(fset, file.Decls)
	if err != nil {
		return nil, err
	}

	bodyStart := firstNonImportDecl(file.Decls)
	if bodyStart == len(file.Decls) {
		return format.Source(src)
	}

	headerEnd := starts[bodyStart]
	blocks, err := bodyBlocks(src, file.Decls[bodyStart:], starts[bodyStart:])
	if err != nil {
		return nil, err
	}

	ordered := append([]byte{}, src[:headerEnd]...)
	for _, block := range reorderBlocks(blocks) {
		ordered = append(ordered, block.text...)
	}

	formatted, err := format.Source(ordered)
	if err != nil {
		return nil, fmt.Errorf("gofmt: %w", err)
	}
	return formatted, nil
}

func declStarts(fset *token.FileSet, decls []ast.Decl) ([]int, error) {
	starts := make([]int, 0, len(decls))
	for _, decl := range decls {
		pos := decl.Pos()
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Doc != nil {
				pos = d.Doc.Pos()
			}
		case *ast.FuncDecl:
			if d.Doc != nil {
				pos = d.Doc.Pos()
			}
		}

		offset := fset.PositionFor(pos, false).Offset
		if offset < 0 {
			return nil, fmt.Errorf("decl start: invalid offset")
		}
		starts = append(starts, offset)
	}
	return starts, nil
}

func firstNonImportDecl(decls []ast.Decl) int {
	for i, decl := range decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			return i
		}
	}
	return len(decls)
}

func bodyBlocks(src []byte, decls []ast.Decl, starts []int) ([]declBlock, error) {
	blocks := make([]declBlock, 0, len(decls))
	for i, decl := range decls {
		end := len(src)
		if i+1 < len(starts) {
			end = starts[i+1]
		}
		if starts[i] > end || end > len(src) {
			return nil, fmt.Errorf("decl block: invalid offsets")
		}

		blocks = append(blocks, declBlock{
			kind: classifyDecl(decl),
			text: append([]byte(nil), src[starts[i]:end]...),
		})
	}
	return blocks, nil
}

func classifyDecl(decl ast.Decl) declKind {
	switch d := decl.(type) {
	case *ast.GenDecl:
		switch d.Tok {
		case token.TYPE:
			return declKindType
		case token.CONST:
			return declKindConst
		default:
			return declKindOther
		}
	case *ast.FuncDecl:
		if d.Recv == nil && d.Name != nil && d.Name.Name == "init" {
			return declKindOther
		}
		return declKindFunc
	default:
		return declKindOther
	}
}

func reorderBlocks(blocks []declBlock) []declBlock {
	ordered := make([]declBlock, 0, len(blocks))
	movable := make([]declBlock, 0, len(blocks))

	flush := func() {
		if len(movable) == 0 {
			return
		}
		ordered = append(ordered, filterBlocks(movable, declKindType)...)
		ordered = append(ordered, filterBlocks(movable, declKindConst)...)
		ordered = append(ordered, filterBlocks(movable, declKindFunc)...)
		movable = movable[:0]
	}

	for _, block := range blocks {
		if block.kind == declKindOther {
			flush()
			ordered = append(ordered, block)
			continue
		}
		movable = append(movable, block)
	}
	flush()

	return ordered
}

func filterBlocks(blocks []declBlock, kind declKind) []declBlock {
	filtered := make([]declBlock, 0, len(blocks))
	for _, block := range blocks {
		if block.kind == kind {
			filtered = append(filtered, block)
		}
	}
	return filtered
}
