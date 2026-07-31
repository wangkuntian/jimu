package pagination

import "testing"

func TestNormalizeDefaultsAndCapsPageSize(t *testing.T) {
	p := Pagination{}
	if err := p.Normalize("id", "created_at"); err != nil {
		t.Fatal(err)
	}
	if p.Page != 1 || p.PageSize != 20 || p.Order != "desc" || p.Sort != "id" {
		t.Fatalf("pagination = %#v", p)
	}

	p = Pagination{Page: 2, PageSize: 500, Sort: "created_at", Order: "ASC", Filter: " alice "}
	if err := p.Normalize("id", "created_at"); err != nil {
		t.Fatal(err)
	}
	if p.PageSize != 100 || p.Order != "asc" || p.Filter != "alice" {
		t.Fatalf("pagination = %#v", p)
	}
}

func TestNormalizeRejectsInvalidSortAndOrder(t *testing.T) {
	for _, p := range []Pagination{
		{Sort: "password"},
		{Order: "drop table"},
	} {
		if err := p.Normalize("id", "created_at"); err == nil {
			t.Fatalf("expected error for %#v", p)
		}
	}
}
