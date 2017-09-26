#!/usr/bin/env bash

set -e

declare -r color_restore='\033[0m'
declare -r color_black='\033[0;30m'
declare -r color_red='\033[0;31m'
declare -r color_green='\033[0;32m'
declare -r color_brown='\033[0;33m'
declare -r color_blue='\033[0;34m'
declare -r color_purple='\033[0;35m'
declare -r color_cyan='\033[0;36m'
declare -r color_light_gray='\033[0;37m'
declare -r color_dark_gray='\033[1;30m'
declare -r color_light_red='\033[1;31m'
declare -r color_light_green='\033[1;32m'
declare -r color_yellow='\033[1;33m'
declare -r color_light_blue='\033[1;34m'
declare -r color_light_purple='\033[1;35m'
declare -r color_light_cyan='\033[1;36m'
declare -r color_white='\033[1;37m'

VERSION=0.0.1

# See: https://stackoverflow.com/a/30068222/2062384 for list of valid targets
OSES=("linux" "windows" "darwin")
ARCHES=("amd64" "386")

echo -e "${color_cyan}Building findref version '${VERSION}' for OSes '${OSES[@]}', and arches '${ARCHES[@]}'...${color_restore}"

root_dir=findref-bin/${VERSION}
for os in ${OSES[@]}; do
    for arch in ${ARCHES[@]}; do
        echo -e "${color_cyan}Building version ${VERSION} for OS ${os}, arch ${arch}${color_restore}"
        docker run \
          --rm \
          --volume "$PWD":/usr/src/findref \
          --workdir /usr/src/findref \
          --env GOOS=${os} \
          --env GOARCH=${arch} \
          golang:1.9-alpine go build
        mkdir -p ${root_dir}/${os}/${arch}
        filename='findref'
        [ "$os" = "windows" ] && filename='findref.exe'
        mv --force "$filename" ${root_dir}/${os}/${arch}/
    done
done

echo -e "${color_cyan}Done!${color_restore}"
