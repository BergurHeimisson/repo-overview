package main

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Ref is the branch this run reports on, with the revision used to read it.
// Rev is a remote-tracking ref when one exists, so nothing depends on the
// state of the working tree.
type Ref struct {
	Name string
	Rev  string
}

// Commit is the most recent change touching a module.
type Commit struct {
	Author  string
	Subject string
	When    time.Time
}

func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return string(out), nil
}

func resolveRoot(dir string) (string, error) {
	out, err := git(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("%s is not inside a git repository", dir)
	}
	return strings.TrimSpace(out), nil
}

func revExists(dir, rev string) bool {
	_, err := git(dir, "rev-parse", "--verify", "--quiet", rev+"^{commit}")
	return err == nil
}

// pickRef prefers develop, then main, then master, preferring the
// remote-tracking copy of each over the local branch.
func pickRef(dir, explicit string) (Ref, error) {
	names := []string{"develop", "main", "master"}
	if explicit != "" {
		names = []string{explicit}
	}
	for _, name := range names {
		for _, rev := range []string{"origin/" + name, name} {
			if revExists(dir, rev) {
				return Ref{Name: name, Rev: rev}, nil
			}
		}
	}
	if explicit != "" {
		return Ref{}, fmt.Errorf("branch %q not found in this repository", explicit)
	}
	return Ref{}, errors.New("none of develop, main or master exist in this repository")
}

func fetch(dir string) error {
	_, err := git(dir, "fetch", "--quiet", "--prune")
	return err
}

func remoteBase(dir string) (string, bool) {
	out, err := git(dir, "remote", "get-url", "origin")
	if err != nil {
		return "", false
	}
	return parseRemote(out)
}

func listTree(dir, rev string) ([]string, error) {
	out, err := git(dir, "ls-tree", "-r", "--name-only", rev)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, p := range strings.Split(out, "\n") {
		if p != "" {
			paths = append(paths, p)
		}
	}
	return paths, nil
}

// readLines returns at most limit lines of a file as committed on rev, along
// with whether it was cut short and how many lines the file actually has.
func readLines(dir, rev, path string, limit int) ([]string, bool, int, error) {
	out, err := git(dir, "show", rev+":"+path)
	if err != nil {
		return nil, false, 0, err
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	total := len(lines)
	if total > limit {
		return lines[:limit], true, total, nil
	}
	return lines, false, total, nil
}

// pathspec limits a log to a module's own files, handing nested modules to
// the modules that own them.
func pathspec(m Module) []string {
	spec := []string{m.Dir}
	if m.Dir == "." {
		spec = []string{"."}
	}
	for _, n := range m.Nested {
		spec = append(spec, ":(exclude)"+n+"/**", ":(exclude)"+n)
	}
	return spec
}

func moduleChurn(dir, rev string, m Module, window time.Duration) (Churn, error) {
	since := time.Now().Add(-window).Format(time.RFC3339)
	args := []string{"log", rev, "--since=" + since, "--format=%x00%H", "--numstat", "--no-renames", "--"}
	out, err := git(dir, append(args, pathspec(m)...)...)
	if err != nil {
		return Churn{}, err
	}
	return parseChurn(out), nil
}

func lastCommit(dir, rev string, m Module) (Commit, error) {
	args := []string{"log", rev, "-1", "--format=%an%x00%s%x00%ct", "--"}
	out, err := git(dir, append(args, pathspec(m)...)...)
	if err != nil {
		return Commit{}, err
	}
	out = strings.TrimRight(out, "\n")
	parts := strings.Split(out, "\x00")
	if len(parts) != 3 {
		return Commit{}, nil
	}
	c := Commit{Author: parts[0], Subject: parts[1]}
	if secs, err := strconv.ParseInt(parts[2], 10, 64); err == nil {
		c.When = time.Unix(secs, 0)
	}
	return c, nil
}
