package main

import "testing"

const numstatOutput = "\x00abc123\n" +
	"12\t3\tapi/src/Main.java\n" +
	"4\t0\tapi/pom.xml\n" +
	"\x00def456\n" +
	"7\t7\tapi/src/Other.java\n"

func TestParseChurnSumsAddedAndDeleted(t *testing.T) {
	got := parseChurn(numstatOutput)
	if got.Added != 23 {
		t.Errorf("Added = %d want 23", got.Added)
	}
	if got.Deleted != 10 {
		t.Errorf("Deleted = %d want 10", got.Deleted)
	}
}

func TestParseChurnCountsCommits(t *testing.T) {
	if got := parseChurn(numstatOutput).Commits; got != 2 {
		t.Errorf("Commits = %d want 2", got)
	}
}

func TestParseChurnIgnoresBinaryFiles(t *testing.T) {
	got := parseChurn("\x00abc\n-\t-\tlogo.png\n5\t1\tapp.go\n")
	if got.Added != 5 || got.Deleted != 1 {
		t.Errorf("got %+v want 5/1", got)
	}
}

func TestParseChurnEmptyOutputIsIdle(t *testing.T) {
	got := parseChurn("")
	if got.Total() != 0 || got.Commits != 0 {
		t.Errorf("got %+v want zero churn", got)
	}
}

func TestChurnTotalIsAddedPlusDeleted(t *testing.T) {
	c := Churn{Added: 10, Deleted: 4}
	if c.Total() != 14 {
		t.Errorf("Total = %d want 14", c.Total())
	}
}
