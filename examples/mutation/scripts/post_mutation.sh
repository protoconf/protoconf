#!/usr/bin/env bash
set -e
set -x

export GIT_TERMINAL_PROMPT=0

SCRIPT_PATH=$(dirname $(realpath $0))
protoconf compile "$SCRIPT_PATH/../../protoconf/"
git -C "$SCRIPT_PATH" add "$SCRIPT_PATH/../../protoconf/"

USER=$(echo -n $1 | cut -d: -f1)
git -C "$SCRIPT_PATH" commit -m "Mutated with API" --author "$USER"
git -C "$SCRIPT_PATH" push "https://$1@github.com/protoconf/protoconf_example.git" master
