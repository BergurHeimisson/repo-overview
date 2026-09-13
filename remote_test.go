package main

import "testing"

func TestParseRemoteSSH(t *testing.T) {
	got, ok := parseRemote("git@github.com:BergurHeimisson/jetlog.git")
	if !ok {
		t.Fatal("expected ssh remote to parse")
	}
	if want := "https://github.com/BergurHeimisson/jetlog"; got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestParseRemoteHTTPS(t *testing.T) {
	got, ok := parseRemote("https://github.com/BergurHeimisson/jetlog.git")
	if !ok {
		t.Fatal("expected https remote to parse")
	}
	if want := "https://github.com/BergurHeimisson/jetlog"; got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestParseRemoteNonGitHub(t *testing.T) {
	if _, ok := parseRemote("git@gitlab.internal:team/thing.git"); ok {
		t.Error("non-github remote should not produce a blob base")
	}
}

func TestBlobURL(t *testing.T) {
	got := blobURL("https://github.com/BergurHeimisson/jetlog", "develop", "knowledge/prds/README.md")
	want := "https://github.com/BergurHeimisson/jetlog/blob/develop/knowledge/prds/README.md"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}
