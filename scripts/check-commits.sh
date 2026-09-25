#!/bin/sh
# CI: checks the message of every commit that the tested revision would bring
# to its target branch, merge commits included, plus the title of pull
# requests into main (it becomes their merge commit message). Needs a full
# history checkout (fetch-depth: 0). Reads the GitHub context from:
#   EVENT                     github.event_name
#   REF_NAME, ACTOR           github.ref_name, github.actor (push)
#   PR_AUTHOR, PR_BASE        pull request author login and base branch
#   PR_BASE_SHA, PR_HEAD_SHA  pull request base and head commits
#   PR_TITLE                  pull request title

set -eu
here=$(dirname "$0")
checker="$here/check-commit-msg.sh"
allowlist=.github/commit-check-allowlist
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
failed=0

# annotate <title> <text>: GitHub error annotation, newlines escaped.
annotate() {
	text=$(printf '%s' "$2" | sed 's/%/%25/g' | awk 'BEGIN { ORS = "%0A" } { print }')
	echo "::error title=$1::$text"
}

case "$EVENT" in
pull_request)
	if [ "$PR_AUTHOR" = "dependabot[bot]" ]; then
		echo "Pull request opened by Dependabot: commit messages are not checked (it is squash-merged)."
		exit 0
	fi
	if [ "$PR_BASE" = main ]; then
		printf '%s\n' "$PR_TITLE" >"$tmp/title"
		if out=$(sh "$checker" "$tmp/title" 2>&1); then
			echo "ok  pull request title: $PR_TITLE"
		else
			printf '%s\n' "$out"
			annotate "Pull request title (it becomes the merge commit message)" "$out"
			failed=1
		fi
	fi
	range="$PR_BASE_SHA..$PR_HEAD_SHA"
	;;
push)
	case "$ACTOR:$REF_NAME" in
	"dependabot[bot]:dependabot/"*)
		echo "Push by Dependabot to its own branch: commit messages are not checked."
		exit 0
		;;
	esac
	case "$REF_NAME" in
	main)
		echo "Push to main: only merged pull requests get there, nothing to check."
		exit 0
		;;
	dev) range="origin/main..HEAD" ;;
	*) range="origin/dev..HEAD" ;;
	esac
	;;
*)
	echo "Event $EVENT: nothing to check."
	exit 0
	;;
esac

echo "Checking the commits of $range"
# Listed first: an invalid range must fail the job, not check nothing (a
# failing command substitution in a "for" list is ignored by set -e).
commits=$(git rev-list "$range") || {
	annotate "Commit messages" "cannot list the commits of $range"
	exit 1
}
for sha in $commits; do
	short=$(printf '%.10s' "$sha")
	if grep -q "^$sha" "$allowlist" 2>/dev/null; then
		echo "skip $short (listed in $allowlist)"
		continue
	fi
	# The raw message: everything after the first blank line of the commit.
	git cat-file commit "$sha" | sed '1,/^$/d' >"$tmp/msg"
	if out=$(sh "$checker" "$tmp/msg" 2>&1); then
		echo "ok   $short $(sed -n 1p "$tmp/msg")"
	else
		printf 'FAIL %s\n%s\n' "$short" "$out"
		annotate "Commit $short" "$out"
		failed=1
	fi
done
exit $failed
