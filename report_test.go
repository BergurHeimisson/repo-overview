package main

import (
	"strings"
	"testing"
	"time"
)

func busyRepo(t *testing.T) *fixture {
	f := newFixture(t)
	f.write("api/pom.xml", "<project/>")
	f.write("api/README.md", "# API\nBooking service.\nMore.\n")
	f.write("api/Main.java", "a\nb\n")
	f.write("web/package.json", "{}")
	f.write("web/index.ts", "x\n")
	f.git("add", "-A")
	f.commitAt("scaffold", 72*time.Hour, "Old Author")

	f.write("api/Main.java", "a\nb\nc\nd\n")
	f.git("add", "-A")
	f.commitAt("extend the api", 2*time.Hour, "Bergur Heimisson")
	f.git("branch", "develop")
	return f
}

func TestBuildReportRanksActiveModulesFirst(t *testing.T) {
	f := busyRepo(t)
	rep, err := buildReport(f.dir, Options{Lines: 30, Window: 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Ref.Name != "develop" {
		t.Errorf("ref = %q want develop", rep.Ref.Name)
	}
	if len(rep.Modules) != 2 {
		t.Fatalf("got %d modules want 2", len(rep.Modules))
	}
	if rep.Modules[0].Module.Dir != "api" {
		t.Errorf("most active = %q want api", rep.Modules[0].Module.Dir)
	}
	if rep.Modules[0].Churn.Added != 2 {
		t.Errorf("api Added = %d want 2", rep.Modules[0].Churn.Added)
	}
	if rep.Modules[1].Churn.Total() != 0 {
		t.Errorf("web should be idle, got %+v", rep.Modules[1].Churn)
	}
}

func TestBuildReportAttachesReadmeContent(t *testing.T) {
	f := busyRepo(t)
	rep, err := buildReport(f.dir, Options{Lines: 2, Window: 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	api := rep.Modules[0]
	if len(api.ReadmeLines) != 2 || api.ReadmeLines[0] != "# API" {
		t.Errorf("readme lines = %v", api.ReadmeLines)
	}
	if !api.Truncated || api.ReadmeTotal != 3 {
		t.Errorf("truncated=%v total=%d want true/3", api.Truncated, api.ReadmeTotal)
	}
}

func TestBuildReportFallsBackToRelativePathWithoutGitHubRemote(t *testing.T) {
	f := busyRepo(t)
	rep, err := buildReport(f.dir, Options{Lines: 30, Window: 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if got := rep.Modules[0].URL; got != "api/README.md" {
		t.Errorf("URL = %q want the plain repo-relative path", got)
	}
}

func TestBuildReportUsesGitHubURLWhenOriginIsGitHub(t *testing.T) {
	f := busyRepo(t)
	f.git("remote", "add", "origin", "git@github.com:BergurHeimisson/jetlog.git")

	rep, err := buildReport(f.dir, Options{Lines: 30, Window: 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	want := "https://github.com/BergurHeimisson/jetlog/blob/develop/api/README.md"
	if rep.Modules[0].URL != want {
		t.Errorf("URL = %q want %q", rep.Modules[0].URL, want)
	}
	if rep.Repo != "BergurHeimisson/jetlog" {
		t.Errorf("Repo = %q", rep.Repo)
	}
}

func TestBuildReportSkipsLastCommitWhenShort(t *testing.T) {
	f := busyRepo(t)
	short, _ := buildReport(f.dir, Options{Lines: 30, Window: 24 * time.Hour})
	if short.Modules[0].Last.Subject != "" {
		t.Error("last commit gathered with --short")
	}
	full, err := buildReport(f.dir, Options{Lines: 30, Window: 24 * time.Hour, ShowCommit: true})
	if err != nil {
		t.Fatal(err)
	}
	if full.Modules[0].Last.Author != "Bergur Heimisson" {
		t.Errorf("author = %q", full.Modules[0].Last.Author)
	}
	if !strings.Contains(full.Modules[0].Last.Subject, "extend the api") {
		t.Errorf("subject = %q", full.Modules[0].Last.Subject)
	}
}

func TestBuildReportWarnsWhenFetchFails(t *testing.T) {
	f := busyRepo(t)
	f.git("remote", "add", "origin", "file:///nonexistent-repo-path")

	rep, err := buildReport(f.dir, Options{Lines: 30, Window: 24 * time.Hour, Fetch: true})
	if err != nil {
		t.Fatalf("a failed fetch must not abort the run: %v", err)
	}
	if len(rep.Warnings) == 0 {
		t.Error("expected a warning about the failed fetch")
	}
	if len(rep.Modules) != 2 {
		t.Error("report should still be produced from local refs")
	}
}

func TestBuildReportShowsRootReadmeWhenRootIsNotAModule(t *testing.T) {
	f := busyRepo(t)
	f.write("README.md", "# Jetlog\nThe whole thing.\nThird line.\n")
	f.git("add", "-A")
	f.commitAt("add root readme", time.Hour, "Bergur Heimisson")
	f.git("branch", "-f", "develop", "HEAD")

	rep, err := buildReport(f.dir, Options{Lines: 2, Window: 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.RootReadme.Lines) != 2 || rep.RootReadme.Lines[0] != "# Jetlog" {
		t.Errorf("root readme lines = %v", rep.RootReadme.Lines)
	}
	if !rep.RootReadme.Truncated || rep.RootReadme.Total != 3 {
		t.Errorf("truncated=%v total=%d want true/3", rep.RootReadme.Truncated, rep.RootReadme.Total)
	}
	if rep.RootReadme.URL != "README.md" {
		t.Errorf("URL = %q want README.md", rep.RootReadme.URL)
	}
	for _, m := range rep.Modules {
		if m.Module.Dir == "." {
			t.Error("root should not also appear as a module")
		}
	}
}

func TestBuildReportUsesGitHubURLForRootReadme(t *testing.T) {
	f := busyRepo(t)
	f.write("README.md", "# Jetlog\n")
	f.git("add", "-A")
	f.commitAt("add root readme", time.Hour, "Bergur Heimisson")
	f.git("branch", "-f", "develop", "HEAD")
	f.git("remote", "add", "origin", "git@github.com:BergurHeimisson/jetlog.git")

	rep, err := buildReport(f.dir, Options{Lines: 30, Window: 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	want := "https://github.com/BergurHeimisson/jetlog/blob/develop/README.md"
	if rep.RootReadme.URL != want {
		t.Errorf("URL = %q want %q", rep.RootReadme.URL, want)
	}
}

func TestBuildReportSkipsRootBannerWhenRootIsAModule(t *testing.T) {
	f := newFixture(t)
	f.write("go.mod", "module x\n")
	f.write("README.md", "# Root module\n")
	f.write("tools/go.mod", "module t\n")
	f.git("add", "-A")
	f.commitAt("init", time.Hour, "Test Person")

	rep, err := buildReport(f.dir, Options{Lines: 30, Window: 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.RootReadme.Lines) != 0 {
		t.Error("root is already a module; the banner would duplicate it")
	}
}

func TestBuildReportHasNoRootBannerWithoutARootReadme(t *testing.T) {
	f := busyRepo(t)
	rep, err := buildReport(f.dir, Options{Lines: 30, Window: 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.RootReadme.Lines) != 0 || rep.RootReadme.URL != "" {
		t.Errorf("unexpected root banner: %+v", rep.RootReadme)
	}
}

func TestBuildReportSkipsStaleModules(t *testing.T) {
	f := busyRepo(t)
	rep, err := buildReport(f.dir, Options{Lines: 1, Window: 24 * time.Hour, SkipStale: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Modules) != 1 {
		t.Fatalf("got %d modules want 1", len(rep.Modules))
	}
	if rep.Modules[0].Module.Dir != "api" {
		t.Errorf("kept %q want api", rep.Modules[0].Module.Dir)
	}
	if rep.Skipped != 1 {
		t.Errorf("Skipped = %d want 1", rep.Skipped)
	}
}
