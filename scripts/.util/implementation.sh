#!/usr/bin/env bash

set -eu
set -o pipefail

# shellcheck source=SCRIPTDIR/tools.sh
source "$(dirname "${BASH_SOURCE[0]}")/tools.sh"

# Shared helpers for implementation repositories.
#
# A repository that packages a combined buildpack plus one buildpack per
# component opts in by adding a multi-implementation.json file at its root.
# Repositories without that file get the default in-memory config below, so
# every caller can read the same normalized shape regardless of layout.

function util::implementation::config-path() {
  local root
  root="${1}"
  printf '%s/multi-implementation.json' "${root}"
}

function util::implementation::default-config() {
  local root name
  root="${1}"
  name="$(util::tools::repo-name "${root}")"

  jq --null-input --compact-output --arg name "${name}" '{
    multi: false,
    combined: $name,
    integration: true,
    "libpak-tools": null,
    buildpacks: [{name: $name, image: $name}]
  }'
}

function util::implementation::config() {
  local root path name
  root="${1}"
  path="$(util::implementation::config-path "${root}")"

  if [[ ! -f "${path}" ]]; then
    util::implementation::default-config "${root}"
    return
  fi

  name="$(util::tools::repo-name "${root}")"
  jq --compact-output --arg name "${name}" '{
    multi: true,
    combined: (.combined // $name),
    integration: (if has("integration") then .integration else true end),
    "libpak-tools": (."libpak-tools" // null),
    buildpacks: ((.buildpacks // []) | map({name: .name, image: (.image // .name)}))
  }' "${path}"
}

function util::implementation::is_multi() {
  util::implementation::config "${1}" | jq -r '.multi'
}

function util::implementation::combined() {
  util::implementation::config "${1}" | jq -r '.combined'
}

function util::implementation::buildpacks() {
  util::implementation::config "${1}" | jq --compact-output '[.buildpacks[].name]'
}

function util::implementation::components() {
  local root combined
  root="${1}"
  combined="$(util::implementation::combined "${root}")"

  util::implementation::config "${root}" \
    | jq --compact-output --arg combined "${combined}" \
      '[.buildpacks[].name | select(. != $combined)]'
}

function util::implementation::integration() {
  util::implementation::config "${1}" | jq -r '.integration'
}

function util::implementation::image() {
  local root name
  root="${1}"
  name="${2}"

  util::implementation::config "${root}" \
    | jq -r --arg name "${name}" '.buildpacks[] | select(.name == $name) | .image'
}

function util::implementation::asset-suffix() {
  local root name
  root="${1}"
  name="${2}"

  if [[ "$(util::implementation::is_multi "${root}")" == "true" ]]; then
    printf '%s' "-${name}"
  else
    printf ''
  fi
}
