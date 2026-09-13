package main

import (
	"strings"
	"testing"
	"time"
)

func TestParseWindowAcceptsDays(t *testing.T) {
	got, err := parseWindow("7d")
	if err != nil {
		t.Fatal(err)
	}
	if got != 7*24*time.Hour {
		t.Errorf("got %v want 168h", got)
	}
}

func TestParseWindowAcceptsGoDurations(t *testing.T) {
	got, err := parseWindow("90m")
	if err != nil {
		t.Fatal(err)
	}
	if got != 90*time.Minute {
		t.Errorf("got %v want 90m", got)
	}
}

func TestParseWindowRejectsGarbage(t *testing.T) {
	if _, err := parseWindow("soon"); err == nil {
		t.Error("expected an error for an unparseable window")
	}
}

func TestSortReportsRanksByChurnThenName(t *testing.T) {
	reports := []ModuleReport{
		{Module: Module{Dir: "zeta"}, Churn: Churn{Added: 5}},
		{Module: Module{Dir: "alpha"}, Churn: Churn{Added: 5}},
		{Module: Module{Dir: "busy"}, Churn: Churn{Added: 100}},
		{Module: Module{Dir: "quiet"}},
	}
	sortReports(reports)
	var got []string
	for _, r := range reports {
		got = append(got, r.Module.Dir)
	}
	want := []string{"busy", "alpha", "zeta", "quiet"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v want %v", got, want)
		}
	}
}

func plainReport() Report {
	return Report{
		Repo: "BergurHeimisson/jetlog",
		Ref:  Ref{Name: "develop"},
		Modules: []ModuleReport{{
			Module:      Module{Dir: "pipeline", Readme: "pipeline/README.md"},
			Churn:       Churn{Added: 412, Deleted: 87, Commits: 6},
			URL:         "https://github.com/BergurHeimisson/jetlog/blob/develop/pipeline/README.md",
			ReadmeLines: []string{"# Pipeline", "Fetches feeds."},
			Truncated:   true,
			Last:        Commit{Author: "Bergur Heimisson", Subject: "add PYMNTS feed", When: time.Now().Add(-2 * time.Hour)},
		}},
	}
}

func TestRenderShowsModuleChurnAndReadme(t *testing.T) {
	out := render(plainReport(), Options{Lines: 30})
	for _, want := range []string{"pipeline", "+412", "87", "6 commits", "═══ Pipeline", "blob/develop/pipeline/README.md"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
}

func TestRenderMarksTruncatedReadme(t *testing.T) {
	out := render(plainReport(), Options{Lines: 30})
	if !strings.Contains(out, "more lines") {
		t.Errorf("expected a truncation notice:\n%s", out)
	}
}

func TestRenderHidesCommitDetailsWithShort(t *testing.T) {
	out := render(plainReport(), Options{Lines: 30})
	if strings.Contains(out, "Bergur Heimisson") {
		t.Errorf("author leaked into short output:\n%s", out)
	}
}

func TestRenderShowsCommitDetailsByDefault(t *testing.T) {
	out := render(plainReport(), Options{Lines: 30, ShowCommit: true})
	if !strings.Contains(out, "Bergur Heimisson") || !strings.Contains(out, "add PYMNTS feed") {
		t.Errorf("expected author and subject:\n%s", out)
	}
}

func TestRenderLabelsIdleModules(t *testing.T) {
	r := Report{Ref: Ref{Name: "develop"}, Modules: []ModuleReport{
		{Module: Module{Dir: "web"}},
	}}
	out := render(r, Options{Lines: 30})
	if !strings.Contains(out, "idle") {
		t.Errorf("expected an idle marker:\n%s", out)
	}
	if !strings.Contains(out, "no README.md") {
		t.Errorf("expected a missing-readme note:\n%s", out)
	}
}

func TestRenderEmitsNoAnsiWhenColorDisabled(t *testing.T) {
	out := render(plainReport(), Options{Lines: 30, ShowCommit: true})
	if strings.Contains(out, "\x1b[") {
		t.Errorf("colour escapes present with colour disabled:\n%q", out)
	}
}

func TestRenderEmitsAnsiWhenColorEnabled(t *testing.T) {
	out := render(plainReport(), Options{Lines: 30, Color: true})
	if !strings.Contains(out, "\x1b[") {
		t.Error("expected colour escapes when colour is enabled")
	}
}

func TestHumanAge(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "just now"},
		{5 * time.Minute, "5m ago"},
		{3 * time.Hour, "3h ago"},
		{50 * time.Hour, "2d ago"},
	}
	for _, c := range cases {
		if got := humanAge(c.d); got != c.want {
			t.Errorf("humanAge(%v) = %q want %q", c.d, got, c.want)
		}
	}
}

func TestRenderTruncationNoticeCountsRemainingLines(t *testing.T) {
	r := plainReport()
	r.Modules[0].ReadmeTotal = 30
	out := render(r, Options{Lines: 2})
	if !strings.Contains(out, "… 28 more lines") {
		t.Errorf("expected '… 28 more lines':\n%s", out)
	}
}

func TestRenderNeverShowsNegativeRemainingLines(t *testing.T) {
	r := plainReport()
	r.Modules[0].ReadmeTotal = 0 // total unknown
	out := render(r, Options{Lines: 2})
	if strings.Contains(out, "-1 more") || strings.Contains(out, "-2 more") {
		t.Errorf("negative line count rendered:\n%s", out)
	}
	if !strings.Contains(out, "more lines") {
		t.Errorf("expected some truncation notice:\n%s", out)
	}
}

func TestWindowLabelIsCompact(t *testing.T) {
	cases := map[time.Duration]string{
		24 * time.Hour:     "24h",
		90 * time.Minute:   "90m",
		7 * 24 * time.Hour: "7d",
		36 * time.Hour:     "36h",
		30 * time.Second:   "30s",
	}
	for d, want := range cases {
		if got := windowLabel(d); got != want {
			t.Errorf("windowLabel(%v) = %q want %q", d, got, want)
		}
	}
}

func TestRenderShowsRootReadmeBannerBeforeModules(t *testing.T) {
	r := plainReport()
	r.RootReadme = Readme{
		Lines:     []string{"# Jetlog", "Travel briefings."},
		Truncated: true,
		Total:     12,
		URL:       "https://github.com/BergurHeimisson/jetlog/blob/develop/README.md",
	}
	out := render(r, Options{Lines: 2})

	if !strings.Contains(out, "═══ Jetlog") || !strings.Contains(out, "blob/develop/README.md") {
		t.Errorf("banner missing:\n%s", out)
	}
	if !strings.Contains(out, "… 10 more lines") {
		t.Errorf("banner truncation notice missing:\n%s", out)
	}
	if strings.Index(out, "═══ Jetlog") > strings.Index(out, "● pipeline") {
		t.Errorf("banner should come before the module list:\n%s", out)
	}
}

func TestRenderOmitsRootBannerWhenAbsent(t *testing.T) {
	out := render(plainReport(), Options{Lines: 2})
	if strings.Contains(out, "README.md\n  ═══") && !strings.Contains(out, "pipeline") {
		t.Errorf("unexpected banner:\n%s", out)
	}
	if strings.Count(out, "●") != 1 {
		t.Errorf("expected exactly one module bullet:\n%s", out)
	}
}
