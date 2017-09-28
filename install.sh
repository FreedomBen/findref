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
    echo "https://raw.githubusercontent.com/FreedomBen/findref-bin/master/latest/darwin/amd64/findref.zip"
}

linux_link ()
{
    echo "https://raw.githubusercontent.com/FreedomBen/findref-bin/master/latest/linux/amd64/findref.zip"
}

downlink_link ()
{
    curl -o "${1}/findref.zip" "${2}"
}

main ()
{
    DEST_DIR="$HOME/bin"
    [ -n "$1" ] && DEST_DIR="$1"
    mkdir -p $DEST_DIR

    if runningLinux; then
        downlink_link "${DEST_DIR}" "$(linux_link)"
    elif runningOSX; then
        downlink_link "${DEST_DIR}" "$(mac_link)"
    else
        die "Unsupported platform!"
    fi

    cd $DEST_DIR
    unzip findref.zip
    rm findref.zip
    chmod +x "$DEST_DIR/findref"
}

main $@
