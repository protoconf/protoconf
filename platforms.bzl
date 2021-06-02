load(
    "@graknlabs_bazel_distribution//common:rules.bzl",
    "assemble_targz",
    "assemble_versioned",
    "assemble_zip",
)
load("@io_bazel_rules_go//go:def.bzl", "go_binary")
PROTOCONF_PLATFORMS = {
    "darwin": ["amd64", "arm64"],
    "linux": ["386", "amd64", "arm", "arm64"],
    "freebsd": ["386", "amd64", "arm", "arm64"],
    "openbsd": ["386", "amd64"],
    "solaris": ["amd64"],
    "windows": ["386", "amd64"],
}


def versions():
    targets = []
    for os in PROTOCONF_PLATFORMS:
        for arch in PROTOCONF_PLATFORMS[os]:
            go_binary(
                name="protoconf-%s-%s__bin" % (os, arch),
                embed=["//cmd/protoconf:go_default_library"],
                goarch=arch,
                goos=os,
                pure="on",
                visibility=["//visibility:public"],
            )
            assemble_targz(
                name="protoconf-%s-%s" % (os, arch),
                additional_files={
                    ":protoconf-%s-%s__bin" % (os, arch): "./protoconf.exe"
                    if os == "windows"
                    else "./protoconf",
                },
                output_filename="",
            )
            if os == "windows":
                assemble_zip(
                    name="protoconf-%s-%s__zip" % (os, arch),
                    output_filename="protoconf-%s-%s" % (os, arch),
                    targets=[":protoconf-%s-%s" % (os, arch)],
                )
                targets.append("protoconf-%s-%s__zip" % (os, arch))
            else:
                targets.append("protoconf-%s-%s" % (os, arch))
    return targets