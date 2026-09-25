#!/bin/sh
# Scans git-cleaner for known vulnerabilities with govulncheck, once per
# release platform: linux, darwin and windows, with cgo disabled like the
# release builds (.goreleaser.yaml). Fails when the code reaches a vulnerable
# function, unless the vulnerability is listed in .github/vulncheck-allowlist;
# vulnerabilities of imported packages whose vulnerable functions are never
# reached are only reported as informational.
#
#   vulncheck.sh [<module-directory>]   default: the repository of the script
#
# The standard library analyzed is the one of the go command on the PATH (set
# GOTOOLCHAIN=local to keep go from switching to another toolchain). The
# latest govulncheck is installed with that go command, unless GOVULNCHECK
# names a binary built beforehand: needed to analyze with a Go older than the
# one that govulncheck needs to build. govulncheck cannot analyze the standard
# library of a Go newer than the one that built it.
#
# Exits 0 when no vulnerability is reached, 3 when some are (like govulncheck
# does), and with another status on errors. Needs jq.

set -eu
export LC_ALL=C # sort and comm must order the IDs the same way

platforms="linux darwin windows"
repo=$(cd "$(dirname "$0")/.." && pwd)
allowlist=.github/vulncheck-allowlist
module=${1:-$repo}

if [ $# -gt 1 ] || [ ! -f "$module/go.mod" ]; then
	echo "usage: vulncheck.sh [<module-directory>]" >&2
	exit 2
fi
for tool in go jq; do
	if ! command -v "$tool" >/dev/null; then
		echo "vulncheck.sh: $tool is required" >&2
		exit 2
	fi
done

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# retry <command>...: runs the command until it succeeds, 4 times at most. In
# JSON mode, govulncheck reports findings with exit status 0: a failure is a
# tool or network error (proxy.golang.org, vuln.go.dev), worth retrying.
retry() {
	attempt=1
	until "$@"; do
		if [ "$attempt" -gt 3 ]; then
			echo "vulncheck.sh: giving up after $attempt attempts" >&2
			return 1
		fi
		echo "vulncheck.sh: attempt $attempt failed, retrying in $((attempt * 10))s" >&2
		sleep $((attempt * 10))
		attempt=$((attempt + 1))
	done
}

# scan <goos>: the JSON messages of govulncheck for that platform.
scan() {
	GOOS=$1 CGO_ENABLED=0 "$govulncheck" -C "$module" -format json -scan symbol ./... >"$tmp/$1.json"
}

# ids <filter> <file>: the sorted vulnerability IDs that the jq filter extracts
# from the messages of the file. Through a file: in a pipeline, a failure of
# jq would go unnoticed.
ids() {
	jq -r "$1" "$2" >"$tmp/ids"
	sort -u "$tmp/ids"
}

count() {
	awk 'END { print NR }' "$1"
}

# words <file>: the lines of the file, space-separated and wrapped.
words() {
	paste -s -d ' ' "$1" | fold -s -w 76 | sed 's/^/  /; s/ *$//'
}

# annotate <title> <message>: a GitHub error annotation, in CI only. The title
# must not contain commas, which separate the annotation properties.
annotate() {
	if [ "${GITHUB_ACTIONS:-}" = true ]; then
		printf '::error title=%s::%s\n' "$1" "$(printf '%s' "$2" | sed 's/%/%25/g')"
	fi
}

# jq functions printing a trace (frames from the vulnerable symbol to the entry
# point) in one line, the way govulncheck does: where the code of the module
# makes the call that leads to the vulnerable symbol, e.g. "scan.go:50:25:
# git-cleaner.FindRepositories calls filepath.WalkDir, which calls os.Lstat".
compact='
def symbol:
	(.package | split("/") | last) + "."
	+ (if .receiver then (.receiver | ltrimstr("*")) + "." else "" end)
	+ (.function | split("$") | first);
def compact:
	. as $t
	| first(range(length) | select($t[.].module == $t[-1].module)) as $i
	| ($t[$i].position | if .line then "\(.filename):\(.line):\(.column): " else "" end)
	+ ($t[$i] | symbol)
	+ (if $i > 1 then " calls \($t[$i - 1] | symbol), which \(if $i > 2 then "eventually " else "" end)calls \($t[0] | symbol)"
	elif $i == 1 then " calls \($t[0] | symbol)"
	else "" end);
'

if [ -n "${GOVULNCHECK:-}" ]; then
	govulncheck=$GOVULNCHECK
else
	echo "Installing govulncheck@latest"
	retry env GOBIN="$tmp/bin" go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck=$tmp/bin/govulncheck
fi

echo "Scanning $module"
for goos in $platforms; do
	retry scan "$goos"
	# Findings whose trace starts with a function are reached by the code; the
	# others only name an imported package or a required module.
	ids 'select((.finding.trace[0].function // "") != "") | .finding.osv' "$tmp/$goos.json" >"$tmp/$goos.reached"
	ids '.finding.osv // empty' "$tmp/$goos.json" >"$tmp/found"
	comm -23 "$tmp/found" "$tmp/$goos.reached" >"$tmp/$goos.informational"
	jq -r '.config // empty | "\(.go_version), govulncheck \(.scanner_version), database of \(.db_last_modified)"' \
		"$tmp/$goos.json" >"$tmp/config"
	printf '%-8s %s reached, %s informational (%s)\n' "$goos:" \
		"$(count "$tmp/$goos.reached")" "$(count "$tmp/$goos.informational")" "$(cat "$tmp/config")"
done

for goos in $platforms; do cat "$tmp/$goos.reached"; done >"$tmp/ids"
sort -u "$tmp/ids" >"$tmp/reached"
for goos in $platforms; do cat "$tmp/$goos.informational"; done >"$tmp/ids"
sort -u "$tmp/ids" | comm -23 - "$tmp/reached" >"$tmp/informational"
# The allowlist holds one ID per line; "#" starts a comment.
awk '{ sub(/#.*/, "") } NF { print $1 }' "$repo/$allowlist" >"$tmp/ids"
sort -u "$tmp/ids" >"$tmp/allowed"
comm -23 "$tmp/reached" "$tmp/allowed" >"$tmp/failing"
comm -12 "$tmp/reached" "$tmp/allowed" >"$tmp/accepted"
comm -13 "$tmp/reached" "$tmp/allowed" >"$tmp/stale"

if [ -s "$tmp/informational" ]; then
	echo "Informational, in imported code that is never called ($(count "$tmp/informational")):"
	words "$tmp/informational"
fi
if [ -s "$tmp/accepted" ]; then
	echo "Reached but accepted in $allowlist:"
	words "$tmp/accepted"
fi
if [ -s "$tmp/stale" ]; then
	echo "Listed in $allowlist but no longer reached, the entries can be removed:"
	words "$tmp/stale"
fi
if [ ! -s "$tmp/failing" ]; then
	if [ -s "$tmp/accepted" ]; then
		echo "No known vulnerability reached by the code besides the accepted ones."
	else
		echo "No known vulnerability reached by the code."
	fi
	exit 0
fi

set --
for goos in $platforms; do set -- "$@" "$tmp/$goos.json"; done
echo
while read -r id; do
	on=
	for goos in $platforms; do
		if grep -qxF "$id" "$tmp/$goos.reached"; then on="$on $goos"; fi
	done
	summary=$(jq -rn --arg id "$id" 'first(inputs | .osv | select(.id? == $id) | .summary // "")' "$@")
	versions=$(jq -rn --arg id "$id" '
		first(inputs | .finding | select(.osv? == $id))
		| .trace[0] as $v
		| "found in \($v.module)@\($v.version), "
		+ (if .fixed_version then "fixed in \($v.module)@\(.fixed_version)" else "no fixed version yet" end)' "$@")
	jq -r --arg id "$id" "$compact"'
		.finding | select(.osv? == $id and (.trace[0].function // "") != "")
		| "  " + (.trace | compact)' "$@" >"$tmp/traces"
	echo "$id, reached on$on: $summary"
	echo "  $versions"
	sort -u "$tmp/traces"
	echo "  https://pkg.go.dev/vuln/$id"
	echo
	annotate "Vulnerability $id" "$summary (reached on$on; $versions)"
done <"$tmp/failing"
echo "Known vulnerabilities reached by the code: $(count "$tmp/failing")."
echo "Build with a Go release that fixes them, or, while none does, list them in"
echo "$allowlist with the reason."
exit 3
