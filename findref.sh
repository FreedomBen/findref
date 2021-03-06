#!/usr/bin/env bash

# vim: set filetype=sh ts=4 sw=4 sts=4 expandtab :

LINE_CHAR_LIMIT=200  # don't match on crazy long lines
DEBUG_MODE=0

runningOSX ()
{
    uname -a | grep "Darwin" >/dev/null
}

# Uncomment the next line to turn on debug output
# DEBUG_MODE=1

# have special syntax for Macs 
if runningOSX; then
    TEMP_FILE=$(mktemp /tmp/findref.XXXXXX)
else
    TEMP_FILE=$(mktemp)
fi

ABS_PATH_COMMAND="realpath"

if runningOSX; then
    realpath () {
        [[ $1 = /* ]] && echo "$1" || echo "$PWD/${1#./}"
    }
fi

IN_GIT_REPO=0

color_light_blue='\033[1;34m'
color_restore='\033[0m'

cur_millis_since_epoch ()
{
    echo $(date +%3N)
}

cur_secs_since_epoch ()
{
    echo $(date +%s)
}

cur_secs_millis_since_epoch ()
{
    echo $(date +%s%3N)
}

START_DEBUG_SECS="$(cur_secs_since_epoch)"

DEBUG ()
{
    if (( $DEBUG_MODE )); then
        echo -e "${color_light_blue}[*] $(( $(cur_secs_since_epoch) - $START_DEBUG_SECS)).$(cur_millis_since_epoch) - DEBUG: $1${color_restore}"
    fi
}

print_usage () 
{
    echo "Usage: findref [-f|--fast (skip git ignore list)] [-i|--ignore-case] \"what text (RegEx) to look for\" \"[starting location (root dir)]\" \"[filenames to check (must match pattern)]\"";
}


if [ -z "$1" ]; then
    print_usage
    exit;
fi


# Need an absolute path - prefer realpath to readlink if we have it installed
if ! runningOSX && ! $(which realpath > /dev/null 2>&1); then
    ABS_PATH_COMMAND="readlink -f"
fi


# skip this if given --fast or -f
# TODO: Checking $3 here seems like a bug because if the -f flag is present the filename would be $4...
if [ -n "$3" ]; then
    DEBUG "Skipping git ignore check because we have filename specified ($3)"
elif [[ ! $@ =~ -f ]]; then
    DEBUG "Checking if we're in a git repo"
    # Determine if we are inside of a Git repo so we can consider the .gitignore file
    prevDir="$(pwd)"

    while true; do
        if [ -d ".git" ]; then
            IN_GIT_REPO=1
            break
        fi

        if [ "$(pwd)" = "/" ]; then
            break;
        fi

        cd ..
    done

    cd $prevDir

    if (( DEBUG_MODE )); then
        if (( IN_GIT_REPO )); then
            DEBUG "In Git repo"
        else
            DEBUG "Not in a Git repo"
        fi
    fi
else
    # need to shift away the --fast or -f
    DEBUG "Fast mode turned on, not checking for git repo"
    shift
fi

DEBUG "Checking for ignore case setting"
# If there are no capital letters, be case insensitive even if not passed the flag
if [ "$1" = "-i" ] || [ "$1" = "--ignore-case" ]; then
    ignore_case='--ignore-case'
    shift
    DEBUG "Ignoring case due to explicit flag"
elif [[ ! $1 =~ [A-Z] ]]; then
    ignore_case='--ignore-case'
    DEBUG "Ignoring case cause pattern contains no upper case letters (smartcase)"
else
    ignore_case=''
    DEBUG "Not ignoring case"
fi

if [ -z "$2" ] || [ ! -e "$2" ]; then
    where_abs="$(pwd)";
    where_rel=""
else
    # strip off trailing / if it exists
    where_rel="$(echo $2 | sed -e 's|/$||g')"

    # prefer realpath if installed, otherwise use readlink to get absolute path
    where_abs="$(eval $ABS_PATH_COMMAND $where_rel)"
fi

DEBUG "Determined to search from $where_abs. rel path is $where_rel"

if [ -z "$1" ]; then
    print_usage
    exit;
else
    what=$(echo "$1" | sed 's/ /\\s/g');
fi

DEBUG "Determined to search for lines matching $what"

if [ -z "$3" ]; then
    filename="";
else
    filename='-iname "$3"';
fi;

DEBUG "Determined to search filenames matching $filename"

DEBUG "Populating the TEMP_FILE $TEMP_FILE with filenames to search"
if (( $IN_GIT_REPO )); then
    DEBUG "In Git repo, so using git's list of files"
    for i in $(git ls-files --ignore --exclude=${where_rel}*); do
        [ -f "$i" ] && echo "$i" >> "$TEMP_FILE"
    done
    for i in $(git ls-files --others --exclude-standard); do
        [ -f "$i" ] && echo "$i" >> "$TEMP_FILE"
    done
else
    DEBUG "Not in git repo, finding all filenames matching pattern $filename"
    eval find "$where_abs" -type f "$filename" > $TEMP_FILE;
fi

DEBUG "Done populating the TEMP_FILE"

numlines=$(cat $TEMP_FILE | wc -l);
DEBUG "Grepping $numlines files"

for (( i=1; ((1)); i+=1000 ))
do
    if (( numlines > 1000 )); then
        topBoundary=1000;
    else
        topBoundary=numlines;
    fi;
    sed -n $i,$(( i + topBoundary ))p $TEMP_FILE | sed 's/ /\\ /g' | sed "s/'//g" | sed "/^.\{${LINE_CHAR_LIMIT}\}..*/d" | xargs grep $ignore_case --extended-regexp --color --binary-files=without-match --directories=skip --devices=skip --line-number $what;
    numlines=$(( numlines - topBoundary ));
    if (( numlines <= 0 )); then
        break;
    fi;
done;

DEBUG "Done grepping the files"

# clean up the temp files if they exist
if [ -f "$TEMP_FILE" ]; then
    DEBUG "Deleting TEMP_FILE $TEMP_FILE"
    rm -f "$TEMP_FILE";
fi

DEBUG "All done!"
