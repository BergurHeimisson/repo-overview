package main

import "strings"

// parseRemote turns a git remote URL into the https base of a GitHub repo.
// Non-GitHub remotes report false so callers fall back to plain paths.
func parseRemote(url string) (string, bool) {
	u := strings.TrimSpace(url)
	u = strings.TrimSuffix(u, ".git")

	switch {
	case strings.HasPrefix(u, "git@github.com:"):
		u = strings.TrimPrefix(u, "git@github.com:")
	case strings.HasPrefix(u, "ssh://git@github.com/"):
		u = strings.TrimPrefix(u, "ssh://git@github.com/")
	case strings.HasPrefix(u, "https://github.com/"):
		u = strings.TrimPrefix(u, "https://github.com/")
	case strings.HasPrefix(u, "http://github.com/"):
		u = strings.TrimPrefix(u, "http://github.com/")
	default:
		return "", false
	}

	u = strings.Trim(u, "/")
	if strings.Count(u, "/") != 1 || u == "" {
		return "", false
	}
	return "https://github.com/" + u, true
}

func blobURL(base, branch, path string) string {
	return base + "/blob/" + branch + "/" + path
}
