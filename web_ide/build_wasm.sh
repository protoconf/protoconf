#!/usr/bin/env bash
set -e
cd "$(dirname "$0")"/../
bazel build --compilation_mode=opt //compiler/wasm 2>&1 | grep Target -A1 | grep bin | xargs -I{} cp -f {} web_ide/compiler.wasm
