package main

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// The markdown palette mirrors md-viewer's terminal renderer, so a README reads
// the same whether it is opened there or surfaced here.
const (
	mdH1     = "\x1b[1;36m"
	mdH2     = "\x1b[1;93m"
	mdH3     = "\x1b[1;35m"
	mdStrong = "\x1b[1;37m"
	mdEm     = "\x1b[0;33m"
	mdCode   = "\x1b[0;36m"
	mdLink   = "\x1b[0;34m"
	mdQuote  = "\x1b[1;32m"
	mdDone   = "\x1b[1;32m"
	mdDim    = "\x1b[0;90m"
	mdBullet = "\x1b[0;36m"
)

var (
	fenceRE   = regexp.MustCompile("^\\s*(```|~~~)")
	headingRE = regexp.MustCompile(`^(#{1,6})\s+(.*?)\s*#*$`)
	breakRE   = regexp.MustCompile(`^\s*((-\s*){3,}|(\*\s*){3,}|(_\s*){3,})$`)
	quoteRE   = regexp.MustCompile(`^\s*>\s?(.*)$`)
	bulletRE  = regexp.MustCompile(`^(\s*)[-*+]\s+(.*)$`)
	orderedRE = regexp.MustCompile(`^(\s*)(\d+)[.)]\s+(.*)$`)
	taskRE    = regexp.MustCompile(`^\[([ xX])\]\s+(.*)$`)
	delimRE   = regexp.MustCompile(`^:?-{1,}:?$`)
	ansiRE    = regexp.MustCompile(`\x1b\[[0-9;]*m`)

	inlineRE = regexp.MustCompile(strings.Join([]string{
		`!\[[^\]]*\]\([^)]*\)`,
		`\[[^\]]*\]\([^)]*\)`,
		`\*\*[^*]+\*\*`,
		`__[^_]+__`,
		`~~[^~]+~~`,
		`\*[^*]+\*`,
		`_[^_]+_`,
		`https?://[^\s<>()\[\]]+`,
	}, "|"))

	destRE = regexp.MustCompile(`^!?\[([^\]]*)\]\(\s*<?([^)\s>]*)>?[^)]*\)$`)
)

// renderMarkdown turns README source lines into styled terminal lines. It works
// line by line rather than over a parsed document because the input is an
// arbitrary excerpt: an unterminated code fence or a half-finished list must
// still render, and one source line stays one output line — tables aside, which
// gain their borders — so that the --lines budget means what it says.
func renderMarkdown(lines []string, p painter) []string {
	var out []string
	var table []string
	inFence := false

	flush := func() {
		if len(table) > 0 {
			out = append(out, renderTable(table, p)...)
			table = nil
		}
	}

	for _, line := range lines {
		if fenceRE.MatchString(line) {
			flush()
			inFence = !inFence
			continue
		}
		if inFence {
			out = append(out, p.paint(mdCode, "│ "+line))
			continue
		}
		if isTableRow(line) {
			table = append(table, line)
			continue
		}
		flush()
		out = append(out, renderBlockLine(line, p))
	}
	flush()
	return out
}

func renderBlockLine(line string, p painter) string {
	if m := headingRE.FindStringSubmatch(line); m != nil {
		text := renderInline(m[2], p)
		switch len(m[1]) {
		case 1:
			return p.paint(mdH1, "═══ ") + p.paint(mdH1, text)
		case 2:
			return p.paint(mdH2, "─── ") + p.paint(mdH2, text)
		default:
			return p.paint(mdH3, "▸ ") + p.paint(mdH3, text)
		}
	}
	if breakRE.MatchString(line) {
		return p.paint(mdDim, strings.Repeat("─", 60))
	}
	if m := quoteRE.FindStringSubmatch(line); m != nil {
		return p.paint(mdQuote, "▌ ") + renderInline(m[1], p)
	}
	if m := bulletRE.FindStringSubmatch(line); m != nil {
		indent, rest := m[1], m[2]
		if t := taskRE.FindStringSubmatch(rest); t != nil {
			box := p.paint(mdDim, "☐")
			if t[1] != " " {
				box = p.paint(mdDone, "☑")
			}
			return indent + box + " " + renderInline(t[2], p)
		}
		return indent + p.paint(mdBullet, "•") + " " + renderInline(rest, p)
	}
	if m := orderedRE.FindStringSubmatch(line); m != nil {
		return m[1] + m[2] + ". " + renderInline(m[3], p)
	}
	return renderInline(line, p)
}

