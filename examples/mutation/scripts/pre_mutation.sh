#!/usr/bin/env bash
set -e
set -x

SCRIPTPATH=$(dirname $(realpath $0))
git -C "$SCRIPTPATH" fetch "https://$1@github.com/protoconf/protoconf.git"
git -C "$SCRIPTPATH" reset --hard origin/master
