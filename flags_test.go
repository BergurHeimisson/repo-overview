package main

import (
	"io"
	"testing"
	"time"
)

func TestParseFlagsDefaultsToOneReadmeLine(t *testing.T) {
	o, _, err := parseFlags([]string{}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if o.Lines != 1 {
		t.Errorf("Lines = %d want 1", o.Lines)
	}
	if o.SkipStale {
		t.Error("SkipStale should default to off")
	}
	if !o.Page {
		t.Error("paging should default to on")
	}
	if o.Window != 24*time.Hour {
		t.Errorf("Window = %v want 24h", o.Window)
	}
}

func TestParseFlagsReadsSkipStaleAndPath(t *testing.T) {
	o, path, err := parseFlags([]string{"--skip-stale", "--lines", "10", "/tmp/x"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if !o.SkipStale {
		t.Error("--skip-stale should turn on stale skipping")
	}
	if o.Lines != 10 {
		t.Errorf("Lines = %d want 10", o.Lines)
	}
	if path != "/tmp/x" {
		t.Errorf("path = %q want /tmp/x", path)
	}
}

func TestParseFlagsRejectsBadWindow(t *testing.T) {
	if _, _, err := parseFlags([]string{"--since", "soon"}, io.Discard); err == nil {
		t.Error("expected an error for an unparseable window")
	}
}

func TestParseFlagsNoPagerTurnsPagingOff(t *testing.T) {
	o, _, err := parseFlags([]string{"--no-pager"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if o.Page {
		t.Error("--no-pager should turn paging off")
	}
}
