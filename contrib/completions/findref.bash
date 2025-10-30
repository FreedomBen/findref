#!/usr/bin/env bash
# Bash completion support for findref.

__findref_safe_compopt() {
    if declare -F compopt >/dev/null 2>&1; then
        compopt "$@" 2>/dev/null || true
    fi
}

_findref_completion() {
    local cur prev words cword
    if declare -F _init_completion >/dev/null 2>&1; then
        _init_completion -n '=' || true
    fi
    if [[ -z ${words+x} ]]; then
        words=("${COMP_WORDS[@]}")
    fi
    if [[ -z ${cword+x} ]]; then
        cword=$COMP_CWORD
    fi
    if [[ -z ${cur+x} ]]; then
        cur="${words[cword]}"
    fi
    if [[ -z ${prev+x} ]]; then
        if (( cword > 0 )); then
            prev="${words[cword-1]}"
        else
            prev=""
        fi
    fi

    COMPREPLY=()

    local -a opts_no_value=(
        -a --all
        -s --stats
        -d --debug
        -h --hidden
        -v --version
        -n --no-color
        -m --match-case
        -i --ignore-case
        -f --filename-only
        -x --no-max-line-length
        --help
    )
    local -a opts_with_value=(
        -l --max-line-length
        -e --exclude
    )
    local -a exclude_defaults=(.git .svn .hg .bzr CVS vendor node_modules build dist out coverage)
    local -a regex_suggestions=('".*\\.go$"' '".*\\.py$"' '".*\\.(js|ts)$"' '".*\\.(c|h)$"')
    local -a match_examples=('"TODO"' '"TODO|FIXME"' '"(?i)http"')

    local expecting_value=""
    case "$prev" in
        --exclude|-e)
            expecting_value="exclude"
            ;;
        --max-line-length|-l)
            expecting_value="max-length"
            ;;
    esac

    if [[ $cur == --exclude=* ]]; then
        expecting_value="exclude"
        prev="--exclude"
    elif [[ $cur == --max-line-length=* ]]; then
        expecting_value="max-length"
        prev="--max-line-length"
    elif [[ $cur == -l=* ]]; then
        expecting_value="max-length"
        prev="-l"
    fi

    if [[ -n $expecting_value ]]; then
        case "$expecting_value" in
            exclude)
                local prefix=""
                local value="$cur"
                if [[ $cur == --exclude=* ]]; then
                    prefix="--exclude="
                    value="${cur#*=}"
                elif [[ $cur == -e=* ]]; then
                    prefix="-e="
                    value="${cur#*=}"
                else
                    value="$cur"
                fi
                local -a suggestions=()
                for default_dir in "${exclude_defaults[@]}"; do
                    if [[ -z $value || $default_dir == "$value"* ]]; then
                        suggestions+=("$default_dir")
                    fi
                done
                while IFS= read -r line; do
                    suggestions+=("$line")
                done < <(compgen -d -- "$value")
                if [[ -n $prefix ]]; then
                    local -a prefixed=()
                    local unique
                    unique=$(printf '%s\n' "${suggestions[@]}" | awk 'NF' | LC_ALL=C sort -u)
                    while IFS= read -r item; do
                        prefixed+=("$prefix$item")
                    done <<<"$unique"
                    COMPREPLY=("${prefixed[@]}")
                else
                    COMPREPLY=($(printf '%s\n' "${suggestions[@]}" | awk 'NF' | LC_ALL=C sort -u))
                    __findref_safe_compopt -o filenames -o nospace
                fi
                return 0
                ;;
            max-length)
                local prefix=""
                local value="$cur"
                if [[ $cur == --max-line-length=* ]]; then
                    prefix="--max-line-length="
                    value="${cur#*=}"
                elif [[ $cur == -l=* ]]; then
                    prefix="-l="
                    value="${cur#*=}"
                fi
                local -a numbers=(120 200 500 1000 2000)
                local -a matches=()
                for n in "${numbers[@]}"; do
                    if [[ -z $value || $n == "$value"* ]]; then
                        matches+=("$n")
                    fi
                done
                if [[ -n $prefix ]]; then
                    local -a prefixed=()
                    for m in "${matches[@]}"; do
                        prefixed+=("$prefix$m")
                    done
                    COMPREPLY=("${prefixed[@]}")
                else
                    COMPREPLY=("${matches[@]}")
                fi
                return 0
                ;;
        esac
    fi

    if [[ $cur == --* ]]; then
        COMPREPLY=($(compgen -W "${opts_no_value[*]} ${opts_with_value[*]} -- --version" -- "$cur"))
        return 0
    fi

    if [[ $cur == -* ]]; then
        COMPREPLY=($(compgen -W "${opts_no_value[*]} ${opts_with_value[*]}" -- "$cur"))
        return 0
    fi

    local after_double_dash=0
    local positional_index=0
    local total_words=${#words[@]}
    local token
    local pending_option=""
    for ((i=1; i<total_words; i++)); do
        token="${words[i]}"
        if (( i == cword )); then
            break
        fi
        if [[ $after_double_dash -eq 1 ]]; then
            (( positional_index++ ))
            continue
        fi
        if [[ -n $pending_option ]]; then
            pending_option=""
            continue
        fi
        if [[ $token == -- ]]; then
            after_double_dash=1
            continue
        fi
        case "$token" in
            --exclude|--max-line-length|-e|-l)
                pending_option="$token"
                continue
                ;;
            --exclude=*|--max-line-length=*|-l=*)
                continue
                ;;
            -*)
                continue
                ;;
            *)
                (( positional_index++ ))
                ;;
        esac
    done

    if [[ $after_double_dash -eq 1 ]]; then
        if [[ -n $cur ]]; then
            return 0
        fi
        return 0
    fi

    case "$positional_index" in
        0)
            COMPREPLY=($(compgen -W "${match_examples[*]}" -- "$cur"))
            return 0
            ;;
        1)
            __findref_safe_compopt -o filenames -o nospace
            local -a dirs=()
            while IFS= read -r candidate; do
                dirs+=("$candidate")
            done < <(compgen -d -- "$cur")
            if [[ -z $cur ]]; then
                dirs+=(".")
            fi
            COMPREPLY=("${dirs[@]}")
            return 0
            ;;
        2)
            COMPREPLY=($(compgen -W "${regex_suggestions[*]}" -- "$cur"))
            return 0
            ;;
        *)
            return 0
            ;;
    esac
}

complete -F _findref_completion -o bashdefault -o default findref
