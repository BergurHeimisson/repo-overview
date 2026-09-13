package main

import (
	"path"
	"sort"
	"strings"
)

// Module is one unit of the repository: a directory holding a build manifest,
// or a top-level directory when the repo has no manifests at all.
type Module struct {
	Dir    string
	Readme string
	Nested []string
}

var manifestNames = map[string]bool{
	"pom.xml":          true,
	"go.mod":           true,
	"package.json":     true,
	"build.gradle":     true,
	"build.gradle.kts": true,
	"settings.gradle":  true,
	"pyproject.toml":   true,
	"setup.py":         true,
	"Cargo.toml":       true,
	"CMakeLists.txt":   true,
	"composer.json":    true,
	"Gemfile":          true,
}

func isManifest(base string) bool {
	return manifestNames[base] || strings.HasSuffix(base, ".csproj")
}

// Dependency caches and build output carry manifests of their own; they are
// not modules of this repository.
var vendorDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	"target":       true,
	"build":        true,
	"dist":         true,
	"out":          true,
	".venv":        true,
	"venv":         true,
	".git":         true,
	".gradle":      true,
	"__pycache__":  true,
}

func isVendored(p string) bool {
	for _, seg := range strings.Split(path.Dir(p), "/") {
		if vendorDirs[seg] {
			return true
		}
	}
	return false
}

func discoverModules(tree []string) []Module {
	dirs := manifestDirs(tree)
	if len(dirs) == 0 {
		dirs = topLevelDirs(tree)
	}
	sort.Strings(dirs)

	mods := make([]Module, len(dirs))
	for i, d := range dirs {
		mods[i] = Module{Dir: d, Nested: nestedUnder(d, dirs)}
	}
	attachReadmes(mods, tree)
	return mods
}

func manifestDirs(tree []string) []string {
	seen := map[string]bool{}
	var dirs []string
	for _, p := range tree {
		if !isManifest(path.Base(p)) || isVendored(p) {
			continue
		}
		d := path.Dir(p)
		if !seen[d] {
			seen[d] = true
			dirs = append(dirs, d)
		}
	}
	return dirs
}

func topLevelDirs(tree []string) []string {
	seen := map[string]bool{}
	var dirs []string
	for _, p := range tree {
		d, _, ok := strings.Cut(p, "/")
		if !ok || seen[d] || vendorDirs[d] {
			continue
		}
		seen[d] = true
		dirs = append(dirs, d)
	}
	return dirs
}

func nestedUnder(dir string, all []string) []string {
	var nested []string
	for _, other := range all {
		if other != dir && contains(dir, other) {
			nested = append(nested, other)
		}
	}
	sort.Strings(nested)
	return nested
}

func contains(parent, child string) bool {
	if parent == "." {
		return true
	}
	return strings.HasPrefix(child, parent+"/")
}

func attachReadmes(mods []Module, tree []string) {
	for _, p := range tree {
		if !strings.EqualFold(path.Base(p), "README.md") {
			continue
		}
		d := path.Dir(p)
		for i := range mods {
			if mods[i].Dir == d && mods[i].Readme == "" {
				mods[i].Readme = p
			}
		}
	}
}
