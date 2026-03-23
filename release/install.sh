#!/usr/bin/env bash
set -euo pipefail

REPO="${REPO:-oleksandrskoruk/docusnap}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${VERSION:-${1:-}}"

if [[ -z "${VERSION}" ]]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
fi

if [[ -z "${VERSION}" ]]; then
  echo "Unable to resolve release version. Set VERSION=vX.Y.Z and retry." >&2
  exit 1
fi

uname_s="$(uname -s)"
uname_m="$(uname -m)"

case "${uname_s}" in
  Linux) goos="linux" ;;
  Darwin) goos="darwin" ;;
  MINGW*|MSYS*|CYGWIN*) goos="windows" ;;
  *)
    echo "Unsupported OS: ${uname_s}" >&2
    exit 1
    ;;
esac

case "${uname_m}" in
  x86_64|amd64) goarch="amd64" ;;
  arm64|aarch64) goarch="arm64" ;;
  *)
    echo "Unsupported architecture: ${uname_m}" >&2
    exit 1
    ;;
esac

archive_ext=".tar.gz"
if [[ "${goos}" == "windows" ]]; then
  archive_ext=".zip"
fi

artifact="docusnap-${VERSION}-${goos}-${goarch}${archive_ext}"
url="https://github.com/${REPO}/releases/download/${VERSION}/${artifact}"

tmp_dir="$(mktemp -d)"
trap 'rm -rf "${tmp_dir}"' EXIT

curl -fsSL "${url}" -o "${tmp_dir}/${artifact}"

if [[ "${archive_ext}" == ".zip" ]]; then
  unzip -q "${tmp_dir}/${artifact}" -d "${tmp_dir}"
  binary_source="$(find "${tmp_dir}" -type f -name 'docusnap.exe' | head -n1)"
  binary_target="${INSTALL_DIR}/docusnap.exe"
else
  tar -xzf "${tmp_dir}/${artifact}" -C "${tmp_dir}"
  binary_source="$(find "${tmp_dir}" -type f -name 'docusnap' | head -n1)"
  binary_target="${INSTALL_DIR}/docusnap"
fi

if [[ -z "${binary_source}" ]]; then
  echo "Downloaded archive does not contain the docusnap binary." >&2
  exit 1
fi

mkdir -p "${INSTALL_DIR}"
install -m 0755 "${binary_source}" "${binary_target}"

echo "Installed ${binary_target}"
