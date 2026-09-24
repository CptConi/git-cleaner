package main

import (
	"fmt"
	"io"
	"strings"
)

// The startup banner, printed when git-cleaner runs in an interactive
// terminal (or when FORCE_COLOR is set). It only uses characters that every
// console font has, Windows included: block elements, box drawing, "●", "×".

// logoGlyphs spells "git-cleaner" with half-block characters: each glyph is
// two lines of text, i.e. four rows of "pixels".
var logoGlyphs = [...][2]string{
	{"█▀▀", "█▄█"},   // g
	{"█", "█"},       // i
	{"▀█▀", " █ "},   // t
	{"  ", "▀▀"},     // -
	{"█▀▀", "█▄▄"},   // c
	{"█  ", "█▄▄"},   // l
	{"█▀▀", "██▄"},   // e
	{"▄▀█", "█▀█"},   // a
	{"█▄ █", "█ ▀█"}, // n
	{"█▀▀", "██▄"},   // e
	{"█▀█", "█▀▄"},   // r
}

// logoPalette gives each glyph its color, from cyan to magenta. These
// 256-color codes work in every modern terminal and Windows 10+ console.
var logoPalette = [len(logoGlyphs)]int{51, 45, 39, 33, 27, 63, 99, 135, 171, 207, 201}

// segment is a piece of text printed in one color.
type segment struct{ color, text string }

// bannerGraph is a commit graph whose stale branches are pruned (×) while a
// whitelisted one lives on.
var bannerGraph = [][]segment{
	{{green, "●──●──●──●──●──●──●──●──●──●──●──●──●──●"}, {dim, "  main"}},
	{{dim, "   └──●──"}, {red, "×"}, {green, "     └──●──●──●"}, {dim, "     └──●──"}, {red, "×"}},
}

// Banner prints the logo, the commit graph and the tagline.
func (p *Printer) Banner() {
	var b strings.Builder
	for row := range 2 {
		b.WriteString(" ")
		for i, glyph := range logoGlyphs {
			b.WriteString(" " + p.paint(fmt.Sprintf("1;38;5;%d", logoPalette[i]), glyph[row]))
		}
		b.WriteString("\n")
	}
	for _, line := range bannerGraph {
		b.WriteString("  ")
		for _, s := range line {
			b.WriteString(p.paint(s.color, s.text))
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "  %s %s\n\n", p.paint(dim, "prune stale branches, keep what matters ·"), p.paint(bold, appVersion()))
	io.WriteString(p.w, b.String())
}
