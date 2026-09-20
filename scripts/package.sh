#!/usr/bin/env bash

set -eu
set -o pipefail

readonly ROOT_DIR="$(cd "$(dirname "${0}")/.." && pwd)"
readonly BIN_DIR="${ROOT_DIR}/.bin"
readonly BUILD_DIR="${ROOT_DIR}/build"

# shellcheck source=SCRIPTDIR/.util/tools.sh
source "${ROOT_DIR}/scripts/.util/tools.sh"

# shellcheck source=SCRIPTDIR/.util/print.sh
source "${ROOT_DIR}/scripts/.util/print.sh"

function main {
  local version output token
  token=""

  while [[ "${#}" != 0 ]]; do
    case "${1}" in
      --version|-v)
        version="${2}"
        shift 2
        ;;

      --output|-o)
        output="${2}"
        shift 2
        ;;

      --token|-t)
        token="${2}"
        shift 2
        ;;

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

  if [[ -z "${version:-}" ]]; then
    usage
    echo
    util::print::error "--version is required"
  fi

  if [[ -z "${output:-}" ]]; then
    output="${BUILD_DIR}/buildpackage.cnb"
  fi

  repo::prepare

  tools::install "${token}"

  buildpack_type=buildpack
  if [ -f "${ROOT_DIR}/extension.toml" ]; then
    buildpack_type=extension
  fi

  buildpack::archive "${version}" "${buildpack_type}"
  buildpackage::create "${output}" "${buildpack_type}"
}

function usage() {
  cat <<-USAGE
package.sh --version <version> [OPTIONS]

Packages a buildpack or an extension into a buildpackage .cnb file.

OPTIONS
  --help               -h            prints the command usage
  --version <version>  -v <version>  specifies the version number to use when packaging a buildpack or an extension
  --output <output>    -o <output>   location to output the packaged buildpackage or extension artifact (default: ${ROOT_DIR}/build/buildpackage.cnb)
  --token <token>                    Token used to download assets from GitHub (e.g. jam, pack, etc) (optional)
USAGE
}

function repo::prepare() {
  util::print::title "Preparing repo..."

  rm -rf "${BUILD_DIR}"

  mkdir -p "${BIN_DIR}"
  mkdir -p "${BUILD_DIR}"

  export PATH="${BIN_DIR}:${PATH}"
}

function tools::install() {
  local token
  token="${1}"

  util::tools::pack::install \
    --directory "${BIN_DIR}" \
    --token "${token}"

  if [[ -f "${ROOT_DIR}/.libbuildpack" ]]; then
    util::tools::packager::install \
      --directory "${BIN_DIR}"
  else
    util::tools::jam::install \
      --directory "${BIN_DIR}" \
      --token "${token}"
  fi
}

function buildpack::archive() {
  local version
  version="${1}"
  buildpack_type="${2}"

  util::print::title "Packaging ${buildpack_type} into ${BUILD_DIR}/buildpack.tgz..."

  if [[ -f "${ROOT_DIR}/.libbuildpack" ]]; then
    packager \
      --uncached \
      --archive \
      --version "${version}" \
      "${BUILD_DIR}/buildpack"
  else
    jam pack \
      "--${buildpack_type}" "${ROOT_DIR}/${buildpack_type}.toml"\
      --version "${version}" \
      --output "${BUILD_DIR}/buildpack.tgz"
  fi
}

function buildpackage::create() {
  local output
  output="${1}"
  buildpack_type="${2}"

  util::print::title "Packaging ${buildpack_type}... ${output}"

  mkdir ${BUILD_DIR}/cnbdir
  tar -xvf ${BUILD_DIR}/buildpack.tgz -C ${BUILD_DIR}/cnbdir

  pack \
    "${buildpack_type}" package "${output}" \
      --path ${BUILD_DIR}/cnbdir \
      --format file

  rm -rf ${BUILD_DIR}/cnbdir
}

main "${@:-}"