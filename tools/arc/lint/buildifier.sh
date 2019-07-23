#!/bin/bash -e

bazel run --script_path=/tmp/buildifier buildifier

/tmp/buildifier < $1
