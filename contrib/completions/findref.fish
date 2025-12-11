# Fish completion support for findref.

if not set -q __fish_findref_exclude_defaults
    # Keep in sync with defaultExcludeDirs in settings.go.
    set -g __fish_findref_exclude_defaults \
        .git .svn .hg .bzr CVS vendor node_modules build dist out coverage \
        package-lock.json yarn.lock pnpm-lock.yaml bun.lockb composer.lock Gemfile.lock \
        mix.lock Cargo.lock Pipfile.lock poetry.lock Podfile.lock go.sum gradle.lockfile
end

if not set -q __fish_findref_match_examples
    set -g __fish_findref_match_examples '"panic"' '"(?i)password"' '"http.NewRequest"'
end

if not set -q __fish_findref_filename_regexes
    set -g __fish_findref_filename_regexes \
        '".*\.go$"' '".*\.py$"' '".*\.(js|ts)$"' '".*\.(c|h)$"'
end

if not set -q __fish_findref_max_line_lengths
    set -g __fish_findref_max_line_lengths 120 200 500 1000 2000
end

if not set -q __fish_findref_write_config_targets
    set -g __fish_findref_write_config_targets local global
end

function __fish_findref_positional_count
    set -l tokens (commandline -opc)
    if test (count $tokens) -eq 0
        echo 0
        return
    end
    set -e tokens[1]

    set -l count 0
    set -l expect_value 0
    set -l after_dd 0
    for token in $tokens
        if test $after_dd -eq 1
            set count (math "$count + 1")
            continue
        end
        if test $expect_value -eq 1
            set expect_value 0
            continue
        end
        switch $token
            case '--'
                set after_dd 1
                continue
            case '-e' '--exclude' '-l' '--max-line-length' '--write-config'
                set expect_value 1
                continue
            case '--exclude=*' '-e=*' '--max-line-length=*' '-l=*' '--write-config=*'
                continue
            case '-*'
                continue
            case '*'
                set count (math "$count + 1")
        end
    end
    echo $count
end

function __fish_findref_needs_match_regex
    if not test (__fish_findref_positional_count) -eq 0
        return 1
    end
    set -l token (commandline -ct)
    if string match -q -- '-*' "$token"
        return 1
    end
    return 0
end

function __fish_findref_needs_start_dir
    test (__fish_findref_positional_count) -eq 1
end

function __fish_findref_needs_filename_regex
    test (__fish_findref_positional_count) -eq 2
end

function __fish_findref_match_examples
    set -l label '[match regex] search file contents'
    for example in $__fish_findref_match_examples
        printf '%s\t%s\n' "$example" "$label"
    end
end

function __fish_findref_filename_regexes
    set -l label '[filename regex] filter files to scan'
    for regex in $__fish_findref_filename_regexes
        printf '%s\t%s\n' "$regex" "$label"
    end
end

function __fish_findref_max_line_lengths
    printf '%s\n' $__fish_findref_max_line_lengths
end

function __fish_findref_start_dirs
    set -l token (commandline -ct)
    if test -z "$token"
        echo .
    end
    __fish_complete_directories
end

function __fish_findref_exclude_suggestions
    printf '%s\n' $__fish_findref_exclude_defaults
    __fish_complete_path
end

complete -c findref -s a -l all -f -d 'Aggressively search for matches (implies -i and -h)'
complete -c findref -s s -l stats -f -d 'Track basic statistics and print them on exit'
complete -c findref -s d -l debug -f -d 'Enable debug mode'
complete -c findref -s h -l hidden -f -d 'Include hidden files and directories'
complete -c findref -s v -l version -f -d 'Print current version and exit'
complete -c findref -s n -l no-color -f -d 'Disable colorized output'
complete -c findref -s m -l match-case -f -d 'Match regex case explicitly'
complete -c findref -s i -l ignore-case -f -d 'Ignore regex case (override smart-case)'
complete -c findref -s f -l filename-only -f -d 'Print only filenames that contain matches'
complete -c findref -s x -l no-max-line-length -f -d 'Remove the maximum line length limit'
complete -c findref -l write-config -fr -d 'Generate a default config file and exit' -a '$__fish_findref_write_config_targets'
complete -c findref -l help -f -d 'Show usage information'

complete -c findref -s l -l max-line-length -fr -d 'Set maximum line length' -a '(__fish_findref_max_line_lengths)'
complete -c findref -s e -l exclude -fr -d 'Exclude matching directories or files' -a '(__fish_findref_exclude_suggestions)'

complete -c findref -n '__fish_findref_needs_match_regex' -f -d 'Regular expression to search for' -a '(__fish_findref_match_examples)'
complete -c findref -n '__fish_findref_needs_start_dir' -f -d 'Directory to start searching from' -a '(__fish_findref_start_dirs)'
complete -c findref -n '__fish_findref_needs_filename_regex' -f -d 'Filename filter regular expression' -a '(__fish_findref_filename_regexes)'
