#!/bin/bash -e

bazel run --script_path=/tmp/buildifier buildifier

/tmp/buildifier -lint fix -warnings all -path $1 < $1
