#!/usr/bin/env bash
if [ $# -eq 0 ]; then
    echo "Usage: $0 <revision>"
    exit 1
fi

set -e
set -x

export GIT_TERMINAL_PROMPT=0

SCRIPT_PATH=$(dirname $(realpath $0))
git -C "$SCRIPT_PATH" fetch "https://github.com/protoconf/protoconf_example.git"

CONFIGS_TO_INSERT=$(git diff --name-only --diff-filter=ACMR "$1~1" "$1" | sed -n 's/^materialized_config\/\(.*\.materialized_JSON\)$/\1/p')
if [ -z "$CONFIGS_TO_INSERT" ]; then
    exit 0
fi

git -C "$SCRIPT_PATH" reset --hard "$1"
echo $CONFIGS_TO_INSERT | xargs protoconf insert "$SCRIPT_PATH"
