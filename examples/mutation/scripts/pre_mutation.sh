#!/usr/bin/env bash
set -e
set -x

export GIT_TERMINAL_PROMPT=0

SCRIPT_PATH=$(dirname $(realpath $0))
git -C "$SCRIPT_PATH" fetch "https://$1@github.com/protoconf/protoconf_example.git"
git -C "$SCRIPT_PATH" reset --hard origin/master
