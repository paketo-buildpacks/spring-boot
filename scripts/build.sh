#!/usr/bin/env bash

set -euo pipefail

GOOS="linux" go build -ldflags='-s -w' -o bin/helper github.com/paketo-buildpacks/spring-boot/cmd/helper
GOOS="linux" go build -ldflags='-s -w' -o bin/main github.com/paketo-buildpacks/spring-boot/cmd/main

strip bin/helper bin/main
upx -q -9 bin/helper bin/main

ln -fs main bin/build
ln -fs main bin/detect

