package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Options are the resolved command line settings for one run.
type Options struct {
	Lines  int
	Long   bool
	Window time.Duration
	Color  bool
	Fetch  bool
	Branch string
}

// ModuleReport is everything gathered about a single module.
type ModuleReport struct {
	Module      Module
	Churn       Churn
	Last        Commit
	URL         string
	ReadmeLines []string
	Truncated   bool
	ReadmeTotal int
}

// Readme is an excerpt of a README.md as committed on the reported branch.
type Readme struct {
	Lines     []string
	Truncated bool
	Total     int
	URL       string
}

// Report is the whole run, ready to render. RootReadme carries the repository's
// top-level README when the root is not itself a module.
type Report struct {
	Repo       string
	Ref        Ref
	Window     time.Duration
	RootReadme Readme
	Modules    []ModuleReport
	Warnings   []string
}

// A restrained 256-colour palette that stays legible on dark terminals.
const (
	ansiReset  = "\x1b[0m"
	ansiTitle  = "\x1b[1;38;5;153m"
	ansiModule = "\x1b[1;38;5;117m"
	ansiAdd    = "\x1b[38;5;114m"
	ansiDel    = "\x1b[38;5;174m"
	ansiMeta   = "\x1b[38;5;245m"
	ansiFaint  = "\x1b[38;5;240m"
	ansiLink   = "\x1b[38;5;109m"
	ansiWarn   = "\x1b[38;5;179m"
)

type painter bool

func (p painter) paint(code, s string) string {
	if !p {
		return s
	}
	return code + s + ansiReset
}

func parseWindow(s string) (time.Duration, error) {
	if days, ok := strings.CutSuffix(s, "d"); ok {
		n, err := strconv.ParseFloat(days, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid window %q", s)
		}
		return time.Duration(n * float64(24*time.Hour)), nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid window %q (try 24h, 90m or 7d)", s)
	}
	return d, nil
}

func sortReports(reports []ModuleReport) {
	sort.SliceStable(reports, func(i, j int) bool {
		if reports[i].Churn.Total() != reports[j].Churn.Total() {
			return reports[i].Churn.Total() > reports[j].Churn.Total()
		}
		return reports[i].Module.Dir < reports[j].Module.Dir
	})
}

func humanAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func render(r Report, o Options) string {
	p := painter(o.Color)
	var b strings.Builder

	b.WriteString(p.paint(ansiTitle, "repo-overview"))
	if r.Repo != "" {
		b.WriteString("  " + p.paint(ansiModule, r.Repo))
	}
	b.WriteString("  " + p.paint(ansiMeta, fmt.Sprintf(
		"branch %s · %d modules · churn over %s", r.Ref.Name, len(r.Modules), windowLabel(r.Window))))
	b.WriteString("\n")

	for _, w := range r.Warnings {
		b.WriteString(p.paint(ansiWarn, "! "+w) + "\n")
	}
	b.WriteString("\n")

	if len(r.RootReadme.Lines) > 0 {
		renderReadme(&b, p, r.RootReadme)
		b.WriteString("\n")
	}

	for _, m := range r.Modules {
		renderModule(&b, p, m, o)
	}
	return b.String()
}

// windowLabel prints a duration the way it was most likely typed: 24h, 90m, 7d.
func windowLabel(d time.Duration) string {
	switch {
	case d == 0:
		return "24h"
	case d >= 48*time.Hour && d%(24*time.Hour) == 0:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	case d >= time.Hour && d%time.Hour == 0:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d >= time.Minute && d%time.Minute == 0:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		return d.String()
	}
}

func renderModule(b *strings.Builder, p painter, m ModuleReport, o Options) {
	name := m.Module.Dir
	if name == "." {
		name = "(repo root)"
	}

	b.WriteString(p.paint(ansiModule, "● "+name))
	b.WriteString("  " + churnLabel(p, m.Churn) + "\n")

	if m.Module.Readme == "" {
		b.WriteString("  " + p.paint(ansiFaint, "(no README.md)") + "\n")
	}

	renderReadme(b, p, Readme{
		Lines:     m.ReadmeLines,
		Truncated: m.Truncated,
		Total:     m.ReadmeTotal,
		URL:       m.URL,
	})

	if o.Long && m.Last.Subject != "" {
		age := ""
		if !m.Last.When.IsZero() {
			age = " (" + humanAge(time.Since(m.Last.When)) + ")"
		}
		b.WriteString("  " + p.paint(ansiMeta, "last: "+m.Last.Author+" — "+m.Last.Subject+age) + "\n")
	}
	b.WriteString("\n")
}

func renderReadme(b *strings.Builder, p painter, r Readme) {
	if r.URL != "" {
		b.WriteString("  " + p.paint(ansiLink, r.URL) + "\n")
	}
	for _, line := range renderMarkdown(r.Lines, p) {
		b.WriteString("  " + line + "\n")
	}
	if r.Truncated {
		if rest := r.Total - len(r.Lines); rest > 0 {
			b.WriteString(p.paint(ansiFaint, fmt.Sprintf("  … %d more lines\n", rest)))
		} else {
			b.WriteString(p.paint(ansiFaint, "  … more lines\n"))
		}
	}
}

func churnLabel(p painter, c Churn) string {
	if c.Total() == 0 {
		return p.paint(ansiFaint, "idle")
	}
	plural := "commits"
	if c.Commits == 1 {
		plural = "commit"
	}
	return p.paint(ansiAdd, fmt.Sprintf("+%d", c.Added)) + " " +
		p.paint(ansiDel, fmt.Sprintf("−%d", c.Deleted)) + " " +
		p.paint(ansiMeta, fmt.Sprintf("(%d %s)", c.Commits, plural))
}
