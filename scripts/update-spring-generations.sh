#!/usr/bin/env bash

set -euo pipefail

# shellcheck source=common.sh
source "$(dirname "$0")"/../../build-common/common.sh

cp "${ROOT}"/dependency/spring-generations.toml "${ROOT}"/source

cd "${ROOT}"/source

git add spring-generations.toml
git checkout -- .

git diff --cached --exit-code &> /dev/null && exit

git \
  -c user.name='Paketo Robot' \
  -c user.email='robot@paketo.io' \
  commit \
  --signoff \
  --message 'Spring Generations Update'
