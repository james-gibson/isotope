#!/usr/bin/env bash
#
# ship-plane.sh — build cmd/plane with the corpus-derived plane weights embedded.
#
# The weights live in this (public) repo only as a SOPS-encrypted blob,
# plane/lore-weights.enc.json. Building with them:
#
#   ./scripts/ship-plane.sh                 # → out/plane
#   ./scripts/ship-plane.sh /tmp/plane      # → /tmp/plane
#
# decrypts the blob to the git-ignored plane/lore-weights.json, builds with
# `-tags loreweights`, then deletes the plaintext so the corpus labels (universe
# ids) never linger in the tree. The result resolves and sorts the corpus's own
# plane names with no --weights file.
#
# Decryption needs the age key: SOPS_AGE_KEY_CMD (e.g. the macOS Keychain
# helper) or SOPS_AGE_KEY_FILE. Without a key, build normally instead — a
# default build embeds only the generic anchor-42i hypothesis model.
#
# To refresh the blob after the corpus changes:
#
#   ./scripts/ship-plane.sh encrypt <path/to/derived-weights.json>
#
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
enc="$here/plane/lore-weights.enc.json"
plain="$here/plane/lore-weights.json"

# The age recipient the blob is encrypted to. A public key: safe to commit.
recipient="age1zna8jjq5gsf3w2spwsqy6evnr37vsl6q252cvs702033djgtusasedq6a9"

sops_cmd() {
	if ! command -v sops >/dev/null 2>&1; then
		echo "ship-plane: sops not found on PATH" >&2
		exit 1
	fi
}

cmd="${1:-build}"
case "$cmd" in
encrypt)
	sops_cmd
	src="${2:?usage: ship-plane.sh encrypt <path/to/derived-weights.json>}"
	[ -f "$src" ] || { echo "ship-plane: no such file: $src" >&2; exit 1; }
	sops --encrypt --input-type binary --output-type binary --age "$recipient" "$src" > "$enc"
	echo "ship-plane: wrote $enc"
	;;
build | *)
	out="${1:-$here/out/plane}"
	case "$out" in /*) ;; *) out="$PWD/$out" ;; esac
	sops_cmd
	cleanup() { rm -f "$plain"; }
	trap cleanup EXIT
	sops --decrypt --input-type binary --output-type binary "$enc" > "$plain"
	mkdir -p "$(dirname "$out")"
	( cd "$here" && CGO_ENABLED=0 go build -tags loreweights -trimpath -ldflags='-s -w' -o "$out" ./cmd/plane )
	echo "ship-plane: wrote $out (corpus weights embedded)"
	;;
esac
