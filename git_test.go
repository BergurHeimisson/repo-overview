package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// fixture builds a throwaway repo and returns its path.
type fixture struct {
	t   *testing.T
	dir string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{t: t, dir: t.TempDir()}
	f.git("init", "-q", "-b", "main")
	f.git("config", "user.email", "test@example.com")
	f.git("config", "user.name", "Test Person")
	return f
}

func (f *fixture) git(args ...string) string {
	f.t.Helper()
	cmd := exec.Command("git", append([]string{"-C", f.dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	out, err := cmd.CombinedOutput()
	if err != nil {
		f.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func (f *fixture) write(rel, content string) {
	f.t.Helper()
	p := filepath.Join(f.dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

// commitAt commits everything staged with a backdated timestamp.
func (f *fixture) commitAt(msg string, age time.Duration, author string) {
	f.t.Helper()
	when := time.Now().Add(-age).Format(time.RFC3339)
	cmd := exec.Command("git", "-C", f.dir, "commit", "-q", "-m", msg,
		"--author", author+" <"+author+"@example.com>", "--date", when)
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_COMMITTER_DATE="+when,
		"GIT_COMMITTER_NAME=Test Person", "GIT_COMMITTER_EMAIL=test@example.com")
	if out, err := cmd.CombinedOutput(); err != nil {
		f.t.Fatalf("commit: %v\n%s", err, out)
	}
}

func TestResolveRootFindsRepoFromSubdirectory(t *testing.T) {
	f := newFixture(t)
	f.write("api/pom.xml", "<project/>")
	f.git("add", "-A")
	f.commitAt("init", time.Hour, "Test Person")

	root, err := resolveRoot(filepath.Join(f.dir, "api"))
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.EvalSymlinks(f.dir)
	if got, _ := filepath.EvalSymlinks(root); got != want {
		t.Errorf("root = %q want %q", got, want)
	}
}

func TestResolveRootRejectsNonRepo(t *testing.T) {
	if _, err := resolveRoot(t.TempDir()); err == nil {
		t.Error("expected an error outside a git repository")
	}
}

func TestPickRefPrefersDevelopOverMain(t *testing.T) {
	f := newFixture(t)
	f.write("go.mod", "module x\n")
	f.git("add", "-A")
	f.commitAt("init", time.Hour, "Test Person")
	f.git("branch", "develop")

	ref, err := pickRef(f.dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if ref.Name != "develop" {
		t.Errorf("branch = %q want develop", ref.Name)
	}
}

func TestPickRefFallsBackToMainWhenNoDevelop(t *testing.T) {
	f := newFixture(t)
	f.write("go.mod", "module x\n")
	f.git("add", "-A")
	f.commitAt("init", time.Hour, "Test Person")

	ref, err := pickRef(f.dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if ref.Name != "main" {
		t.Errorf("branch = %q want main", ref.Name)
	}
}

func TestPickRefHonoursExplicitBranch(t *testing.T) {
	f := newFixture(t)
	f.write("go.mod", "module x\n")
	f.git("add", "-A")
	f.commitAt("init", time.Hour, "Test Person")
	f.git("branch", "release")

	ref, err := pickRef(f.dir, "release")
	if err != nil {
		t.Fatal(err)
	}
	if ref.Name != "release" {
		t.Errorf("branch = %q want release", ref.Name)
	}
}

func TestPickRefErrorsOnUnknownBranch(t *testing.T) {
	f := newFixture(t)
	f.write("go.mod", "module x\n")
	f.git("add", "-A")
	f.commitAt("init", time.Hour, "Test Person")

	if _, err := pickRef(f.dir, "nope"); err == nil {
		t.Error("expected an error for a branch that does not exist")
	}
}

func TestListTreeReadsCommittedPathsOnly(t *testing.T) {
	f := newFixture(t)
	f.write("api/pom.xml", "<project/>")
	f.git("add", "-A")
	f.commitAt("init", time.Hour, "Test Person")
	f.write("untracked.txt", "scratch")

	tree, err := listTree(f.dir, "main")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range tree {
		if p == "untracked.txt" {
			t.Fatal("uncommitted file leaked into the tree listing")
		}
	}
	if len(tree) != 1 || tree[0] != "api/pom.xml" {
		t.Errorf("tree = %v want [api/pom.xml]", tree)
	}
}

func TestReadLinesTruncatesToLimit(t *testing.T) {
	f := newFixture(t)
	f.write("api/pom.xml", "<project/>")
	f.write("api/README.md", "one\ntwo\nthree\nfour\n")
	f.git("add", "-A")
	f.commitAt("init", time.Hour, "Test Person")

	lines, truncated, _, err := readLines(f.dir, "main", "api/README.md", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || lines[0] != "one" || lines[1] != "two" {
		t.Errorf("lines = %v want [one two]", lines)
	}
	if !truncated {
		t.Error("expected truncated = true")
	}
}

func TestChurnCountsOnlyCommitsInsideTheWindow(t *testing.T) {
	f := newFixture(t)
	f.write("api/pom.xml", "<project/>")
	f.write("api/a.txt", "1\n2\n3\n")
	f.git("add", "-A")
	f.commitAt("old work", 72*time.Hour, "Old Author")

	f.write("api/a.txt", "1\n2\n3\n4\n5\n")
	f.git("add", "-A")
	f.commitAt("recent work", 2*time.Hour, "Recent Author")

	c, err := moduleChurn(f.dir, "main", Module{Dir: "api"}, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if c.Commits != 1 {
		t.Errorf("Commits = %d want 1", c.Commits)
	}
	if c.Added != 2 || c.Deleted != 0 {
		t.Errorf("churn = %+v want +2 -0", c)
	}
}

func TestChurnExcludesNestedModules(t *testing.T) {
	f := newFixture(t)
	f.write("go.mod", "module x\n")
	f.write("root.txt", "a\n")
	f.write("tools/go.mod", "module t\n")
	f.write("tools/big.txt", "a\nb\nc\nd\ne\n")
	f.git("add", "-A")
	f.commitAt("init", time.Hour, "Test Person")

	root := Module{Dir: ".", Nested: []string{"tools"}}
	c, err := moduleChurn(f.dir, "main", root, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if c.Added != 2 {
		t.Errorf("root Added = %d want 2 (go.mod + root.txt, not tools/)", c.Added)
	}
}

func TestLastCommitReportsAuthorAndSubject(t *testing.T) {
	f := newFixture(t)
	f.write("api/pom.xml", "<project/>")
	f.git("add", "-A")
	f.commitAt("add the api module", time.Hour, "Bergur Heimisson")

	last, err := lastCommit(f.dir, "main", Module{Dir: "api"})
	if err != nil {
		t.Fatal(err)
	}
	if last.Author != "Bergur Heimisson" {
		t.Errorf("Author = %q", last.Author)
	}
	if last.Subject != "add the api module" {
		t.Errorf("Subject = %q", last.Subject)
	}
	if last.When.IsZero() {
		t.Error("When should be set")
	}
}

func TestLastCommitOnUntouchedModuleIsEmpty(t *testing.T) {
	f := newFixture(t)
	f.write("api/pom.xml", "<project/>")
	f.git("add", "-A")
	f.commitAt("init", time.Hour, "Test Person")

	last, err := lastCommit(f.dir, "main", Module{Dir: "ghost"})
	if err != nil {
		t.Fatal(err)
	}
	if last.Subject != "" {
		t.Errorf("expected empty commit info, got %+v", last)
	}
}

func TestReadLinesReportsTotalLineCount(t *testing.T) {
	f := newFixture(t)
	f.write("api/pom.xml", "<project/>")
	f.write("api/README.md", "one\ntwo\nthree\nfour\n")
	f.git("add", "-A")
	f.commitAt("init", time.Hour, "Test Person")

	_, _, total, err := readLines(f.dir, "main", "api/README.md", 2)
	if err != nil {
		t.Fatal(err)
	}
	if total != 4 {
		t.Errorf("total = %d want 4", total)
	}
}
