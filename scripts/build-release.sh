#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${VERSION:-}"
COMMIT="${COMMIT:-$(git -C "${ROOT_DIR}" rev-parse --short HEAD 2>/dev/null || echo none)}"
DATE="${DATE:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}"
TARGETS="${TARGETS:-linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64}"
DIST_DIR="${DIST_DIR:-${ROOT_DIR}/dist}"
RELEASE_DIR="${RELEASE_DIR:-${ROOT_DIR}/release}"

if [[ -z "${VERSION}" ]]; then
  echo "VERSION is required, for example VERSION=v0.1.0" >&2
  exit 1
fi

rm -rf "${DIST_DIR}" "${RELEASE_DIR}"
mkdir -p "${DIST_DIR}" "${RELEASE_DIR}"

for target in ${TARGETS}; do
  goos="${target%/*}"
  goarch="${target#*/}"
  ext=""
  if [[ "${goos}" == "windows" ]]; then
    ext=".exe"
  fi

  package_base="docusnap-${VERSION}-${goos}-${goarch}"
  package_dir="${DIST_DIR}/${package_base}"
  binary_path="${package_dir}/docusnap${ext}"

  mkdir -p "${package_dir}"

  env \
    GOOS="${goos}" \
    GOARCH="${goarch}" \
    CGO_ENABLED=0 \
    go build \
    -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${DATE}" \
    -o "${binary_path}" \
    ./cmd/docusnap

  cp "${ROOT_DIR}/README.md" "${package_dir}/README.md"
  cp "${ROOT_DIR}/install.sh" "${package_dir}/install.sh"

  if [[ "${goos}" == "windows" ]]; then
    (
      cd "${DIST_DIR}"
      zip -qr "${RELEASE_DIR}/${package_base}.zip" "${package_base}"
    )
  else
    tar -C "${DIST_DIR}" -czf "${RELEASE_DIR}/${package_base}.tar.gz" "${package_base}"
  fi
done

(
  cd "${RELEASE_DIR}"
  shasum -a 256 ./*.tar.gz ./*.zip > SHA256SUMS.txt
)

cp "${ROOT_DIR}/install.sh" "${RELEASE_DIR}/install.sh"

CHECKSUM_FILE="${RELEASE_DIR}/SHA256SUMS.txt" \
VERSION="${VERSION}" \
REPO="${REPO:-oleksandrskoruk/docusnap}" \
OUTPUT="${RELEASE_DIR}/docusnap.rb" \
"${ROOT_DIR}/scripts/render-homebrew-formula.sh"

echo "release artifacts written to ${RELEASE_DIR}"
