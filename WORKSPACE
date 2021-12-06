workspace(
    name = "protoconf",
)

# gazelle:repository_macro deps.bzl%deps
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

# http_archive(
#     name = "com_github_golang_protobuf",
#     urls = [
#         "https://github.com/golang/protobuf/archive/v1.3.1.tar.gz",
#     ],
#     strip_prefix = "protobuf-1.3.1",
#     patch_args = ["-p1"],
#     patches = [
#         # "@//third_party:com_github_golang_protobuf-gazelle.patch",
#         # "@io_bazel_rules_go//third_party:com_github_golang_protobuf-extras.patch",
#         "@//third_party:protobuf_fix_any_indentation.patch",
#     ],
#     sha256 = "3f3a6123054a9847093c119895f1660612f301fe95358f3a6a1a33fd0933e6cf",
# )
# gazelle args: -go_prefix github.com/golang/protobuf -proto disable_global

# http_archive(
#     name = "io_bazel_rules_go",
#     urls = [
#         "https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/rules_go/releases/download/v0.20.3/rules_go-v0.20.3.tar.gz",
#         "https://github.com/bazelbuild/rules_go/releases/download/v0.20.3/rules_go-v0.20.3.tar.gz",
#     ],
#     sha256 = "e88471aea3a3a4f19ec1310a55ba94772d087e9ce46e41ae38ecebe17935de7b",
# )

# http_archive(
#     name = "bazel_gazelle",
#     urls = [
#         "https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/bazel-gazelle/releases/download/v0.19.1/bazel-gazelle-v0.19.1.tar.gz",
#         "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.19.1/bazel-gazelle-v0.19.1.tar.gz",
#     ],
#     sha256 = "86c6d481b3f7aedc1d60c1c211c6f76da282ae197c3b3160f54bd3a8f847896f",
# )

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "7904dbecbaffd068651916dce77ff3437679f9d20e1a7956bff43826e7645fcc",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.25.1/rules_go-v0.25.1.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.25.1/rules_go-v0.25.1.tar.gz",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "222e49f034ca7a1d1231422cdb67066b885819885c356673cb1f72f748a3c9d4",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.22.3/bazel-gazelle-v0.22.3.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.22.3/bazel-gazelle-v0.22.3.tar.gz",
    ],
)

load(
    "@io_bazel_rules_go//go:deps.bzl",
    "go_register_toolchains",
    "go_rules_dependencies",
)
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")

go_rules_dependencies()

go_register_toolchains(
    nogo = "@//:protoconf_nogo",
    version = "1.16",
)

gazelle_dependencies()

http_archive(
    name = "bazel_skylib",
    sha256 = "1dde365491125a3db70731e25658dfdd3bc5dbdfd11b840b3e987ecf043c7ca0",
    url = "https://github.com/bazelbuild/bazel-skylib/releases/download/0.9.0/bazel_skylib-0.9.0.tar.gz",
)

load("@bazel_skylib//:workspace.bzl", "bazel_skylib_workspace")

bazel_skylib_workspace()

http_archive(
    name = "rules_proto",
    sha256 = "602e7161d9195e50246177e7c55b2f39950a9cf7366f74ed5f22fd45750cd208",
    strip_prefix = "rules_proto-97d8af4dc474595af3900dd85cb3a29ad28cc313",
    urls = [
        # Master branch as of 2019-08-01
        "https://mirror.bazel.build/github.com/bazelbuild/rules_proto/archive/97d8af4dc474595af3900dd85cb3a29ad28cc313.tar.gz",
        "https://github.com/bazelbuild/rules_proto/archive/97d8af4dc474595af3900dd85cb3a29ad28cc313.tar.gz",
    ],
)

load(
    "@rules_proto//proto:repositories.bzl",
    "rules_proto_dependencies",
    "rules_proto_toolchains",
)

rules_proto_dependencies()

rules_proto_toolchains()

# Docker
http_archive(
    name = "containerregistry",
    patch_args = ["-p1"],
    patches = ["//third_party:containerregistry.patch"],
    sha256 = "a0c01fcc11db848212f8b11d89df168361f99a31eb7373ff60ce50c5d05cd74b",
    strip_prefix = "containerregistry-0.0.38",
    urls = ["https://github.com/google/containerregistry/archive/v0.0.38.tar.gz"],
)

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "59536e6ae64359b716ba9c46c39183403b01eabfbd57578e84398b4829ca499a",
    strip_prefix = "rules_docker-0.22.0",
    urls = [
        "https://github.com/bazelbuild/rules_docker/releases/download/v0.22.0/rules_docker-v0.22.0.tar.gz",
    ],
)

load(
    "@io_bazel_rules_docker//repositories:repositories.bzl",
    container_repositories = "repositories",
)

container_repositories()

load("@io_bazel_rules_docker//repositories:deps.bzl", container_deps = "deps")

container_deps()

load(
    "@io_bazel_rules_docker//go:image.bzl",
    _go_image_repos = "repositories",
)

_go_image_repos()

load(
    "@com_github_bazelbuild_buildtools//buildifier:deps.bzl",
    "buildifier_dependencies",
)

buildifier_dependencies()

load("@//:deps.bzl", "deps")

deps()

git_repository(
    name = "graknlabs_bazel_distribution",
    commit = "c24ce26390d7e818a6f16a0d2839ed5795ced3dc",  # sync-marker: do not remove this comment, this is used for sync-dependencies by @graknlabs_bazel_distribution
    remote = "https://github.com/graknlabs/bazel-distribution",
)

load("@graknlabs_bazel_distribution//common:dependencies.bzl", "bazelbuild_rules_pkg")

bazelbuild_rules_pkg()

load("@graknlabs_bazel_distribution//github:dependencies.bzl", "tcnksm_ghr")

tcnksm_ghr()

http_archive(
    name = "build_bazel_rules_nodejs",
    sha256 = "0fa2d443571c9e02fcb7363a74ae591bdcce2dd76af8677a95965edf329d778a",
    urls = [
        "https://github.com/bazelbuild/rules_nodejs/releases/download/3.6.0/rules_nodejs-3.6.0.tar.gz",
    ],
)

load("@build_bazel_rules_nodejs//:index.bzl", "node_repositories")

node_repositories(package_json = ["//:package.json"])

http_archive(
    name = "io_buildbuddy_buildbuddy_toolchain",
    sha256 = "a97a87d72417dc4ddd5b434ed4eef6a09fcf7b1a3e87087e3814f50a7c5762d8",
    strip_prefix = "buildbuddy-toolchain-b2f5e7e3b126c6d7cf243227147478c0959bfc95",
    urls = [
        "https://github.com/buildbuddy-io/buildbuddy-toolchain/archive/b2f5e7e3b126c6d7cf243227147478c0959bfc95.tar.gz",
    ],
)

load("@io_buildbuddy_buildbuddy_toolchain//:deps.bzl", "buildbuddy_deps")

buildbuddy_deps()

load("@io_buildbuddy_buildbuddy_toolchain//:rules.bzl", "buildbuddy")

buildbuddy(name = "buildbuddy_toolchain")
