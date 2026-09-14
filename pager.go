package main

import (
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

const morePrompt = "-- more (space, enter, q) --"

// paginate writes text a screenful at a time, asking key for what to do at
// each pause: space for the next page, enter for one more line, q to stop.
func paginate(text string, height int, w io.Writer, key func() (byte, error)) error {
	lines := strings.SplitAfter(text, "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	page := height - 1
	if page < 1 {
		_, err := io.WriteString(w, text)
		return err
	}

	for i := 0; i < len(lines); {
		for n := 0; n < page && i < len(lines); n++ {
			if _, err := io.WriteString(w, lines[i]); err != nil {
				return err
			}
			i++
		}
		if i >= len(lines) {
			return nil
		}
		if _, err := io.WriteString(w, morePrompt); err != nil {
			return err
		}
		k, err := key()
		if _, e := io.WriteString(w, "\r\x1b[K"); e != nil {
			return e
		}
		if err != nil {
			return err
		}
		switch k {
		case 'q', 'Q', 3, 4: // q, ctrl-c, ctrl-d
			return nil
		case '\r', '\n':
			page = 1
		default:
			page = height - 1
		}
	}
	return nil
}

// pageOut sends the report through the built-in pager when stdout is a
// terminal we can read keys from, and prints it plainly otherwise.
func pageOut(text string, enabled bool) {
	height, key, cleanup, ok := terminalPager()
	if !enabled || !ok {
		os.Stdout.WriteString(text)
		return
	}
	defer cleanup()
	if err := paginate(text, height, os.Stdout, key); err != nil {
		os.Stdout.WriteString("\n")
	}
}

// terminalPager wires paginate to the controlling terminal, reading single
// keypresses by dropping into raw mode only while waiting for one.
func terminalPager() (int, func() (byte, error), func(), bool) {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return 0, nil, nil, false
	}
	_, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || height < 3 {
		return 0, nil, nil, false
	}
	tty, err := os.Open("/dev/tty")
	if err != nil {
		return 0, nil, nil, false
	}
	fd := int(tty.Fd())
	key := func() (byte, error) {
		state, err := term.MakeRaw(fd)
		if err != nil {
			return 'q', err
		}
		defer term.Restore(fd, state)
		var buf [1]byte
		if _, err := tty.Read(buf[:]); err != nil {
			return 'q', err
		}
		return buf[0], nil
	}
	return height, key, func() { tty.Close() }, true
}
