package main

import (
	"strings"
	"testing"
)

func plainMarkdown(src string) []string {
	return renderMarkdown(strings.Split(src, "\n"), painter(false))
}

func TestMarkdownHeadingsGetLevelMarkers(t *testing.T) {
	got := plainMarkdown("# One\n## Two\n### Three\n#### Four")
	want := []string{"═══ One", "─── Two", "▸ Three", "▸ Four"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q want %q", i, got[i], want[i])
		}
	}
}

func TestMarkdownHeadingKeepsClosingHashesOut(t *testing.T) {
	if got := plainMarkdown("## Flags ##")[0]; got != "─── Flags" {
		t.Errorf("got %q", got)
	}
}

func TestMarkdownBulletsAndOrderedItems(t *testing.T) {
	got := plainMarkdown("- first\n  * nested\n1. one\n2) two")
	want := []string{"• first", "  • nested", "1. one", "2. two"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q want %q", i, got[i], want[i])
		}
	}
}

func TestMarkdownTaskItems(t *testing.T) {
	got := plainMarkdown("- [x] done\n- [ ] todo")
	if got[0] != "☑ done" || got[1] != "☐ todo" {
		t.Errorf("got %q and %q", got[0], got[1])
	}
}

func TestMarkdownFencedCodeKeepsMarkupLiteral(t *testing.T) {
	got := plainMarkdown("```go\nx := *p // **not bold**\n```")
	if len(got) != 1 || got[0] != "│ x := *p // **not bold**" {
		t.Errorf("got %q", got)
	}
}

func TestMarkdownUnterminatedFenceStillRenders(t *testing.T) {
	got := plainMarkdown("```\ngo install ./...")
	if len(got) != 1 || got[0] != "│ go install ./..." {
		t.Errorf("got %q", got)
	}
}

func TestMarkdownInlineEmphasisAndCode(t *testing.T) {
	got := plainMarkdown("**bold** *em* ~~gone~~ `code`")[0]
	if got != "bold em gone code" {
		t.Errorf("got %q", got)
	}
}

func TestMarkdownCodeSpanProtectsMarkup(t *testing.T) {
	got := plainMarkdown("use `a *b* c` here")[0]
	if got != "use a *b* c here" {
		t.Errorf("got %q", got)
	}
}

func TestMarkdownLinksShowDestination(t *testing.T) {
	got := plainMarkdown("see [the docs](https://example.com/x) now")[0]
	if got != "see the docs [https://example.com/x] now" {
		t.Errorf("got %q", got)
	}
}

func TestMarkdownImagesGetAMarker(t *testing.T) {
	got := plainMarkdown("![logo](img/logo.png)")[0]
	if !strings.Contains(got, "🖼") || !strings.Contains(got, "[img/logo.png]") {
		t.Errorf("got %q", got)
	}
}

func TestMarkdownBlockQuoteAndRule(t *testing.T) {
	got := plainMarkdown("> note\n---")
	if got[0] != "▌ note" {
		t.Errorf("quote = %q", got[0])
	}
	if got[1] != strings.Repeat("─", 60) {
		t.Errorf("rule = %q", got[1])
	}
}

func TestMarkdownTableAlignsColumns(t *testing.T) {
	got := plainMarkdown("| Flag | Meaning |\n|---|---|\n| --long | show the last commit |\n| --raw | off |")
	want := []string{
		"┌────────┬──────────────────────┐",
		"│ Flag   │ Meaning              │",
		"├────────┼──────────────────────┤",
		"│ --long │ show the last commit │",
		"│ --raw  │ off                  │",
		"└────────┴──────────────────────┘",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d lines:\n%s", len(got), strings.Join(got, "\n"))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d =\n%q want\n%q", i, got[i], want[i])
		}
	}
}

func TestMarkdownTableWidthIgnoresAnsi(t *testing.T) {
	lines := renderMarkdown(strings.Split("| A | B |\n|---|---|\n| `x` | **yy** |", "\n"), painter(true))
	var widths []int
	for _, l := range lines {
		widths = append(widths, visibleLength(l))
	}
	for i, w := range widths {
		if w != widths[0] {
			t.Fatalf("line %d width %d differs from %d:\n%s", i, w, widths[0], strings.Join(lines, "\n"))
		}
	}
}

func TestMarkdownColorIsOptional(t *testing.T) {
	src := []string{"# Title", "- **bold** item", "| a | b |", "|---|---|", "| 1 | 2 |"}
	for _, l := range renderMarkdown(src, painter(false)) {
		if strings.Contains(l, "\x1b[") {
			t.Errorf("escape leaked with colour off: %q", l)
		}
	}
	colored := strings.Join(renderMarkdown(src, painter(true)), "\n")
	if !strings.Contains(colored, "\x1b[") {
		t.Error("expected escapes with colour on")
	}
}

func TestMarkdownKeepsOneOutputLinePerSourceLine(t *testing.T) {
	src := []string{"# Title", "", "Body text.", "- item", "> quote"}
	if got := renderMarkdown(src, painter(false)); len(got) != len(src) {
		t.Errorf("got %d lines for %d source lines: %q", len(got), len(src), got)
	}
}
