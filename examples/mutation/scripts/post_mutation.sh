#!/usr/bin/env bash
set -e
set -x

SCRIPTPATH=$(dirname $(realpath $0))
protoconf compile "$SCRIPTPATH/../../protoconf/"
git -C "$SCRIPTPATH" add "$SCRIPTPATH/../../protoconf/"

USER=$(echo -n $1 | cut -d: -f1)
git -C "$SCRIPTPATH" commit -m "Mutated with API" --author "$USER"
git -C "$SCRIPTPATH" push "https://$1@github.com/protoconf/protoconf.git" master
