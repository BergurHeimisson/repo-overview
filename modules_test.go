package main

import (
	"reflect"
	"testing"
)

func names(mods []Module) []string {
	out := make([]string, len(mods))
	for i, m := range mods {
		out[i] = m.Dir
	}
	return out
}

func TestDiscoverUsesBuildManifests(t *testing.T) {
	tree := []string{
		"README.md",
		"api/pom.xml",
		"api/src/Main.java",
		"web/package.json",
		"web/src/index.ts",
		"docs/notes.md",
	}
	got := names(discoverModules(tree))
	want := []string{"api", "web"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestDiscoverIncludesRepoRootWhenItHasAManifest(t *testing.T) {
	tree := []string{"go.mod", "main.go", "tools/go.mod", "tools/t.go"}
	got := names(discoverModules(tree))
	want := []string{".", "tools"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestDiscoverFallsBackToTopLevelDirs(t *testing.T) {
	tree := []string{
		"README.md",
		"knowledge/prds/README.md",
		"knowledge/prds/one.md",
		"scripts/run.sh",
	}
	got := names(discoverModules(tree))
	want := []string{"knowledge", "scripts"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestDiscoverMatchesCsprojByExtension(t *testing.T) {
	tree := []string{"Service/Service.csproj", "Service/Program.cs"}
	got := names(discoverModules(tree))
	want := []string{"Service"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestModuleReadmeFoundCaseInsensitively(t *testing.T) {
	tree := []string{"api/pom.xml", "api/Readme.md", "web/package.json"}
	mods := discoverModules(tree)
	if mods[0].Readme != "api/Readme.md" {
		t.Errorf("api readme = %q, want api/Readme.md", mods[0].Readme)
	}
	if mods[1].Readme != "" {
		t.Errorf("web readme = %q, want empty", mods[1].Readme)
	}
}

func TestNestedModulesAreExcludedFromTheirParent(t *testing.T) {
	tree := []string{"go.mod", "tools/go.mod", "sub/deep/package.json"}
	mods := discoverModules(tree)
	if mods[0].Dir != "." {
		t.Fatalf("first module = %q", mods[0].Dir)
	}
	want := []string{"sub/deep", "tools"}
	if !reflect.DeepEqual(mods[0].Nested, want) {
		t.Errorf("root nested = %v want %v", mods[0].Nested, want)
	}
	if len(mods[1].Nested) != 0 || len(mods[2].Nested) != 0 {
		t.Error("leaf modules should have no nested modules")
	}
}

func TestDiscoverIgnoresVendoredDirectories(t *testing.T) {
	tree := []string{
		"web/package.json",
		"web/node_modules/left-pad/package.json",
		"api/pom.xml",
		"api/target/classes/pom.xml",
		"py/pyproject.toml",
		"py/.venv/lib/thing/pyproject.toml",
		"go/go.mod",
		"go/vendor/example.com/dep/go.mod",
	}
	got := names(discoverModules(tree))
	want := []string{"api", "go", "py", "web"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}
