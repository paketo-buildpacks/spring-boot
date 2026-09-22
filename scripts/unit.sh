#!/usr/bin/env bash

set -eu
set -o pipefail

readonly PROGDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly BUILDPACKDIR="$(cd "${PROGDIR}/.." && pwd)"

GO_TEST_PARAMS="${GO_TEST_PARAMS:-}"

# shellcheck source=SCRIPTDIR/.util/tools.sh
source "${PROGDIR}/.util/tools.sh"

# shellcheck source=SCRIPTDIR/.util/print.sh
source "${PROGDIR}/.util/print.sh"

function main() {
  while [[ "${#}" != 0 ]]; do
    case "${1}" in
      --help|-h)
        shift 1
        usage
        exit 0
        ;;

      "")
        # skip if the argument is empty
        shift 1
        ;;

      *)
        util::print::error "unknown argument \"${1}\""
    esac
  done

  unit::run
}

function usage() {
  cat <<-USAGE
unit.sh [OPTIONS]

Runs the unit test suite.

OPTIONS
  --help  -h  prints the command usage
USAGE
}

function unit::run() {
  util::print::title "Run Buildpack Unit Tests"

  testout=$(mktemp)
  pushd "${BUILDPACKDIR}" > /dev/null
    if go test ./... -v ${GO_TEST_PARAMS} -run Unit | tee "${testout}"; then
      util::tools::tests::checkfocus "${testout}"
      util::print::success "** GO Test Succeeded **"
    else
      util::print::error "** GO Test Failed **"
    fi
  popd > /dev/null
}

main "${@:-}"
