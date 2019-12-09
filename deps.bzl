load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

def deps():
    go_repository(
        name = "com_github_docker_libkv",
        commit = "458977154600b9f23984d9f4b82e79570b5ae12b",
        importpath = "github.com/docker/libkv",
        patches = ["@protoconf//third_party:consul_fail_on_missing_key.patch"],
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
        patches = ["@protoconf//third_party:protoreflect_proto_std_lib.patch"],
        build_directives = [
            "gazelle:go_visibility @//:__subpackages__",
            "gazelle:go_visibility @protoconf//:__subpackages__",
        ],
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
