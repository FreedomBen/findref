#compdef findref

# Keep in sync with defaultExcludeDirs in settings.go.
typeset -ga _findref_exclude_defaults=(
  .git
  .svn
  .hg
  .bzr
  CVS
  vendor
  node_modules
  build
  dist
  out
  coverage
  package-lock.json
  yarn.lock
  pnpm-lock.yaml
  bun.lockb
  composer.lock
  Gemfile.lock
  mix.lock
  Cargo.lock
  Pipfile.lock
  poetry.lock
  Podfile.lock
  go.sum
  gradle.lockfile
)

typeset -ga _findref_match_examples=(
  '"panic"'
  '"(?i)password"'
  '"http.NewRequest"'
)

typeset -ga _findref_filename_regexes=(
  '".*\.go$"'
  '".*\.py$"'
  '".*\.(js|ts)$"'
  '".*\.(c|h)$"'
)

typeset -ga _findref_max_line_lengths=(120 200 500 1000 2000)

_findref_complete_excludes() {
  emulate -L zsh
  (( $#_findref_exclude_defaults )) && \
    compadd -Q -X 'Common excludes' -- "${_findref_exclude_defaults[@]}"
  local ret=1
  _path_files -/ && ret=0
  _path_files && ret=0
  return ret
}

_findref_complete_lengths() {
  emulate -L zsh
  compadd -Q -- "${_findref_max_line_lengths[@]}"
}

_findref_complete_match_examples() {
  emulate -L zsh
  if [[ $PREFIX == -* ]]; then
    return 1
  fi
  compadd -Q -X '[match regex] search file contents' -- "${_findref_match_examples[@]}"
}

_findref_complete_filename_regexes() {
  emulate -L zsh
  compadd -Q -X '[filename regex] filter files to scan' -- "${_findref_filename_regexes[@]}"
}

_findref_complete_start_dir() {
  emulate -L zsh
  local ret=1
  if [[ -z $PREFIX ]]; then
    compadd -Q -- .
    ret=0
  fi
  _path_files -/ && ret=0
  return ret
}

_findref() {
  emulate -L zsh
  local state ret=1
  _arguments -C \
    '(-a --all)'{-a,--all}'[Aggressively search for matches and disable default excludes]' \
    '(-s --stats)'{-s,--stats}'[Track basic statistics and print them on exit]' \
    '(-d --debug)'{-d,--debug}'[Enable debug mode]' \
    '(-h --hidden)'{-h,--hidden}'[Include hidden files and directories]' \
    '(-v --version)'{-v,--version}'[Print current version and exit]' \
    '(-n --no-color)'{-n,--no-color}'[Disable colorized output]' \
    '(-m --match-case)'{-m,--match-case}'[Match regex case explicitly]' \
    '(-i --ignore-case)'{-i,--ignore-case}'[Ignore regex case (override smart-case)]' \
    '(-f --filename-only)'{-f,--filename-only}'[Output only filenames containing matches]' \
    '(-x --no-max-line-length)'{-x,--no-max-line-length}'[Remove the maximum line length limit]' \
    '(-l --max-line-length)'{-l+,--max-line-length=-}'[Set maximum line length in characters]:max line length:_findref_complete_lengths' \
    '(-e --exclude)'{-e+,--exclude=-}'[Exclude matching directories or files (repeatable)]:exclude entry:_findref_complete_excludes' \
    '--help[Show usage information]' \
    '1:match regex:_findref_complete_match_examples' \
    '2:start directory:_findref_complete_start_dir' \
    '3:filename regex:_findref_complete_filename_regexes' \
    '*:: :->rest' && ret=0

  case $state in
    rest)
      _default && ret=0
      ;;
  esac
  return ret
}

_findref "$@"
