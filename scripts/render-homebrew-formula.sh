#!/usr/bin/env bash
set -euo pipefail

VERSION="${VERSION:-}"
REPO="${REPO:-oleksandrskoruk/docusnap}"
CHECKSUM_FILE="${CHECKSUM_FILE:-}"
OUTPUT="${OUTPUT:-}"

if [[ -z "${VERSION}" || -z "${CHECKSUM_FILE}" || -z "${OUTPUT}" ]]; then
  echo "VERSION, CHECKSUM_FILE, and OUTPUT are required" >&2
  exit 1
fi

checksum_for() {
  local artifact="$1"
  awk -v artifact="${artifact}" '{
    path = $2
    sub(/^\.\//, "", path)
    if (path == artifact) {
      print $1
    }
  }' "${CHECKSUM_FILE}"
}

darwin_amd64="docusnap-${VERSION}-darwin-amd64.tar.gz"
darwin_arm64="docusnap-${VERSION}-darwin-arm64.tar.gz"
linux_amd64="docusnap-${VERSION}-linux-amd64.tar.gz"
linux_arm64="docusnap-${VERSION}-linux-arm64.tar.gz"

cat > "${OUTPUT}" <<EOF
class Docusnap < Formula
  desc "Local-first CLI for repository snapshots and generated documentation"
  homepage "https://github.com/${REPO}"
  version "${VERSION#v}"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/${REPO}/releases/download/${VERSION}/${darwin_arm64}"
      sha256 "$(checksum_for "${darwin_arm64}")"
    else
      url "https://github.com/${REPO}/releases/download/${VERSION}/${darwin_amd64}"
      sha256 "$(checksum_for "${darwin_amd64}")"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/${REPO}/releases/download/${VERSION}/${linux_arm64}"
      sha256 "$(checksum_for "${linux_arm64}")"
    else
      url "https://github.com/${REPO}/releases/download/${VERSION}/${linux_amd64}"
      sha256 "$(checksum_for "${linux_amd64}")"
    end
  end

  def install
    bin.install Dir["**/docusnap"].first => "docusnap"
    pkgshare.install Dir["**/README.md"].first if Dir["**/README.md"].any?
  end

  test do
    assert_match "DocuSnap", shell_output("#{bin}/docusnap version")
  end
end
EOF
