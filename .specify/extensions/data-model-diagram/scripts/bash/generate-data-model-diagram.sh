#!/usr/bin/env bash
#
# Generate Mermaid ER diagrams from Spec Kit data-model.md files.
# Authored by: OpenCode

set -euo pipefail

# Find the Spec Kit project root by walking upward to the nearest .specify directory.
find_repo_root() {
    local dir="$1"

    dir="$(cd -- "$dir" 2>/dev/null && pwd -P)" || return 1

    while true; do
        if [[ -d "$dir/.specify" ]]; then
            printf '%s\n' "$dir"
            return 0
        fi

        if [[ "$dir" == "/" ]]; then
            return 1
        fi

        dir="$(dirname "$dir")"
    done
}

# Require python3 because the parser uses only the Python standard library.
require_python3() {
    if ! command -v python3 >/dev/null 2>&1; then
        printf 'ERROR: python3 is required to generate data-model-diagram.mmd\n' >&2
        return 1
    fi

    command -v python3
}

# Generate the diagram for the active feature directory and print the resolved paths.
main() {
    local script_dir extension_root repo_root common_script parser_script paths_output
    local feature_dir data_model_path diagram_path python_bin

    script_dir="$(CDPATH="" cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
    extension_root="$(CDPATH="" cd "$script_dir/../.." && pwd -P)"
    repo_root="$(find_repo_root "$extension_root")" || {
        printf 'ERROR: failed to locate the repository root from %s\n' "$extension_root" >&2
        return 1
    }

    common_script="$repo_root/.specify/scripts/bash/common.sh"
    parser_script="$extension_root/scripts/python/generate_data_model_diagram.py"

    if [[ ! -f "$common_script" ]]; then
        printf 'ERROR: Spec Kit common helpers not found at %s\n' "$common_script" >&2
        return 1
    fi

    if [[ ! -f "$parser_script" ]]; then
        printf 'ERROR: data model diagram parser not found at %s\n' "$parser_script" >&2
        return 1
    fi

    # shellcheck source=/dev/null
    source "$common_script"

    paths_output="$(get_feature_paths)" || {
        printf 'ERROR: failed to resolve the active feature directory\n' >&2
        return 1
    }

    eval "$paths_output"
    unset paths_output

    feature_dir="$FEATURE_DIR"
    data_model_path="$feature_dir/data-model.md"
    diagram_path="$feature_dir/data-model-diagram.mmd"

    if [[ ! -f "$data_model_path" ]]; then
        printf 'ERROR: data model file not found at %s\n' "$data_model_path" >&2
        return 1
    fi

    python_bin="$(require_python3)"
    "$python_bin" "$parser_script" "$data_model_path" "$diagram_path"

    printf 'INPUT=%s\n' "$data_model_path"
    printf 'OUTPUT=%s\n' "$diagram_path"
}

main "$@"
