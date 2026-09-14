package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

func main() {
	opts, path, err := parseFlags(os.Args[1:], os.Stderr)
	if err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		fail(err)
	}

	rep, err := buildReport(path, opts)
	if err != nil {
		fail(err)
	}
	pageOut(render(rep, opts), opts.Page)
}

// parseFlags turns a command line into resolved options plus the path to
// report on, writing usage and flag errors to out.
func parseFlags(args []string, out io.Writer) (Options, string, error) {
	fs := flag.NewFlagSet("repo-overview", flag.ContinueOnError)
	fs.SetOutput(out)

	lines := fs.Int("lines", 1, "README lines to show per module")
	short := fs.Bool("short", false, "omit the last committer and commit subject")
	since := fs.String("since", "24h", "churn window, e.g. 24h, 90m, 7d")
	skipStale := fs.Bool("skip-stale", false, "hide modules with no commit inside the window")
	branch := fs.String("branch", "", "report on this branch instead of develop/main/master")
	noFetch := fs.Bool("no-fetch", false, "skip git fetch and use local refs")
	noColor := fs.Bool("no-color", false, "disable ANSI colour")
	noPager := fs.Bool("no-pager", false, "print everything at once instead of pausing per screen")

	fs.Usage = func() {
		fmt.Fprintf(out, "usage: repo-overview [flags] [path]\n\n"+
			"Per-module overview of a git repository, read from develop\n"+
			"(falling back to main, then master) without touching your working tree.\n\nflags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return Options{}, "", err
	}

	window, err := parseWindow(*since)
	if err != nil {
		return Options{}, "", err
	}
	if *lines < 0 {
		return Options{}, "", fmt.Errorf("--lines must not be negative")
	}

	path := "."
	if fs.NArg() > 0 {
		path = fs.Arg(0)
	}

	return Options{
		Lines:      *lines,
		ShowCommit: !*short,
		Window:     window,
		Branch:     *branch,
		Fetch:      !*noFetch,
		SkipStale:  *skipStale,
		Page:       !*noPager,
		Color:      colorEnabled(*noColor),
	}, path, nil
}

func colorEnabled(disabled bool) bool {
	if disabled || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "repo-overview: "+err.Error())
	os.Exit(1)
}
