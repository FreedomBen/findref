#!/usr/bin/env bash


declare -r color_restore='\033[0m'
declare -r color_red='\033[0;31m'

red ()
{
    echo -e "${color_red}${1}${color_restore}"
}

declare -r color_blue='\033[0;34m'

blue ()
{
    echo -e "${color_blue}${1}${color_restore}"
}

declare -r color_cyan='\033[0;36m'

cyan ()
{
    echo -e "${color_cyan}${1}${color_restore}"
}

declare -r color_green='\033[0;32m'

green ()
{
    echo -e "${color_green}${1}${color_restore}"
}

declare -r color_brown='\033[0;33m'

brown ()
{
    echo -e "${color_brown}${1}${color_restore}"
}

declare -r color_black='\033[0;30m'

black ()
{
    echo -e "${color_black}${1}${color_restore}"
}

declare -r color_white='\033[1;37m'

white ()
{
    echo -e "${color_white}${1}${color_restore}"
}

declare -r color_purple='\033[0;35m'

purple ()
{
    echo -e "${color_purple}${1}${color_restore}"
}

declare -r color_yellow='\033[1;33m'

yellow ()
{
    echo -e "${color_yellow}${1}${color_restore}"
}

declare -r color_light_red='\033[1;31m'

light_red ()
{
    echo -e "${color_light_red}${1}${color_restore}"
}

declare -r color_dark_gray='\033[1;30m'

dark_gray ()
{
    echo -e "${color_dark_gray}${1}${color_restore}"
}

declare -r color_light_gray='\033[0;37m'

light_gray ()
{
    echo -e "${color_light_gray}${1}${color_restore}"
}

declare -r color_light_blue='\033[1;34m'

light_blue ()
{
    echo -e "${color_light_blue}${1}${color_restore}"
}

declare -r color_light_cyan='\033[1;36m'

light_cyan ()
{
    echo -e "${color_light_cyan}${1}${color_restore}"
}

declare -r color_light_green='\033[1;32m'

light_green ()
{
    echo -e "${color_light_green}${1}${color_restore}"
}

declare -r color_light_purple='\033[1;35m'

light_purple ()
{
    echo -e "${color_light_purple}${1}${color_restore}"
}


die ()
{
    echo "${color_red}[die]: ${1}${color_restore}"
    exit 1
}

runningOSX ()
{
    uname -a | grep "Darwin" > /dev/null
}

runningLinux ()
{
    uname -a | grep -i "Linux" > /dev/null
}

running32Bit ()
{
    [ "$(lshw -c cpu 2>/dev/null | grep width | awk '{print $2}')" = "32" ]
}

running64Bit ()
{
    [ "$(lshw -c cpu 2>/dev/null | grep width | awk '{print $2}')" = "64" ]
}

runningAmd64 ()
{
    [ "$(uname -m)" = 'x86_64' ]
}

running386 ()
{
    local arch="$(uname -m)"
    [ "$arch" = 'i686' ] || [ "$arch" = 'i386' ]
}

#runningArm ()
#{
#}
#
#runningArm64 ()
#{
#}

mac_link ()
{
    echo 'https://raw.githubusercontent.com/FreedomBen/findref-bin/master/latest/darwin/amd64/findref.zip'
}

linux_link ()
{
    echo 'https://raw.githubusercontent.com/FreedomBen/findref-bin/master/latest/linux/amd64/findref.zip'
}

downlink_link ()
{
    curl -o "${1}/findref.zip" "${2}"
}

main ()
{
    dest_dir="${HOME}/bin"
    [ -n "$1" ] && dest_dir="$1"
    cyan "Creating destination directory '${dest_dir}' if it doesn't exist"
    mkdir -p "${dest_dir}"

    bin_name=''

    if runningLinux; then
        cyan 'You are running linux!  Downloading findref v0.0.7 for linux...'
        bin_name='findref'
        downlink_link "${dest_dir}" "$(linux_link)"
    elif runningOSX; then
        cyan 'You are running macOS!  Downloading findref v0.0.7 for macOS...'
        bin_name='findref'
        downlink_link "${dest_dir}" "$(mac_link)"
    else
        die 'Unsupported platform!'
    fi

    cd "${dest_dir}"

    # If there's already a findref version, remove it
    cyan 'Cleaning out any old versions'
    rm -f "${bin_name}"
    cyan 'Unzipping v0.0.7'
    unzip 'findref.zip'
    cyan 'Cleaning up the zip file for v0.0.7'
    rm -f 'findref.zip'
    cyan 'Making ${bin_name} v0.0.7 executable'
    chmod +x "${bin_name}"
    cyan "All done!  If you can't run findref now, make sure that '${dest_dir}' is in your PATH"
}

main $@
