#!/usr/bin/env bash
#
# Publish a provider version to the HCP Terraform private registry.
#
# The HCP Terraform private registry does NOT auto-update from GitHub releases.
# Each new version must be added manually via the API:
#   https://developer.hashicorp.com/terraform/cloud-docs/registry/publish-providers
#
# This script is idempotent: it skips steps whose artifacts are already uploaded,
# so it is safe to re-run after a partial failure.
#
# Prerequisites:
#   - GoReleaser dist/ artifacts present (zips + SHA256SUMS + SHA256SUMS.sig).
#   - The private provider record and the GPG signing key already exist in the
#     org (they do, since 2.3.0 is published). Discover the key id with:
#       curl -sH "Authorization: Bearer $TOKEN" \
#         https://app.terraform.io/api/registry/private/v2/gpg-keys
#   - curl, jq, shasum available.
#
# Required environment variables:
#   TOKEN   HCP Terraform API token with owners / "Manage Private Registry"
#           permission. NEVER paste this into chat; export it in your shell.
#   KEY_ID  GPG key id that matches the SHA256SUMS.sig signing key.
#
# Optional environment variables (defaults shown):
#   ORG=Flexpair
#   PROVIDER=guacamole
#   VERSION=2.3.1
#   DIST_DIR=dist
#   PROTOCOLS=5.0          (comma-separated, e.g. "5.0,6.0")
#   TFE_HOST=app.terraform.io
#
# Usage:
#   export TOKEN=...      # in your shell, not in chat
#   export KEY_ID=...
#   ./scripts/publish-private-provider.sh

set -euo pipefail

ORG="${ORG:-Flexpair}"
PROVIDER="${PROVIDER:-guacamole}"
VERSION="${VERSION:-2.3.1}"
DIST_DIR="${DIST_DIR:-dist}"
PROTOCOLS="${PROTOCOLS:-5.0}"
TFE_HOST="${TFE_HOST:-app.terraform.io}"

API="https://${TFE_HOST}/api/v2"
BASE="${API}/organizations/${ORG}/registry-providers/private/${ORG}/${PROVIDER}"
CT="Content-Type: application/vnd.api+json"
NAME="terraform-provider-${PROVIDER}"

die() { echo "ERROR: $*" >&2; exit 1; }

command -v curl >/dev/null || die "curl is required"
command -v jq   >/dev/null || die "jq is required"
command -v shasum >/dev/null || die "shasum is required"

[ -n "${TOKEN:-}" ]  || die "TOKEN is not set (export it in your shell, do not paste it into chat)"
[ -n "${KEY_ID:-}" ] || die "KEY_ID is not set (discover via /api/registry/private/v2/gpg-keys)"

AUTH="Authorization: Bearer ${TOKEN}"

SUMS="${DIST_DIR}/${NAME}_${VERSION}_SHA256SUMS"
SIG="${SUMS}.sig"
[ -f "${SUMS}" ] || die "missing ${SUMS}"
[ -f "${SIG}" ]  || die "missing ${SIG}"

# Build the protocols JSON array from the comma-separated PROTOCOLS value.
PROTO_JSON=$(printf '%s' "${PROTOCOLS}" | jq -R 'split(",")')

echo ">>> Target: ${BASE}"
echo ">>> Version: ${VERSION}  Key: ${KEY_ID}  Protocols: ${PROTOCOLS}"

# ---------------------------------------------------------------------------
# Step 1: ensure the version exists, capture shasums upload/download links.
# ---------------------------------------------------------------------------
echo ">>> [1/4] Ensuring version ${VERSION} exists"
ver_resp=$(curl -sS -H "${AUTH}" -H "${CT}" "${BASE}/versions/${VERSION}" || true)

if echo "${ver_resp}" | jq -e '.data.id' >/dev/null 2>&1; then
  echo "    version already present"
else
  echo "    creating version ${VERSION}"
  payload=$(jq -n --arg v "${VERSION}" --arg k "${KEY_ID}" --argjson p "${PROTO_JSON}" \
    '{data:{type:"registry-provider-versions",attributes:{version:$v,"key-id":$k,protocols:$p}}}')
  ver_resp=$(curl -sS -H "${AUTH}" -H "${CT}" -X POST --data "${payload}" "${BASE}/versions")
  echo "${ver_resp}" | jq -e '.data.id' >/dev/null 2>&1 \
    || die "failed to create version: ${ver_resp}"
fi

