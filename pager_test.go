package main

import (
	"strings"
	"testing"
)

type keyScript struct {
	keys  []byte
	calls int
}

func (k *keyScript) next() (byte, error) {
	if k.calls >= len(k.keys) {
		k.calls++
		return ' ', nil
	}
	b := k.keys[k.calls]
	k.calls++
	return b, nil
}

func TestPaginateWritesShortOutputWithoutPausing(t *testing.T) {
	var out strings.Builder
	keys := &keyScript{}
	text := "one\ntwo\nthree\n"

	if err := paginate(text, 24, &out, keys.next); err != nil {
		t.Fatal(err)
	}
	if out.String() != text {
		t.Errorf("got %q want %q", out.String(), text)
	}
	if keys.calls != 0 {
		t.Errorf("waited for %d keys, expected none", keys.calls)
	}
}

func TestPaginatePausesAfterAScreenful(t *testing.T) {
	var out strings.Builder
	keys := &keyScript{keys: []byte{' '}}
	lines := []string{"1", "2", "3", "4", "5", "6"}
	text := strings.Join(lines, "\n") + "\n"

	if err := paginate(text, 5, &out, keys.next); err != nil {
		t.Fatal(err)
	}
	if keys.calls != 1 {
		t.Errorf("waited for %d keys want 1", keys.calls)
	}
	got := out.String()
	if !strings.Contains(got, "more") {
		t.Errorf("expected a more prompt in %q", got)
	}
	for _, l := range lines {
		if !strings.Contains(got, l+"\n") {
			t.Errorf("line %q missing from output %q", l, got)
		}
	}
}

func TestPaginateStopsOnQ(t *testing.T) {
	var out strings.Builder
	keys := &keyScript{keys: []byte{'q'}}
	text := "1\n2\n3\n4\n5\n6\n"

	if err := paginate(text, 5, &out, keys.next); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "6") {
		t.Errorf("q should stop paging, got %q", out.String())
	}
}

func TestPaginateAdvancesOneLineOnEnter(t *testing.T) {
	var out strings.Builder
	keys := &keyScript{keys: []byte{'\r', 'q'}}
	text := "1\n2\n3\n4\n5\n6\n7\n8\n"

	if err := paginate(text, 5, &out, keys.next); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "5\n") {
		t.Errorf("enter should reveal one more line, got %q", got)
	}
	if strings.Contains(got, "6\n") {
		t.Errorf("enter should reveal only one line, got %q", got)
	}
}
