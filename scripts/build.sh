#!/usr/bin/env bash

set -eu
set -o pipefail

readonly ROOT_DIR="$(cd "$(dirname "${0}")/.." && pwd)"
readonly PROGDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly BUILDPACKDIR="$(cd "${PROGDIR}/.." && pwd)"

# shellcheck source=SCRIPTDIR/.util/print.sh
source "${ROOT_DIR}/scripts/.util/print.sh"

function main() {
  local targets=()
  while [[ "${#}" != 0 ]]; do
    case "${1}" in
      --help|-h)
        shift 1
        usage
        exit 0
        ;;

      --target)
        targets+=("${2}")
        shift 2
        ;;

      "")
        # skip if the argument is empty
        shift 1
        ;;

      *)
        util::print::error "unknown argument \"${1}\""
    esac
  done

  mkdir -p "${BUILDPACKDIR}/bin"

  if [[ ${#targets[@]} -eq 0 ]]; then
    targets=("linux/amd64")
    util::print::info "Setting default target platform architecture to: linux/amd64"
  fi

  run::build
  cmd::build

  ## For backwards compatibility with amd64 workflows
  if [[ ${#targets[@]} -eq 1 && "${targets[0]}" == "linux/amd64" ]]; then
    cp -r "${BUILDPACKDIR}/linux/amd64/bin/" "${BUILDPACKDIR}/"
  fi
}

function usage() {
  cat <<-USAGE
build.sh [OPTIONS]

Builds the buildpack executables.

OPTIONS
  --target strings  Target platforms to build for.
                    Targets should be in the format '[os][/arch][/variant]'.
                      - To specify two different architectures: '--target "linux/amd64" --target "linux/arm64"'
  --help  -h        prints the command usage
USAGE
}

function run::build() {
  if [[ -f "${BUILDPACKDIR}/run/main.go" ]]; then
    pushd "${BUILDPACKDIR}" > /dev/null || return
      for target in "${targets[@]}"; do
        platform=$(echo "${target}" | cut -d '/' -f1)
        arch=$(echo "${target}" | cut -d'/' -f2)

        util::print::title "Building run... for platform: ${platform} and arch: ${arch}"

        GOOS=$platform \
        GOARCH=$arch \
        CGO_ENABLED=0 \
          go build \
            -ldflags="-s -w" \
            -o "${platform}/${arch}/bin/run" \
              "${BUILDPACKDIR}/run"

          echo "Success!"

          names=("detect")

          if [ -f "${BUILDPACKDIR}/extension.toml" ]; then
            names+=("generate")
          else
            names+=("build")
          fi

        for name in "${names[@]}"; do
          printf "%s" "Linking ${name}... "

          ln -fs "run" "${platform}/${arch}/bin/${name}"

          echo "Success!"
        done
      done

    popd > /dev/null || return
  fi
}

function cmd::build() {
  if [[ -d "${BUILDPACKDIR}/cmd" ]]; then
    local name
    for src in "${BUILDPACKDIR}"/cmd/*; do
      name="$(basename "${src}")"
      for target in "${targets[@]}"; do
        platform=$(echo "${target}" | cut -d '/' -f1)
        arch=$(echo "${target}" | cut -d'/' -f2)

        if [[ -f "${src}/main.go" ]]; then
          util::print::title "Building ${name}... for platform: ${platform} and arch: ${arch}"

          GOOS=$platform \
          GOARCH=$arch \
          CGO_ENABLED=0 \
            go build \
              -ldflags="-s -w" \
              -o "${BUILDPACKDIR}/${platform}/${arch}/bin/${name}" \
                "${src}/main.go"

          echo "Success!"
        else
          printf "%s" "Skipping ${name}... "
        fi
      done
    done
  fi
}

main "${@:-}"
