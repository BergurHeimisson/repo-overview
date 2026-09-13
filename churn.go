package main

import (
	"strconv"
	"strings"
)

// Churn is how much a module moved inside the reporting window.
type Churn struct {
	Added   int
	Deleted int
	Commits int
}

func (c Churn) Total() int { return c.Added + c.Deleted }

// parseChurn reads `git log --format=%x00%H --numstat` output. Commit lines are
// prefixed with NUL so they never collide with a file path.
func parseChurn(out string) Churn {
	var c Churn
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "\x00") {
			c.Commits++
			continue
		}
		added, deleted, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		deleted, _, _ = strings.Cut(deleted, "\t")
		a, errA := strconv.Atoi(added)
		d, errD := strconv.Atoi(deleted)
		if errA != nil || errD != nil {
			continue // binary file, recorded by git as "-"
		}
		c.Added += a
		c.Deleted += d
	}
	return c
}
