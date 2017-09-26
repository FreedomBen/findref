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
    echo "https://raw.githubusercontent.com/FreedomBen/findref-bin/master/current_version/darwin/amd64/findref"
}

linux_link ()
{
    echo "https://raw.githubusercontent.com/FreedomBen/findref-bin/master/current_version/linux/amd64/findref"
}

downlink_link ()
{
    curl -o "${1}/findref" "${2}"
}

main ()
{
    DEST_DIR="$HOME/bin"
    [ -n "$1" ] && DEST_DIR="$1"

    if runningLinux; then
        downlink_link "${DEST_DIR} $(linux_link)"
    elif runningOSX; then
        downlink_link "${DEST_DIR} $(mac_link)"
    else
        die "Unsupported platform!"
    fi

    chmod +x "$DEST_DIR/findref"
}

main $@