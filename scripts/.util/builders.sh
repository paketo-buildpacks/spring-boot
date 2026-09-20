#!/usr/bin/env bash

set -eu
set -o pipefail

# shellcheck source=SCRIPTDIR/print.sh
source "$(dirname "${BASH_SOURCE[0]}")/print.sh"

function util::builders::list() {
  local integrationJSON="${1}"
  local builders=""
  if [[ -f "${integrationJSON}" ]]; then
    builders="$(jq --compact-output 'select(.builder != null) | [.builder]' "${integrationJSON}")"

    if [[ -z "${builders}" ]]; then
      builders="$(jq --compact-output 'select(.builders != null) | .builders' "${integrationJSON}")"
    fi
  fi

  if [[ -z "${builders}" ]]; then
    util::print::info "No builders specified. Falling back to default builder..."
    builders="$(jq --compact-output --null-input '["index.docker.io/paketobuildpacks/builder-jammy-buildpackless-base:latest"]')"
  fi

  echo "${builders}"
}
