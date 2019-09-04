workspace(name = "protoconf")

load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

git_repository(
    name = "com_github_golang_protobuf",
    commit = "c823c79ea1570fb5ff454033735a8e68575d1d0f",  # v1.3.0, as of 2019-03-03
    patch_args = ["-p1"],
    patches = [
        "//third_party:com_github_golang_protobuf-gazelle.patch",
        "@io_bazel_rules_go//third_party:com_github_golang_protobuf-extras.patch",
        "//third_party:protobuf_fix_any_indentation.patch",
    ],
    remote = "https://github.com/golang/protobuf",
    shallow_since = "1549405252 -0800",
)
# gazelle args: -go_prefix github.com/golang/protobuf -proto disable_global

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "313f2c7a23fecc33023563f082f381a32b9b7254f727a7dd2d6380ccc6dfe09b",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/rules_go/releases/download/0.19.3/rules_go-0.19.3.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/0.19.3/rules_go-0.19.3.tar.gz",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "7fc87f4170011201b1690326e8c16c5d802836e3a0d617d8f75c3af2b23180c4",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/bazel-gazelle/releases/download/0.18.2/bazel-gazelle-0.18.2.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/0.18.2/bazel-gazelle-0.18.2.tar.gz",
    ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains(nogo = "@//:protoconf_nogo")

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

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

load("@rules_proto//proto:repositories.bzl", "rules_proto_dependencies", "rules_proto_toolchains")

rules_proto_dependencies()

rules_proto_toolchains()

go_repository(
    name = "com_github_docker_libkv",
    commit = "458977154600b9f23984d9f4b82e79570b5ae12b",
    importpath = "github.com/docker/libkv",
    patches = ["//third_party:consul_fail_on_missing_key.patch"],
)

# Implicitly used by libkv
go_repository(
    name = "com_github_hashicorp_consul",
    importpath = "github.com/hashicorp/consul",
    tag = "v1.0.7",
)

# Implicitly used by libkv
go_repository(
    name = "com_github_samuel_go_zookeeper",
    commit = "c4fab1ac1bec58281ad0667dc3f0907a9476ac47",
    importpath = "github.com/samuel/go-zookeeper",
)

# Used by zookeeper
go_repository(
    name = "com_github_coreos_go_semver",
    importpath = "github.com/coreos/go-semver",
    tag = "v0.2.0",
)

go_repository(
    name = "com_github_jhump_protoreflect",
    commit = "6c4c7792338ef4769325550489b407691790ffa1",
    importpath = "github.com/jhump/protoreflect",
    patch_args = ["-p1"],
    patches = ["//third_party:protoreflect_proto_std_lib.patch"],
)

go_repository(
    name = "net_starlark_go",
    commit = "d6561f809f318cb4098a9e17073b3dfbf45d3289",
    importpath = "go.starlark.net",
)

# Used by starlark
go_repository(
    name = "com_github_chzyer_readline",
    commit = "2972be24d48e78746da79ba8e24e8b488c9880de",
    importpath = "github.com/chzyer/readline",
)

go_repository(
    name = "com_github_fsnotify_fsnotify",
    commit = "1d13583d846ea9d66dcabbfefbfb9d8e6fb05216",
    importpath = "github.com/fsnotify/fsnotify",
)

go_repository(
    name = "com_github_mitchellh_cli",
    importpath = "github.com/mitchellh/cli",
    tag = "v1.0.0",
)

# Used by cli
go_repository(
    name = "com_github_posener_complete",
    importpath = "github.com/posener/complete",
    tag = "v1.1.1",
)

# Used by cli
go_repository(
    name = "com_github_mattn_go_isatty",
    importpath = "github.com/mattn/go-isatty",
    tag = "v0.0.3",
)

# Used by cli
go_repository(
    name = "com_github_bgentry_speakeasy",
    importpath = "github.com/bgentry/speakeasy",
    tag = "v0.1.0",
)

# Used by cli
go_repository(
    name = "com_github_armon_go_radix",
    commit = "7fddfc383310abc091d79a27f116d30cf0424032",
    importpath = "github.com/armon/go-radix",
)

# Used by cli
go_repository(
    name = "com_github_fatih_color",
    importpath = "github.com/fatih/color",
    tag = "v1.7.0",
)

# Used by cli
go_repository(
    name = "com_github_hashicorp_go_multierror",
    importpath = "github.com/hashicorp/go-multierror",
    tag = "v1.0.0",
)

# Implicitly used by cli
go_repository(
    name = "com_github_hashicorp_errwrap",
    importpath = "github.com/hashicorp/errwrap",
    tag = "v1.0.0",
)

http_archive(
    name = "com_github_bazelbuild_buildtools",
    sha256 = "fabcd8a7f593f6dbe010fffb1d7e032438bd61342ccf0d4791e5211ea01e994a",
    strip_prefix = "buildtools-f720930ceb608b8c0d09528440ce1adeb01e61e0",
    urls = [
        # Master branch as of 2019-07-31
        "https://github.com/bazelbuild/buildtools/archive/f720930ceb608b8c0d09528440ce1adeb01e61e0.tar.gz",
    ],
)

load("@com_github_bazelbuild_buildtools//buildifier:deps.bzl", "buildifier_dependencies")

buildifier_dependencies()

# Docker
http_archive(
    name = "containerregistry",
    patch_args = ["-p1"],
    patches = ["//third_party:containerregistry.patch"],
    sha256 = "a8cdf2452323e0fefa4edb01c08b2ec438c9fa3192bc9f408b89287598c12abc",
    strip_prefix = "containerregistry-0.0.36",
    urls = ["https://github.com/google/containerregistry/archive/v0.0.36.tar.gz"],
)

http_archive(
    name = "io_bazel_rules_docker",
    patch_args = ["-p1"],
    patches = ["//third_party:rules_docker_container_push.patch"],
    sha256 = "87fc6a2b128147a0a3039a2fd0b53cc1f2ed5adb8716f50756544a572999ae9a",
    strip_prefix = "rules_docker-0.8.1",
    urls = ["https://github.com/bazelbuild/rules_docker/archive/v0.8.1.tar.gz"],
)

load(
    "@io_bazel_rules_docker//repositories:repositories.bzl",
    container_repositories = "repositories",
)

container_repositories()

load(
    "@io_bazel_rules_docker//go:image.bzl",
    _go_image_repos = "repositories",
)

_go_image_repos()

go_repository(
    name = "com_github_mitchellh_go_homedir",
    commit = "af06845cf3004701891bf4fdb884bfe4920b3727",
    importpath = "github.com/mitchellh/go-homedir",
)

go_repository(
    name = "org_golang_google_grpc",
    importpath = "google.golang.org/grpc",
    sum = "h1:AzbTB6ux+okLTzP8Ru1Xs41C303zdcfEht7MQnYJt5A=",
    version = "v1.23.0",
)

go_repository(
    name = "org_golang_x_net",
    importpath = "golang.org/x/net",
    sum = "h1:k7pJ2yAPLPgbskkFdhRCsA77k2fySZ1zf2zCjvQCiIM=",
    version = "v0.0.0-20190827160401-ba9fcec4b297",
)

go_repository(
    name = "org_golang_x_text",
    importpath = "golang.org/x/text",
    sum = "h1:tW2bmiBqwgJj/UpqtC8EpXEZVYOwU0yG4iWbprSVAcs=",
    version = "v0.3.2",
)
