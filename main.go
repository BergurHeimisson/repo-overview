package main

import (
	"flag"
	"fmt"
	"os"

	"golang.org/x/term"
)

func main() {
	lines := flag.Int("lines", 30, "README lines to show per module")
	long := flag.Bool("long", false, "also show the last committer and commit subject")
	since := flag.String("since", "24h", "churn window, e.g. 24h, 90m, 7d")
	branch := flag.String("branch", "", "report on this branch instead of develop/main/master")
	noFetch := flag.Bool("no-fetch", false, "skip git fetch and use local refs")
	noColor := flag.Bool("no-color", false, "disable ANSI colour")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: repo-overview [flags] [path]\n\n"+
			"Per-module overview of a git repository, read from develop\n"+
			"(falling back to main, then master) without touching your working tree.\n\nflags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	path := "."
	if flag.NArg() > 0 {
		path = flag.Arg(0)
	}

	window, err := parseWindow(*since)
	if err != nil {
		fail(err)
	}
	if *lines < 0 {
		fail(fmt.Errorf("--lines must not be negative"))
	}

	opts := Options{
		Lines:  *lines,
		Long:   *long,
		Window: window,
		Branch: *branch,
		Fetch:  !*noFetch,
		Color:  colorEnabled(*noColor),
	}

	rep, err := buildReport(path, opts)
	if err != nil {
		fail(err)
	}
	fmt.Print(render(rep, opts))
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
