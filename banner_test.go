package main

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestBannerArt(t *testing.T) {
	for i, glyph := range logoGlyphs {
		if top, bottom := utf8.RuneCountInString(glyph[0]), utf8.RuneCountInString(glyph[1]); top != bottom {
			t.Errorf("glyph %d: rows are %d and %d columns wide", i, top, bottom)
		}
	}

	// Every fork of the commit graph must start right under a commit.
	var main, forks []rune
	for _, s := range bannerGraph[0] {
		main = append(main, []rune(s.text)...)
	}
	for _, s := range bannerGraph[1] {
		forks = append(forks, []rune(s.text)...)
	}
	for i, r := range forks {
		if r == '└' && (i >= len(main) || main[i] != '●') {
			t.Errorf("fork at column %d is not under a commit", i)
		}
	}

	var out bytes.Buffer
	(&Printer{w: &out, opts: &Options{}}).Banner()
	for _, line := range strings.Split(out.String(), "\n") {
		if n := utf8.RuneCountInString(line); n > 80 {
			t.Errorf("banner line is %d columns wide, more than 80: %q", n, line)
		}
	}
	if strings.Contains(out.String(), "\x1b[") {
		t.Error("banner contains color codes although colors are disabled")
	}
}

func TestBannerOnlyForTerminals(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FORCE_COLOR", "")
	var out bytes.Buffer
	if code := run([]string{"--no-fetch", root}, &out, &out); code != exitOK {
		t.Fatalf("exit code %d:\n%s", code, &out)
	}
	if strings.Contains(out.String(), logoGlyphs[0][0]) || strings.Contains(out.String(), "\x1b[") {
		t.Errorf("banner or colors written to a non-terminal:\n%s", &out)
	}

	t.Setenv("FORCE_COLOR", "1")
	out.Reset()
	run([]string{"--no-fetch", root}, &out, &out)
	if !strings.Contains(out.String(), "\x1b[1;38;5;51m") || !strings.Contains(out.String(), "prune stale branches") {
		t.Errorf("FORCE_COLOR=1 did not produce the colored banner:\n%s", &out)
	}
}

func TestAppVersion(t *testing.T) {
	defer func(v string) { version = v }(version)
	version = "1.2.3"
	if got := appVersion(); got != "1.2.3" {
		t.Errorf("appVersion() = %q, want 1.2.3", got)
	}
	version = ""
	if got := appVersion(); got == "" {
		t.Error("appVersion() is empty without ldflags")
	}
}
