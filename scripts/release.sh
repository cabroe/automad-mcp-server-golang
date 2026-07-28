#!/usr/bin/env bash
#
# release.sh — cut a release by validating, tagging, and pushing a semver tag.
#
# Pushing the tag triggers .github/workflows/release.yml, which builds the
# cross-platform binaries and publishes the GitHub release via GoReleaser.
#
# Usage:
#   scripts/release.sh v1.2.3            # validate, tag, and push v1.2.3
#   scripts/release.sh --dry-run v1.2.3  # run every check but do not tag/push
#   scripts/release.sh --yes v1.2.3      # skip the confirmation prompt
#
# Steps: verify tooling and a clean tree on the default branch that is in sync
# with origin, ensure the tag is new, run the quality gate (make check) and
# `goreleaser check`, then create an annotated tag and push it.
set -euo pipefail

readonly REMOTE="origin"
readonly SEMVER_RE='^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?(\+[0-9A-Za-z.]+)?$'

die() {
	printf 'error: %s\n' "$*" >&2
	exit 1
}

info() { printf '==> %s\n' "$*"; }

usage() {
	sed -n '3,15p' "$0" | sed 's/^# \{0,1\}//'
	exit "${1:-0}"
}

main() {
	local dry_run=0 assume_yes=0 version=""

	while [ $# -gt 0 ]; do
		case "$1" in
		--dry-run) dry_run=1 ;;
		--yes | -y) assume_yes=1 ;;
		-h | --help) usage 0 ;;
		-*) die "unknown flag: $1 (see --help)" ;;
		*)
			[ -z "$version" ] || die "unexpected extra argument: $1"
			version="$1"
			;;
		esac
		shift
	done

	[ -n "$version" ] || usage 1
	[[ "$version" =~ $SEMVER_RE ]] || die "version must be a semver tag like v1.2.3 or v1.2.3-rc.1 (got: $version)"

	# Run from the repository root so make/goreleaser see the right files.
	local root
	root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
	cd "$root"

	command -v git >/dev/null || die "git is not installed"
	command -v make >/dev/null || die "make is not installed"

	git rev-parse --is-inside-work-tree >/dev/null 2>&1 || die "not inside a git repository"

	# Clean working tree — a release must reflect exactly what is committed.
	[ -z "$(git status --porcelain)" ] || die "working tree is dirty; commit or stash changes first"

	# On the default branch, in sync with origin.
	local default_branch current_branch
	default_branch="$(git symbolic-ref --quiet --short "refs/remotes/${REMOTE}/HEAD" 2>/dev/null | sed "s#^${REMOTE}/##")"
	default_branch="${default_branch:-main}"
	current_branch="$(git rev-parse --abbrev-ref HEAD)"
	if [ "$current_branch" != "$default_branch" ]; then
		die "on branch '$current_branch', expected the default branch '$default_branch'"
	fi

	info "fetching tags and refs from $REMOTE"
	git fetch --tags --quiet "$REMOTE"

	local local_head remote_head
	local_head="$(git rev-parse HEAD)"
	remote_head="$(git rev-parse "${REMOTE}/${default_branch}")"
	[ "$local_head" = "$remote_head" ] || die "local $default_branch is not in sync with ${REMOTE}/${default_branch}; push or pull first"

	# The tag must not already exist locally or remotely.
	if git rev-parse -q --verify "refs/tags/${version}" >/dev/null; then
		die "tag $version already exists locally"
	fi
	if git ls-remote --exit-code --tags "$REMOTE" "refs/tags/${version}" >/dev/null 2>&1; then
		die "tag $version already exists on $REMOTE"
	fi

	info "running quality gate (make check)"
	make check

	if command -v goreleaser >/dev/null 2>&1; then
		info "validating release config (goreleaser check)"
		goreleaser check
	else
		printf 'warning: goreleaser CLI not found; skipping local config validation\n' >&2
	fi

	local slug url
	slug="$(remote_slug)"
	url="https://github.com/${slug}/actions"

	if [ "$dry_run" -eq 1 ]; then
		info "dry run: all checks passed for $version on $default_branch ($local_head)"
		info "dry run: would run: git tag -a $version -m $version && git push $REMOTE $version"
		return 0
	fi

	if [ "$assume_yes" -eq 0 ]; then
		printf 'Release %s from %s (%s)? [y/N] ' "$version" "$default_branch" "${local_head:0:12}"
		local reply
		read -r reply
		case "$reply" in
		y | Y | yes | YES) ;;
		*) die "aborted" ;;
		esac
	fi

	info "creating annotated tag $version"
	git tag -a "$version" -m "$version"

	info "pushing $version to $REMOTE"
	if ! git push "$REMOTE" "$version"; then
		git tag -d "$version" >/dev/null 2>&1 || true
		die "failed to push $version; removed the local tag so you can retry"
	fi

	info "released $version"
	[ -n "$slug" ] && info "watch the release build: $url"
}

# remote_slug prints the "owner/repo" for the origin remote, or "" if unknown.
remote_slug() {
	local raw
	raw="$(git remote get-url "$REMOTE" 2>/dev/null || true)"
	[ -n "$raw" ] || return 0
	raw="${raw%.git}"
	case "$raw" in
	git@*:*) printf '%s\n' "${raw##*:}" ;;
	*://*) printf '%s\n' "$(printf '%s' "$raw" | sed -E 's#^[a-z]+://[^/]+/##')" ;;
	*) printf '\n' ;;
	esac
}

main "$@"
