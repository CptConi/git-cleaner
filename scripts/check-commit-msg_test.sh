#!/bin/sh
# Tests of check-commit-msg.sh. Run with: sh scripts/check-commit-msg_test.sh
# POSIX sh, so that CI runs it on Linux, macOS and Git Bash for Windows.

set -u
here=$(dirname "$0")
checker="$here/check-commit-msg.sh"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
failures=0

# check <accept|reject> <raw|hook> <description> <printf format> [args...]
# The message is written with printf, so "\n", "\r" and "\t" can be used.
check() {
	want=$1 mode=$2 what=$3
	shift 3
	# shellcheck disable=SC2059
	printf "$@" >"$tmp/msg"
	if [ "$mode" = hook ]; then
		sh "$checker" --hook "$tmp/msg" 2>"$tmp/err"
	else
		sh "$checker" "$tmp/msg" 2>"$tmp/err"
	fi
	status=$?
	if { [ "$want" = accept ] && [ $status -ne 0 ]; } || { [ "$want" = reject ] && [ $status -ne 1 ]; }; then
		echo "FAIL ($mode, want $want, got exit $status): $what"
		sed 's/^/    /' "$tmp/err"
		# In GitHub Actions, also as an annotation, readable without the logs.
		if [ -n "${GITHUB_ACTIONS:-}" ]; then
			details=$(sed 's/%/%25/g' "$tmp/err" | awk 'NR > 1 { printf "%%0A" } { printf "%s", $0 }')
			# Commas and colons separate the parameters: none in the title.
			echo "::error title=check-commit-msg test::$what - $mode mode - want $want - got exit $status%0A$details"
		fi
		failures=$((failures + 1))
	else
		echo "ok   ($mode, $want): $what"
	fi
}

# Compliant messages, as git records them or as GitHub creates them.
check accept raw "feature commit" 'feat (exclude-repositories) add --exclude flag, wording for help\n'
check accept raw "fix commit" 'fix (initial-release) disable git credential manager sign-in prompts\n'
check accept raw "no trailing newline (GitHub UI)" 'docs (openspec) propose homebrew-tap change'
check accept raw "trailing blank lines" 'chore (init) initialize repository\n\n\n'
check accept raw "Dependabot single update, squash-merged" 'chore (deps) bump actions/setup-go from 7 to 8 in the actions group (#12)\n'
check accept raw "Dependabot grouped update, squash-merged" 'chore (deps) bump the actions group with 3 updates (#13)\n'
check accept raw "merge of dev into main" 'release (v1.1.0) merge dev into main (#7)\n'
check accept raw "Homebrew tap update" 'chore (homebrew) update git-cleaner to v1.2.0\n'

# Non-compliant messages.
check reject raw "Conventional Commits colon" 'feat(exclude-repositories): add --exclude flag\n'
check reject raw "colon after the feature" 'feat (exclude-repositories): add --exclude flag\n'
check reject raw "Co-Authored-By trailer" 'feat (x) add y\n\nCo-Authored-By: Someone <someone@example.com>\n'
check reject raw "body" 'feat (x) add y\n\nMore details on a second paragraph.\n'
check reject raw "unknown type" 'wip (exclude-repositories) try something\n'
check reject raw "missing feature" 'feat add y\n'
check reject raw "empty description" 'feat (x) \n'
check reject raw "no description" 'feat (x)\n'
check reject raw "tab before the description" 'feat (x) \tadd y\n'
check reject raw "carriage return" 'feat (x) add y\r\n'
check reject raw "git merge default message" "Merge branch 'dev' into feature/x\n"
check reject raw "GitHub merge default message" 'Merge pull request #3 from CptConi/dev\n'
check reject raw "git revert default message" 'Revert "feat (x) add y"\n\nThis reverts commit 0123456789abcdef.\n'
check reject raw "unsquashed fixup" 'fixup! feat (x) add y\n'
check reject raw "empty message" ''

# Hook mode: the message as git hands it to the commit-msg hook.
check accept hook "comments added by git" 'feat (x) add y\n# Please enter the commit message for your changes.\n#\n# On branch feature/x\n'
check accept hook "git commit -v scissors and diff" 'feat (x) add y\n# ------------------------ >8 ------------------------\n# Do not modify or remove the line above.\ndiff --git a/f b/f\n+added line\n'
check accept hook "leading blank line" '\nfeat (x) add y\n'
check accept hook "fixup commit" 'fixup! feat (x) add y\n'
check accept hook "squash commit with a body" 'squash! feat (x) add y\n\nextra notes\n'
check reject hook "non-compliant subject" 'added some stuff\n# Please enter the commit message for your changes.\n'
check reject hook "body kept below comments" 'feat (x) add y\n\nsecond paragraph\n# comment\n'

if [ $failures -ne 0 ]; then
	echo "$failures test(s) failed"
	exit 1
fi
echo "all tests passed"
