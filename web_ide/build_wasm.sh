#!/usr/bin/env bash
set -e
cd "$(dirname "$0")"/../
bazel build --compilation_mode=opt //compiler/wasm 2>&1 | grep Target -A1 | grep bin | xargs -I{} cp -f {} web_ide/compiler.wasm

# Embed a Subresource Integrity hash so the browser verifies compiler.wasm hasn't been tampered with.
WASM_SRI=$(openssl dgst -sha384 -binary web_ide/compiler.wasm | openssl base64 -A)
sed -i.bak "s/sha384-[A-Za-z0-9+\/=]*/sha384-${WASM_SRI}/" web_ide/index.html
rm -f web_ide/index.html.bak