shasums_uploaded=$(echo "${ver_resp}" | jq -r '.data.attributes."shasums-uploaded" // false')
shasums_sig_uploaded=$(echo "${ver_resp}" | jq -r '.data.attributes."shasums-sig-uploaded" // false')
shasums_upload_url=$(echo "${ver_resp}" | jq -r '.data.links."shasums-upload" // empty')
shasums_sig_upload_url=$(echo "${ver_resp}" | jq -r '.data.links."shasums-sig-upload" // empty')

# ---------------------------------------------------------------------------
# Step 2: upload SHA256SUMS and SHA256SUMS.sig if not already uploaded.
# ---------------------------------------------------------------------------
echo ">>> [2/4] Uploading shasums + signature"
if [ "${shasums_uploaded}" = "true" ]; then
  echo "    SHA256SUMS already uploaded"
elif [ -n "${shasums_upload_url}" ]; then
  curl -sS -T "${SUMS}" "${shasums_upload_url}"
  echo "    uploaded ${SUMS}"
else
  die "no shasums-upload URL and shasums not uploaded; response: ${ver_resp}"
fi

if [ "${shasums_sig_uploaded}" = "true" ]; then
  echo "    SHA256SUMS.sig already uploaded"
elif [ -n "${shasums_sig_upload_url}" ]; then
  curl -sS -T "${SIG}" "${shasums_sig_upload_url}"
  echo "    uploaded ${SIG}"
else
  die "no shasums-sig-upload URL and signature not uploaded; response: ${ver_resp}"
fi

# ---------------------------------------------------------------------------
# Step 3: create each platform and upload its binary if not already uploaded.
# ---------------------------------------------------------------------------
echo ">>> [3/4] Publishing platforms"
# Parse SHA256SUMS: each line is "<sha>  <filename>".
while read -r sha filename; do
  [ -n "${sha}" ] || continue
  case "${filename}" in
    *_SHA256SUMS|*.sig) continue ;;
  esac

  # filename: terraform-provider-guacamole_2.3.1_<os>_<arch>.zip
  rest="${filename#"${NAME}"_"${VERSION}"_}"   # -> <os>_<arch>.zip
  rest="${rest%.zip}"                          # -> <os>_<arch>
  os="${rest%%_*}"
  arch="${rest#*_}"
  zip="${DIST_DIR}/${filename}"
  [ -f "${zip}" ] || die "missing platform artifact ${zip}"

  echo "    -> ${os}/${arch} (${filename})"

  plat_resp=$(curl -sS -H "${AUTH}" -H "${CT}" \
    "${BASE}/versions/${VERSION}/platforms/${os}/${arch}" || true)

  if echo "${plat_resp}" | jq -e '.data.attributes."provider-binary-uploaded" == true' >/dev/null 2>&1; then
    echo "       binary already uploaded, skipping"
    continue
  fi

  if echo "${plat_resp}" | jq -e '.data.id' >/dev/null 2>&1; then
    bin_url=$(echo "${plat_resp}" | jq -r '.data.links."provider-binary-upload" // empty')
  else
    payload=$(jq -n --arg os "${os}" --arg arch "${arch}" --arg sha "${sha}" --arg fn "${filename}" \
      '{data:{type:"registry-provider-version-platforms",attributes:{os:$os,arch:$arch,shasum:$sha,filename:$fn}}}')
    plat_resp=$(curl -sS -H "${AUTH}" -H "${CT}" -X POST --data "${payload}" \
      "${BASE}/versions/${VERSION}/platforms")
    bin_url=$(echo "${plat_resp}" | jq -r '.data.links."provider-binary-upload" // empty')
  fi

  [ -n "${bin_url}" ] || die "no provider-binary-upload URL for ${os}/${arch}: ${plat_resp}"
  curl -sS -T "${zip}" "${bin_url}"
  echo "       uploaded ${zip}"
done < "${SUMS}"

# ---------------------------------------------------------------------------
# Step 4: verify.
# ---------------------------------------------------------------------------
echo ">>> [4/4] Verifying version ${VERSION}"
final=$(curl -sS -H "${AUTH}" -H "${CT}" "${BASE}/versions/${VERSION}")
echo "${final}" | jq '{version:.data.attributes.version, shasums_uploaded:.data.attributes."shasums-uploaded", shasums_sig_uploaded:.data.attributes."shasums-sig-uploaded"}'

platforms=$(curl -sS -H "${AUTH}" -H "${CT}" "${BASE}/versions/${VERSION}/platforms")
echo "    platforms:"
echo "${platforms}" | jq -r '.data[] | "      \(.attributes.os)/\(.attributes.arch) uploaded=\(.attributes."provider-binary-uploaded")"'

echo ">>> Done. Once shasums + the required platform (at least linux/amd64) are uploaded,"
echo ">>> consumers can use ${ORG}/${PROVIDER} ${VERSION} in HCP Terraform."