// renderInline styles emphasis, code spans and links. Code spans are split out
// first so that markup inside them stays literal.
func renderInline(s string, p painter) string {
	var b strings.Builder
	for i, seg := range strings.Split(s, "`") {
		if i%2 == 1 {
			b.WriteString(p.paint(mdCode, seg))
			continue
		}
		b.WriteString(styleSpans(seg, p))
	}
	return b.String()
}

// styleSpans replaces every inline construct in one left-to-right pass, so the
// first alternative that matches at a position wins and replacements are never
// rescanned.
func styleSpans(s string, p painter) string {
	return inlineRE.ReplaceAllStringFunc(s, func(m string) string {
		switch {
		case strings.HasPrefix(m, "!["):
			d := destRE.FindStringSubmatch(m)
			return p.paint(mdH3, "🖼 ") + renderInline(d[1], p) + p.paint(mdLink, " ["+d[2]+"]")
		case strings.HasPrefix(m, "["):
			d := destRE.FindStringSubmatch(m)
			return renderInline(d[1], p) + p.paint(mdLink, " ["+d[2]+"]")
		case strings.HasPrefix(m, "**"):
			return p.paint(mdStrong, strings.Trim(m, "*"))
		case strings.HasPrefix(m, "__"):
			return p.paint(mdStrong, strings.Trim(m, "_"))
		case strings.HasPrefix(m, "~~"):
			return p.paint(mdDim, strings.Trim(m, "~"))
		case strings.HasPrefix(m, "*"):
			return p.paint(mdEm, strings.Trim(m, "*"))
		case strings.HasPrefix(m, "_"):
			return p.paint(mdEm, strings.Trim(m, "_"))
		default:
			return p.paint(mdLink, m)
		}
	})
}

func isTableRow(line string) bool {
	t := strings.TrimSpace(line)
	return strings.HasPrefix(t, "|") && strings.Count(t, "|") >= 2
}

func tableCells(line string) []string {
	t := strings.TrimSpace(line)
	t = strings.TrimSuffix(strings.TrimPrefix(t, "|"), "|")
	cells := strings.Split(t, "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

func isDelimiterRow(cells []string) bool {
	for _, c := range cells {
		if !delimRE.MatchString(strings.ReplaceAll(c, " ", "")) {
			return false
		}
	}
	return len(cells) > 0
}

// renderTable draws a GFM table as a box-drawn grid with aligned columns.
func renderTable(src []string, p painter) []string {
	var rows [][]string
	header := 0
	for _, line := range src {
		cells := tableCells(line)
		if isDelimiterRow(cells) {
			header = len(rows)
			continue
		}
		styled := make([]string, len(cells))
		for i, c := range cells {
			styled[i] = renderInline(c, p)
		}
		rows = append(rows, styled)
	}
	if len(rows) == 0 {
		return nil
	}

	columns := 0
	for _, r := range rows {
		columns = max(columns, len(r))
	}
	widths := make([]int, columns)
	for _, r := range rows {
		for c, cell := range r {
			widths[c] = max(widths[c], visibleLength(cell))
		}
	}

	var out []string
	out = append(out, tableBorder(widths, p, "┌", "┬", "┐"))
	for i, r := range rows {
		out = append(out, tableRow(r, widths, p, i < header))
		if i == header-1 {
			out = append(out, tableBorder(widths, p, "├", "┼", "┤"))
		}
	}
	return append(out, tableBorder(widths, p, "└", "┴", "┘"))
}

func tableBorder(widths []int, p painter, left, mid, right string) string {
	var b strings.Builder
	b.WriteString(left)
	for i, w := range widths {
		b.WriteString(strings.Repeat("─", w+2))
		if i == len(widths)-1 {
			b.WriteString(right)
		} else {
			b.WriteString(mid)
		}
	}
	return p.paint(mdDim, b.String())
}

func tableRow(cells []string, widths []int, p painter, header bool) string {
	var b strings.Builder
	for i, w := range widths {
		content := ""
		if i < len(cells) {
			content = cells[i]
		}
		if header {
			content = p.paint(mdStrong, content)
		}
		b.WriteString(p.paint(mdDim, "│") + " " + content)
		b.WriteString(strings.Repeat(" ", w-visibleLength(content)+1))
	}
	b.WriteString(p.paint(mdDim, "│"))
	return b.String()
}

// visibleLength counts printable characters, ignoring ANSI escape sequences.
func visibleLength(s string) int {
	return utf8.RuneCountInString(ansiRE.ReplaceAllString(s, ""))
}
