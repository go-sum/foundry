package data_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/go-sum/componentry/ui/data"
	testutil "github.com/go-sum/componentry/testutil"
)

func TestTable_Root(t *testing.T) {
	got := testutil.RenderNode(t, data.Table.Root())
	if !strings.Contains(got, "overflow-auto") {
		t.Errorf("Table.Root: expected overflow-auto class, got:\n%s", got)
	}
	if !strings.Contains(got, "<table") {
		t.Errorf("Table.Root: expected inner <table>, got:\n%s", got)
	}
	if !strings.Contains(got, "caption-bottom") {
		t.Errorf("Table.Root: expected caption-bottom class on table, got:\n%s", got)
	}
}

func TestTable_Root_children(t *testing.T) {
	got := testutil.RenderNode(t, data.Table.Root(g.Text("content")))
	if !strings.Contains(got, "content") {
		t.Errorf("Table.Root children: expected 'content', got:\n%s", got)
	}
}

func TestTable_Header(t *testing.T) {
	got := testutil.RenderNode(t, data.Table.Header())
	if !strings.HasPrefix(got, "<thead") {
		t.Errorf("Table.Header: expected <thead>, got:\n%s", got)
	}
	// gomponents HTML-encodes & in attribute values, so [&_tr] becomes [&amp;_tr]
	if !strings.Contains(got, "[&amp;_tr]:border-b") {
		t.Errorf("Table.Header: expected [&amp;_tr]:border-b class, got:\n%s", got)
	}
}

func TestTable_Body_withID(t *testing.T) {
	got := testutil.RenderNode(t, data.Table.Body(data.BodyProps{ID: "rows"}))
	if !strings.HasPrefix(got, "<tbody") {
		t.Errorf("Table.Body withID: expected <tbody>, got:\n%s", got)
	}
	if !strings.Contains(got, `id="rows"`) {
		t.Errorf("Table.Body withID: expected id=rows, got:\n%s", got)
	}
}

func TestTable_Body_noID(t *testing.T) {
	got := testutil.RenderNode(t, data.Table.Body(data.BodyProps{}))
	if !strings.HasPrefix(got, "<tbody") {
		t.Errorf("Table.Body noID: expected <tbody>, got:\n%s", got)
	}
	if strings.Contains(got, `id="`) {
		t.Errorf("Table.Body noID: expected no id attr, got:\n%s", got)
	}
}

func TestTable_Footer(t *testing.T) {
	got := testutil.RenderNode(t, data.Table.Footer())
	if !strings.HasPrefix(got, "<tfoot") {
		t.Errorf("Table.Footer: expected <tfoot>, got:\n%s", got)
	}
	// bg-muted/50 is rendered as-is in class value (/ is not encoded)
	if !strings.Contains(got, "bg-muted") {
		t.Errorf("Table.Footer: expected bg-muted class, got:\n%s", got)
	}
}

func TestTable_Row_default(t *testing.T) {
	got := testutil.RenderNode(t, data.Table.Row(data.RowProps{}))
	if !strings.HasPrefix(got, "<tr") {
		t.Errorf("Table.Row default: expected <tr>, got:\n%s", got)
	}
	if !strings.Contains(got, "border-b") {
		t.Errorf("Table.Row default: expected border-b class, got:\n%s", got)
	}
}

func TestTable_Row_selected(t *testing.T) {
	got := testutil.RenderNode(t, data.Table.Row(data.RowProps{Selected: true}))
	if !strings.Contains(got, "bg-muted") {
		t.Errorf("Table.Row selected: expected bg-muted class, got:\n%s", got)
	}
}

func TestTable_Head(t *testing.T) {
	got := testutil.RenderNode(t, data.Table.Head(g.Text("Name")))
	if !strings.HasPrefix(got, "<th") {
		t.Errorf("Table.Head: expected <th>, got:\n%s", got)
	}
	if !strings.Contains(got, "text-muted-foreground") {
		t.Errorf("Table.Head: expected text-muted-foreground class, got:\n%s", got)
	}
	if !strings.Contains(got, "Name") {
		t.Errorf("Table.Head: expected 'Name' text, got:\n%s", got)
	}
}

func TestTable_Cell(t *testing.T) {
	got := testutil.RenderNode(t, data.Table.Cell(g.Text("value")))
	if !strings.HasPrefix(got, "<td") {
		t.Errorf("Table.Cell: expected <td>, got:\n%s", got)
	}
	if !strings.Contains(got, "align-middle") {
		t.Errorf("Table.Cell: expected align-middle class, got:\n%s", got)
	}
	if !strings.Contains(got, "value") {
		t.Errorf("Table.Cell: expected 'value' text, got:\n%s", got)
	}
}

func TestTable_Caption(t *testing.T) {
	got := testutil.RenderNode(t, data.Table.Caption(g.Text("Results")))
	if !strings.HasPrefix(got, "<caption") {
		t.Errorf("Table.Caption: expected <caption>, got:\n%s", got)
	}
	if !strings.Contains(got, "text-muted-foreground") {
		t.Errorf("Table.Caption: expected text-muted-foreground class, got:\n%s", got)
	}
	if !strings.Contains(got, "Results") {
		t.Errorf("Table.Caption: expected 'Results' text, got:\n%s", got)
	}
}
