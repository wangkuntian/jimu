package sqlpattern

import "testing"

func TestEscapeLike(t *testing.T) {
	cases := map[string]string{
		"100%":  "100!%",
		"a_b":   "a!_b",
		"!":     "!!",
		"普通文本":  "普通文本",
		"":      "",
		"50%_x": "50!%!_x",
	}
	for in, want := range cases {
		if got := EscapeLike(in); got != want {
			t.Errorf("EscapeLike(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLikeContains(t *testing.T) {
	if got := LikeContains("%数据库_"); got != "%!%数据库!_%" {
		t.Fatalf("LikeContains(%q) = %q", "%数据库_", got)
	}
	if got := LikeContains("acme"); got != "%acme%" {
		t.Fatalf("LikeContains(acme) = %q", got)
	}
}
