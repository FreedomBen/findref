#!/usr/bin/env bash



die ()
{
    echo "[die]: $1"
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
    mkdir -p "${dest_dir}"

    bin_name=''

    if runningLinux; then
        bin_name='findref'
        downlink_link "${dest_dir}" "$(linux_link)"
    elif runningOSX; then
        bin_name='findref'
        downlink_link "${dest_dir}" "$(mac_link)"
    else
        die 'Unsupported platform!'
    fi

    cd "${dest_dir}"

    # If there's already a findref version, remove it
    rm -f "${bin_name}"
    unzip 'findref.zip'
    rm -f 'findref.zip'
    chmod +x "${bin_name}"
}

main $@
