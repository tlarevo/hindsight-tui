package views

import (
	"testing"

	"charm.land/bubbles/v2/table"
)

func TestPagedTableCanNext(t *testing.T) {
	t.Parallel()
	pt := newPagedTable(nil, []table.Column{{Title: "X", Width: 8}})

	pt.total = 50
	pt.offset = 25
	if pt.CanNext() {
		t.Fatal("CanNext(offset=25,total=50,limit=25) = true, want false (last page)")
	}

	pt.offset = 0
	if !pt.CanNext() {
		t.Fatal("CanNext(offset=0,total=50) = false, want true")
	}

	// Unknown total: a full page implies there may be more.
	pt.total = 0
	pt.offset = 0
	rows := make([]table.Row, pt.limit)
	for i := range rows {
		rows[i] = table.Row{"x"}
	}
	pt.SetPage(rows, 0)
	if !pt.CanNext() {
		t.Fatal("CanNext(total=0, full page) = false, want true")
	}
}

func TestPagedTableEndLoadDropsStaleGen(t *testing.T) {
	t.Parallel()

	pt := newPagedTable(nil, nil)
	gen1, _ := pt.BeginLoad()
	gen2, _ := pt.BeginLoad()

	if pt.EndLoad(gen1) {
		t.Fatalf("EndLoad(stale gen %d) = true, want false (current gen %d)", gen1, gen2)
	}
	if !pt.EndLoad(gen2) {
		t.Fatal("EndLoad(current gen) = false, want true")
	}
}
