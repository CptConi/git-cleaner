#!/bin/sh
# Checks that a commit message follows the repository convention: one single
# line "<type> (<feature>) <changes>", without body nor trailers (hence no
# Co-Authored-By).
#
#   check-commit-msg.sh <message-file>         raw message, as CI reads it
#   check-commit-msg.sh --hook <message-file>  message being written: cleaned
#                                              the way git records it first
#
# Exits 0 when the message complies, 1 otherwise with the reason on stderr.
# POSIX sh: runs in CI, in the commit-msg hook and in Git Bash on Windows.

set -eu
export LC_ALL=C

# The only place where the allowed types are listed.
types="feat fix docs chore refactor test ci perf build revert release"
alternatives=$(printf '%s' "$types" | tr ' ' '|')
pattern="^($alternatives) \\([A-Za-z0-9._-]+\\) [^[:space:]].*\$"

hook=false
if [ "${1:-}" = "--hook" ]; then
	hook=true
	shift
fi
if [ $# -ne 1 ] || [ ! -f "$1" ]; then
	echo "usage: check-commit-msg.sh [--hook] <message-file>" >&2
	exit 2
fi

reject() {
	{
		echo "commit message rejected: $1"
		[ -n "${msg:-}" ] && printf '  > %s\n' "$(printf '%s\n' "$msg" | sed -n 1p)"
		echo "expected: one single line \"<type> (<feature>) <changes>\","
		echo "          e.g. \"feat (exclude-repositories) add --exclude flag, wording for help\""
		echo "allowed types: $types"
	} >&2
	exit 1
}

if $hook; then
	# What git will record: no "git commit -v" diff below the scissors line,
	# no comment lines (git stripspace honors core.commentChar), no leading or
	# trailing blank lines.
	char=$(git config --get core.commentChar 2>/dev/null || true)
	case "$char" in "" | auto) char="#" ;; esac
	msg=$(sed "/^$char -\{24\} >8 -\{24\}\$/,\$d" "$1" | git stripspace --strip-comments)
	# Fixup commits are squashed before being pushed; CI rejects them otherwise.
	case "$msg" in "fixup! "* | "squash! "* | "amend! "*) exit 0 ;; esac
else
	msg=$(cat "$1") # command substitution drops the trailing newlines
fi

cr=$(printf '\r')
case "$msg" in *"$cr"*) reject "it contains a carriage return (Windows line ending)" ;; esac
[ -n "$msg" ] || reject "it is empty"
lines=$(printf '%s\n' "$msg" | awk 'END { print NR }')
[ "$lines" -eq 1 ] || reject "it has $lines lines (a body or trailers such as Co-Authored-By are not allowed)"
printf '%s\n' "$msg" | grep -Eq "$pattern" || reject "it does not match the format"
