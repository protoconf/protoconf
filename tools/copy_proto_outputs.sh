#!/bin/bash
protocopy() {
    file=${1#"./"}
    target_dir=$(dirname $file)
    file_to_find=$(basename ${file%".proto"}.pb.go)
    find bazel-out/$(uname | tr "A-Z" "a-z")-fastbuild -name ${file_to_find} -exec cp {} ${target_dir} \;
}

for f in $(git ls-files *.proto); do
    protocopy $f
done