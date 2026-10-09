#!/bin/sh
set -eu
version=1.19.1
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in darwin|linux) ;; *) echo "Supported systems: macOS and Linux" >&2; exit 1 ;; esac
case "$(uname -m)" in x86_64|amd64) arch=x86_64 ;; arm64|aarch64) arch=arm64 ;; *) echo "Unsupported CPU architecture" >&2; exit 1 ;; esac
if [ -x "$root/bin/kyverno" ] && "$root/bin/kyverno" version 2>/dev/null | awk -v v="$version" '$0 == "Version: " v { found=1 } END { exit !found }'; then
  echo "Kyverno $version is already installed."
  exit 0
fi
temp=$(mktemp -d)
trap 'rm -rf "$temp"' EXIT HUP INT TERM
asset="kyverno-cli_v${version}_${os}_${arch}.tar.gz"
base="https://github.com/kyverno/kyverno/releases/download/v${version}"
curl -fsSL --retry 2 "$base/$asset" -o "$temp/$asset"
curl -fsSL --retry 2 "$base/checksums.txt" -o "$temp/checksums.txt"
expected=$(awk -v file="$asset" '$2 == file || $2 == "*" file { print $1 }' "$temp/checksums.txt")
if command -v sha256sum >/dev/null 2>&1; then actual=$(sha256sum "$temp/$asset" | awk '{print $1}'); else actual=$(shasum -a 256 "$temp/$asset" | awk '{print $1}'); fi
if [ -z "$expected" ] || [ "$actual" != "$expected" ]; then echo "Kyverno release checksum verification failed" >&2; exit 1; fi
tar -xzf "$temp/$asset" -C "$temp"
mkdir -p "$root/bin"
install -m 755 "$temp/kyverno" "$root/bin/kyverno"
echo "Installed Kyverno $version ($os/$arch); release checksum verified."
