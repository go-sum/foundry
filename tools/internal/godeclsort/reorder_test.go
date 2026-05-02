package godeclsort

import "testing"

func TestReorderSource_ReordersTopLevelDecls(t *testing.T) {
	input := `package sample

import "fmt"

func beta() {
	fmt.Println("beta")
}

const answer = 42

type thing struct{}

func alpha() {
	fmt.Println("alpha")
}
`

	want := `package sample

import "fmt"

type thing struct{}

const answer = 42

func beta() {
	fmt.Println("beta")
}

func alpha() {
	fmt.Println("alpha")
}
`

	got, err := ReorderSource([]byte(input))
	if err != nil {
		t.Fatalf("ReorderSource: %v", err)
	}
	if string(got) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReorderSource_PreservesVarBarrier(t *testing.T) {
	input := `package sample

func beforeVar() {}

var global = sideEffect()

func afterVar() {}

const answer = 42

type thing struct{}
`

	want := `package sample

func beforeVar() {}

var global = sideEffect()

type thing struct{}

const answer = 42

func afterVar() {}
`

	got, err := ReorderSource([]byte(input))
	if err != nil {
		t.Fatalf("ReorderSource: %v", err)
	}
	if string(got) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReorderSource_PreservesInitBarrierAndDocComments(t *testing.T) {
	input := `package sample

func beta() {}

func init() {}

// widget documents the type.
type widget struct{}

// answer documents the constant.
const answer = 42
`

	want := `package sample

func beta() {}

func init() {}

// widget documents the type.
type widget struct{}

// answer documents the constant.
const answer = 42
`

	got, err := ReorderSource([]byte(input))
	if err != nil {
		t.Fatalf("ReorderSource: %v", err)
	}
	if string(got) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReorderSource_IsIdempotent(t *testing.T) {
	input := `package sample

type thing struct{}

const answer = 42

func alpha() {}
`

	once, err := ReorderSource([]byte(input))
	if err != nil {
		t.Fatalf("ReorderSource first pass: %v", err)
	}

	twice, err := ReorderSource(once)
	if err != nil {
		t.Fatalf("ReorderSource second pass: %v", err)
	}

	if string(twice) != string(once) {
		t.Fatalf("got:\n%s\nwant:\n%s", twice, once)
	}
}
