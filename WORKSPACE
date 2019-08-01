workspace(name = "protoconf")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "bazel_skylib",
    sha256 = "1dde365491125a3db70731e25658dfdd3bc5dbdfd11b840b3e987ecf043c7ca0",
    url = "https://github.com/bazelbuild/bazel-skylib/releases/download/0.9.0/bazel_skylib-0.9.0.tar.gz",
)

load("@bazel_skylib//:workspace.bzl", "bazel_skylib_workspace")

bazel_skylib_workspace()

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "a82a352bffae6bee4e95f68a8d80a70e87f42c4741e6a448bec11998fcc82329",
    urls = [
        "https://github.com/bazelbuild/rules_go/releases/download/0.18.5/rules_go-0.18.5.tar.gz",
    ],
)

load(
    "@io_bazel_rules_go//go:deps.bzl",
    "go_register_toolchains",
    "go_rules_dependencies",
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "3c681998538231a2d24d0c07ed5a7658cb72bfb5fd4bf9911157c0e9ac6a2687",
    urls = [
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/0.17.0/bazel-gazelle-0.17.0.tar.gz",
    ],
)

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

gazelle_dependencies()

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
    commit = "ae7b4b3963474bfc47797d6b008f0456d0e3c645",
    importpath = "github.com/jhump/protoreflect",
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

load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

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

go_rules_dependencies()

go_register_toolchains(nogo = "@//:protoconf_nogo")

http_archive(
    name = "com_github_bazelbuild_buildtools",
    strip_prefix = "buildtools-0.28.0",
    url = "https://github.com/bazelbuild/buildtools/archive/0.28.0.zip",
)

load("@com_github_bazelbuild_buildtools//buildifier:deps.bzl", "buildifier_dependencies")

buildifier_dependencies()

# Docker
http_archive(
    name = "containerregistry",
    sha256 = "a8cdf2452323e0fefa4edb01c08b2ec438c9fa3192bc9f408b89287598c12abc",
    strip_prefix = "containerregistry-0.0.36",
    urls = ["https://github.com/google/containerregistry/archive/v0.0.36.tar.gz"],
    patch_args = ["-p1"],
    patches = ["//third_party:containerregistry.patch"],
)

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "87fc6a2b128147a0a3039a2fd0b53cc1f2ed5adb8716f50756544a572999ae9a",
    strip_prefix = "rules_docker-0.8.1",
    urls = ["https://github.com/bazelbuild/rules_docker/archive/v0.8.1.tar.gz"],
    patch_args = ["-p1"],
    patches = ["//third_party:rules_docker_container_push.patch"],
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
