#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
completion_script="${repo_root}/contrib/completions/findref.bash"

if [[ ! -f "${completion_script}" ]]; then
    echo "Completion script not found at ${completion_script}" >&2
    exit 1
fi

# shellcheck disable=SC1090
source "${completion_script}"

run_completion() {
    local -a words=("$@")
    COMP_WORDS=("${words[@]}")
    COMP_CWORD=$(( ${#COMP_WORDS[@]} - 1 ))
    _findref_completion
    if (( ${#COMPREPLY[@]} == 0 )); then
        return 1
    fi
    printf '%s\n' "${COMPREPLY[@]}"
}

assert_contains() {
    local description="$1"
    local needle="$2"
    local haystack="$3"
    if ! grep -Fqx -- "${needle}" <<<"${haystack}"; then
        echo "Expected '${description}' to include '${needle}', but it did not."
        echo "Full output:"
        printf '%s\n' "${haystack}"
        exit 1
    fi
    printf '✓ %s contains %s\n' "${description}" "${needle}"
}

main() {
    local output

    output="$(run_completion findref --)"
    assert_contains "option suggestions" "--exclude" "${output}"

    output="$(run_completion findref --exclude "")"
    assert_contains "exclude suggestions" ".git" "${output}"
    assert_contains "exclude suggestions" "findref.go" "${output}"

    output="$(run_completion findref foo "")"
    assert_contains "start directory suggestions" "." "${output}"

    output="$(run_completion findref foo ./ "")"
    assert_contains "filename regex suggestions" '.*\.go$' "${output}"

    output="$(run_completion findref --max-line-length "")"
    assert_contains "max-line-length suggestions" "2000" "${output}"

    echo "All completion smoke checks passed."
}

main "$@"
