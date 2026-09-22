#!/usr/bin/env bash

set -eu
set -o pipefail

readonly ROOT_DIR="$(cd "$(dirname "${0}")/.." && pwd)"
readonly BIN_DIR="${ROOT_DIR}/.bin"

# shellcheck source=SCRIPTDIR/.util/tools.sh
source "${ROOT_DIR}/scripts/.util/tools.sh"

# shellcheck source=SCRIPTDIR/.util/print.sh
source "${ROOT_DIR}/scripts/.util/print.sh"

function main {
  local archive_path buildpack_type image_ref token
  token=""

  while [[ "${#}" != 0 ]]; do
    case "${1}" in
    --archive-path | -a)
      archive_path="${2}"
      shift 2
      ;;

    --buildpack-type | -bt)
      buildpack_type="${2}"
      shift 2
      ;;

    --image-ref | -i)
      image_ref="${2}"
      shift 2
      ;;

    --token | -t)
      token="${2}"
      shift 2
      ;;

    --help | -h)
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
      ;;
    esac
  done

  if [[ -z "${image_ref:-}" ]]; then
    usage
    util::print::error "--image-ref is required"
  fi

  if [[ -z "${buildpack_type:-}" ]]; then
    usage
    util::print::error "--buildpack-type is required"
  fi

  if [[ ${buildpack_type} != "buildpack" && ${buildpack_type} != "extension" ]]; then
    usage
    util::print::error "--buildpack-type accepted values: [\"buildpack\",\"extension\"]"
  fi

  if [[ -z "${archive_path:-}" ]]; then
    util::print::info "Using default archive path: ${ROOT_DIR}/build/buildpack.tgz"
    archive_path="${ROOT_DIR}/build/buildpack.tgz"
  else
    archive_path="${archive_path}"
  fi

  repo::prepare

  tools::install "${token}"

  buildpack::publish "${image_ref}" "${buildpack_type}" "${archive_path}"
}

function usage() {
  cat <<-USAGE
Publishes a buildpack or an extension in to a registry.

OPTIONS
  -a, --archive-path <filepath>       Path to the buildpack or extension arhive (default: ${ROOT_DIR}/build/buildpack.tgz) (optional)
  -h, --help                          Prints the command usage
  -i, --image-ref <ref>               List of image reference to publish to (required)
  -bt --buildpack-type <string>       Type of buildpack to publish (accepted values: buildpack, extension) (required)
  -t, --token <token>                 Token used to download assets from GitHub (e.g. jam, pack, etc) (optional)

USAGE
}

function repo::prepare() {
  util::print::title "Preparing repo..."

  mkdir -p "${BIN_DIR}"

  export PATH="${BIN_DIR}:${PATH}"
}

function tools::install() {
  local token
  token="${1}"

  util::tools::pack::install \
    --directory "${BIN_DIR}" \
    --token "${token}"
}

function buildpack::publish() {

  local image_ref buildpack_type archive_path
  image_ref="${1}"
  buildpack_type="${2}"
  archive_path="${3}"

  util::print::title "Publishing ${buildpack_type}..."

  util::print::info "Extracting archive..."
  tmp_dir=$(mktemp -d -p $ROOT_DIR)
  tar -xvf $archive_path -C $tmp_dir

  util::print::info "Publishing ${buildpack_type} to ${image_ref}"

  pack \
    ${buildpack_type} package $image_ref \
    --path $tmp_dir \
    --format image \
    --publish

  rm -rf $tmp_dir
}

main "${@:-}"
