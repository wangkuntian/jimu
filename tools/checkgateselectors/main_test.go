package main

import "testing"

func TestParseSelectors(t *testing.T) {
	src := "SCAFFOLD_OFFICIAL_TESTS := ^(TestA|TestB)$\nSCAFFOLD_UPGRADE_TESTS := ^(TestC)$\n"
	got, err := parseSelectors(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d selectors, want 2", len(got))
	}
	if got["SCAFFOLD_OFFICIAL_TESTS"] != "^(TestA|TestB)$" {
		t.Fatalf("official = %q", got["SCAFFOLD_OFFICIAL_TESTS"])
	}
}

func TestParseSelectorsRejectsEmptyValue(t *testing.T) {
	if _, err := parseSelectors("SCAFFOLD_OFFICIAL_TESTS := \n"); err == nil {
		t.Fatal("empty selector value must be an error")
	}
}

func TestEmptyMatchIsAnError(t *testing.T) {
	if err := requireMatches("^TestNope$", "ok  \tjimu/tools/generator\t0.5s\n"); err == nil {
		t.Fatal("empty match must be an error")
	}
}

func TestNonEmptyMatchPasses(t *testing.T) {
	out := "TestAlpha\nTestBeta\nok  \tjimu/tools/generator\t0.5s\n"
	if err := requireMatches("^Test(Alpha|Beta)$", out); err != nil {
		t.Fatal(err)
	}
}
