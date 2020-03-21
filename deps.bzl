load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

def deps():
    go_repository(
        name = "com_github_docker_libkv",
        importpath = "github.com/docker/libkv",
        patches = ["@protoconf//third_party:consul_fail_on_missing_key.patch"],
        sum = "h1:PNXYaftMVCFS5CmnDtDWTg3wbBO61Q/cEo3KX1oKxto=",
        version = "v0.2.1",
    )

    # Implicitly used by libkv
    go_repository(
        name = "com_github_hashicorp_consul",
        importpath = "github.com/hashicorp/consul",
        sum = "h1:OjnFfc1vHPLtwrYAMC9HlJ4WTSmgBTq2erWuBERm3hY=",
        version = "v1.6.2",
    )

    # Implicitly used by libkv
    go_repository(
        name = "com_github_samuel_go_zookeeper",
        importpath = "github.com/samuel/go-zookeeper",
        sum = "h1:p3Vo3i64TCLY7gIfzeQaUJ+kppEO5WQG3cL8iE8tGHU=",
        version = "v0.0.0-20190923202752-2cc03de413da",
    )

    # Used by zookeeper
    go_repository(
        name = "com_github_coreos_go_semver",
        importpath = "github.com/coreos/go-semver",
        sum = "h1:U0BbGfKnviqVBJQB4etvm+mKx53KfkumNLBt6YeF/0Q=",
        version = "v0.2.1-0.20180108230905-e214231b295a",
    )

    go_repository(
        name = "com_github_jhump_protoreflect",
        build_directives = [
            "gazelle:go_visibility @//:__subpackages__",
            "gazelle:go_visibility @protoconf//:__subpackages__",
        ],
        importpath = "github.com/jhump/protoreflect",
        patch_args = ["-p1"],
        patches = ["@protoconf//third_party:protoreflect_proto_std_lib.patch"],
        sum = "h1:/ZCz2pTHWjPMWQsB0qMWK55vG8XS8EnLvEwaj9iQyrU=",
        version = "v1.4.5-0.20190719031800-6c4c7792338e",
    )

    go_repository(
        name = "net_starlark_go",
        importpath = "go.starlark.net",
        sum = "h1:GMoeMjox1bR6zacPH+6Z1biOylJzpDbnzven0ymPRg8=",
        version = "v0.0.0-20191218235703-9fcb808a6221",
    )

    # Used by starlark
    go_repository(
        name = "com_github_chzyer_readline",
        importpath = "github.com/chzyer/readline",
        sum = "h1:fY5BOSpyZCqRo5OhCuC+XN+r/bBCmeuuJtjz+bCNIf8=",
        version = "v0.0.0-20180603132655-2972be24d48e",
    )

    go_repository(
        name = "com_github_fsnotify_fsnotify",
        importpath = "github.com/fsnotify/fsnotify",
        sum = "h1:IXs+QLmnXW2CcXuY+8Mzv/fWEsPGWxqefPtCP5CnV9I=",
        version = "v1.4.7",
    )

    go_repository(
        name = "com_github_mitchellh_cli",
        importpath = "github.com/mitchellh/cli",
        sum = "h1:iGBIsUe3+HZ/AD/Vd7DErOt5sU9fa8Uj7A2s1aggv1Y=",
        version = "v1.0.0",
    )

    # Used by cli
    go_repository(
        name = "com_github_posener_complete",
        importpath = "github.com/posener/complete",
        sum = "h1:LrvDIY//XNo65Lq84G/akBuMGlawHvGBABv8f/ZN6DI=",
        version = "v1.2.1",
    )

    # Used by cli
    go_repository(
        name = "com_github_mattn_go_isatty",
        importpath = "github.com/mattn/go-isatty",
        sum = "h1:HLtExJ+uU2HOZ+wI0Tt5DtUDrx8yhUqDcp7fYERX4CE=",
        version = "v0.0.8",
    )

    # Used by cli
    go_repository(
        name = "com_github_bgentry_speakeasy",
        importpath = "github.com/bgentry/speakeasy",
        sum = "h1:ByYyxL9InA1OWqxJqqp2A5pYHUrCiAL6K3J+LKSsQkY=",
        version = "v0.1.0",
    )

    # Used by cli
    go_repository(
        name = "com_github_armon_go_radix",
        importpath = "github.com/armon/go-radix",
        sum = "h1:F4z6KzEeeQIMeLFa97iZU6vupzoecKdU5TX24SNppXI=",
        version = "v1.0.0",
    )

    # Used by cli
    go_repository(
        name = "com_github_fatih_color",
        importpath = "github.com/fatih/color",
        sum = "h1:DkWD4oS2D8LGGgTQ6IvwJJXSL5Vp2ffcQg58nFV38Ys=",
        version = "v1.7.0",
    )

    # Used by cli
    go_repository(
        name = "com_github_hashicorp_go_multierror",
        importpath = "github.com/hashicorp/go-multierror",
        sum = "h1:iVjPR7a6H0tWELX5NxNe7bYopibicUzc7uPribsnS6o=",
        version = "v1.0.0",
    )

    # Implicitly used by cli
    go_repository(
        name = "com_github_hashicorp_errwrap",
        importpath = "github.com/hashicorp/errwrap",
        sum = "h1:hLrqtEDnRye3+sgx6z4qVLNuviH3MR5aQ0ykNJa/UYA=",
        version = "v1.0.0",
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
        importpath = "github.com/mitchellh/go-homedir",
        sum = "h1:lukF9ziXFxDFPkA1vsr5zpc1XuPDn/wFntq5mG+4E0Y=",
        version = "v1.1.0",
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
        sum = "h1:efeOvDhwQ29Dj3SdAV/MJf8oukgn+8D8WgaCaRMchF8=",
        version = "v0.0.0-20191209160850-c0dbc17a3553",
    )

    go_repository(
        name = "org_golang_x_text",
        importpath = "golang.org/x/text",
        sum = "h1:tW2bmiBqwgJj/UpqtC8EpXEZVYOwU0yG4iWbprSVAcs=",
        version = "v0.3.2",
    )
    go_repository(
        name = "com_github_coreos_etcd",
        build_directives = [
            "gazelle:proto disable",
        ],
        build_file_generation = "on",
        importpath = "github.com/coreos/etcd",
        sum = "h1:Zz1aXgDrFFi1nadh58tA9ktt06cmPTwNNP3dXwIq1lE=",
        version = "v3.3.18+incompatible",
    )
    go_repository(
        name = "io_etcd_go_bbolt",
        importpath = "go.etcd.io/bbolt",
        sum = "h1:MUGmc65QhB3pIlaQ5bB4LwqSj6GIonVJXpZiaKNyaKk=",
        version = "v1.3.3",
    )
    go_repository(
        name = "com_github_stretchr_objx",
        importpath = "github.com/stretchr/objx",
        sum = "h1:2vfRuCMp5sSVIDSqO8oNnWJq7mPa6KVP3iPIwFBuy8A=",
        version = "v0.1.1",
    )
    go_repository(
        name = "com_github_modern_go_reflect2",
        importpath = "github.com/modern-go/reflect2",
        sum = "h1:9f412s+6RmYXLWZSEzVVgPGK7C2PphHj5RJrvfx9AWI=",
        version = "v1.0.1",
    )
    go_repository(
        name = "co_honnef_go_tools",
        importpath = "honnef.co/go/tools",
        sum = "h1:/hemPrYIhOhy8zYrNj+069zDB68us2sMGsfkFJO0iZs=",
        version = "v0.0.0-20190523083050-ea95bdfd59fc",
    )
    go_repository(
        name = "com_github_abdullin_seq",
        importpath = "github.com/abdullin/seq",
        sum = "h1:DBNMBMuMiWYu0b+8KMJuWmfCkcxl09JwdlqwDZZ6U14=",
        version = "v0.0.0-20160510034733-d5467c17e7af",
    )
    go_repository(
        name = "com_github_agext_levenshtein",
        importpath = "github.com/agext/levenshtein",
        sum = "h1:0S/Yg6LYmFJ5stwQeRp6EeOcCbj7xiqQSdNelsXvaqE=",
        version = "v1.2.2",
    )
    go_repository(
        name = "com_github_agl_ed25519",
        importpath = "github.com/agl/ed25519",
        sum = "h1:LoeFxdq5zUCBQPhbQKE6zvoGwHMxCBlqwbH9+9kHoHA=",
        version = "v0.0.0-20150830182803-278e1ec8e8a6",
    )
    go_repository(
        name = "com_github_alecthomas_template",
        importpath = "github.com/alecthomas/template",
        sum = "h1:cAKDfWh5VpdgMhJosfJnn5/FoN2SRZ4p7fJNX58YPaU=",
        version = "v0.0.0-20160405071501-a0175ee3bccc",
    )
    go_repository(
        name = "com_github_alecthomas_units",
        importpath = "github.com/alecthomas/units",
        sum = "h1:qet1QNfXsQxTZqLG4oE62mJzwPIB8+Tee4RNCL9ulrY=",
        version = "v0.0.0-20151022065526-2efee857e7cf",
    )
    go_repository(
        name = "com_github_aliyun_alibaba_cloud_sdk_go",
        importpath = "github.com/aliyun/alibaba-cloud-sdk-go",
        sum = "h1:APorzFpCcv6wtD5vmRWYqNm4N55kbepL7c7kTq9XI6A=",
        version = "v0.0.0-20190329064014-6e358769c32a",
    )
    go_repository(
        name = "com_github_aliyun_aliyun_oss_go_sdk",
        importpath = "github.com/aliyun/aliyun-oss-go-sdk",
        sum = "h1:FrF4uxA24DF3ARNXVbUin3wa5fDLaB1Cy8mKks/LRz4=",
        version = "v0.0.0-20190103054945-8205d1f41e70",
    )
    go_repository(
        name = "com_github_aliyun_aliyun_tablestore_go_sdk",
        importpath = "github.com/aliyun/aliyun-tablestore-go-sdk",
        sum = "h1:ABQ7FF+IxSFHDMOTtjCfmMDMHiCq6EsAoCV/9sFinaM=",
        version = "v4.1.2+incompatible",
    )
    go_repository(
        name = "com_github_antchfx_xpath",
        importpath = "github.com/antchfx/xpath",
        sum = "h1:ptBAamGVd6CfRsUtyHD+goy2JGhv1QC32v3gqM8mYAM=",
        version = "v0.0.0-20190129040759-c8489ed3251e",
    )
    go_repository(
        name = "com_github_antchfx_xquery",
        importpath = "github.com/antchfx/xquery",
        sum = "h1:JaCC8jz0zdMLk2m+qCCVLLLM/PL93p84w4pK3aJWj60=",
        version = "v0.0.0-20180515051857-ad5b8c7a47b0",
    )
    go_repository(
        name = "com_github_apparentlymart_go_cidr",
        importpath = "github.com/apparentlymart/go-cidr",
        sum = "h1:NmIwLZ/KdsjIUlhf+/Np40atNXm/+lZ5txfTJ/SpF+U=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_github_apparentlymart_go_dump",
        importpath = "github.com/apparentlymart/go-dump",
        sum = "h1:MzVXffFUye+ZcSR6opIgz9Co7WcDx6ZcY+RjfFHoA0I=",
        version = "v0.0.0-20190214190832-042adf3cf4a0",
    )
    go_repository(
        name = "com_github_apparentlymart_go_textseg",
        importpath = "github.com/apparentlymart/go-textseg",
        sum = "h1:rRmlIsPEEhUTIKQb7T++Nz/A5Q6C9IuX2wFoYVvnCs0=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_armon_circbuf",
        importpath = "github.com/armon/circbuf",
        sum = "h1:7Ip0wMmLHLRJdrloDxZfhMm0xrLXZS8+COSu2bXmEQs=",
        version = "v0.0.0-20190214190532-5111143e8da2",
    )
    go_repository(
        name = "com_github_armon_go_metrics",
        importpath = "github.com/armon/go-metrics",
        sum = "h1:EFSB7Zo9Eg91v7MJPVsifUysc/wPdN+NOnVe6bWbdBM=",
        version = "v0.0.0-20190430140413-ec5e00d3c878",
    )
    go_repository(
        name = "com_github_aws_aws_sdk_go",
        importpath = "github.com/aws/aws-sdk-go",
        sum = "h1:uM16hIw9BotjZKMZlX05SN2EFtaWfi/NonPKIARiBLQ=",
        version = "v1.25.3",
    )
    go_repository(
        name = "com_github_azure_azure_sdk_for_go",
        importpath = "github.com/Azure/azure-sdk-for-go",
        sum = "h1:09cv2WoH0g6jl6m2iT+R9qcIPZKhXEL0sbmLhxP895s=",
        version = "v36.2.0+incompatible",
    )
    go_repository(
        name = "com_github_azure_go_autorest_autorest",
        importpath = "github.com/Azure/go-autorest/autorest",
        sum = "h1:6AWuh3uWrsZJcNoCHrCF/+g4aKPCU39kaMO6/qrnK/4=",
        version = "v0.9.2",
    )
    go_repository(
        name = "com_github_azure_go_autorest_autorest_adal",
        importpath = "github.com/Azure/go-autorest/autorest/adal",
        sum = "h1:Hxqlh1uAA8aGpa1dFhDNhll7U/rkWtG8ZItFvRMr7l0=",
        version = "v0.8.1-0.20191028180845-3492b2aff503",
    )
    go_repository(
        name = "com_github_azure_go_autorest_autorest_azure_cli",
        importpath = "github.com/Azure/go-autorest/autorest/azure/cli",
        sum = "h1:pSwNMF0qotgehbQNllUWwJ4V3vnrLKOzHrwDLEZK904=",
        version = "v0.2.0",
    )
    go_repository(
        name = "com_github_azure_go_autorest_autorest_date",
        importpath = "github.com/Azure/go-autorest/autorest/date",
        sum = "h1:yW+Zlqf26583pE43KhfnhFcdmSWlm5Ew6bxipnr/tbM=",
        version = "v0.2.0",
    )
    go_repository(
        name = "com_github_azure_go_autorest_autorest_mocks",
        importpath = "github.com/Azure/go-autorest/autorest/mocks",
        sum = "h1:qJumjCaCudz+OcqE9/XtEPfvtOjOmKaui4EOpFI6zZc=",
        version = "v0.3.0",
    )
    go_repository(
        name = "com_github_azure_go_autorest_autorest_to",
        importpath = "github.com/Azure/go-autorest/autorest/to",
        sum = "h1:zebkZaadz7+wIQYgC7GXaz3Wb28yKYfVkkBKwc38VF8=",
        version = "v0.3.0",
    )
    go_repository(
        name = "com_github_azure_go_autorest_autorest_validation",
        importpath = "github.com/Azure/go-autorest/autorest/validation",
        sum = "h1:15vMO4y76dehZSq7pAaOLQxC6dZYsSrj2GQpflyM/L4=",
        version = "v0.2.0",
    )
    go_repository(
        name = "com_github_azure_go_autorest_logger",
        importpath = "github.com/Azure/go-autorest/logger",
        sum = "h1:ruG4BSDXONFRrZZJ2GUXDiUyVpayPmb1GnWeHDdaNKY=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_azure_go_autorest_tracing",
        importpath = "github.com/Azure/go-autorest/tracing",
        sum = "h1:TRn4WjSnkcSy5AEG3pnbtFSwNtwzjr4VYyQflFE619k=",
        version = "v0.5.0",
    )
    go_repository(
        name = "com_github_azure_go_ntlmssp",
        importpath = "github.com/Azure/go-ntlmssp",
        sum = "h1:pSm8mp0T2OH2CPmPDPtwHPr3VAQaOwVF/JbllOPP4xA=",
        version = "v0.0.0-20180810175552-4a21cbd618b4",
    )
    go_repository(
        name = "com_github_baiyubin_aliyun_sts_go_sdk",
        importpath = "github.com/baiyubin/aliyun-sts-go-sdk",
        sum = "h1:ZNv7On9kyUzm7fvRZumSyy/IUiSC7AzL0I1jKKtwooA=",
        version = "v0.0.0-20180326062324-cfa1a18b161f",
    )
    go_repository(
        name = "com_github_beorn7_perks",
        importpath = "github.com/beorn7/perks",
        sum = "h1:HWo1m869IqiPhD389kmkxeTalrjNbbJTC8LXupb+sl0=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_bgentry_go_netrc",
        importpath = "github.com/bgentry/go-netrc",
        sum = "h1:xDfNPAt8lFiC1UJrqV3uuy861HCTo708pDMbjHHdCas=",
        version = "v0.0.0-20140422174119-9fd32a8b3d3d",
    )
    go_repository(
        name = "com_github_blang_semver",
        importpath = "github.com/blang/semver",
        sum = "h1:cQNTCjp13qL8KC3Nbxr/y2Bqb63oX6wdnnjpJbkM4JQ=",
        version = "v3.5.1+incompatible",
    )
    go_repository(
        name = "com_github_bmatcuk_doublestar",
        importpath = "github.com/bmatcuk/doublestar",
        sum = "h1:2bNwBOmhyFEFcoB3tGvTD5xanq+4kyOZlB8wFYbMjkk=",
        version = "v1.1.5",
    )
    go_repository(
        name = "com_github_boltdb_bolt",
        importpath = "github.com/boltdb/bolt",
        sum = "h1:JQmyP4ZBrce+ZQu0dY660FMfatumYDLun9hBCUVIkF4=",
        version = "v1.3.1",
    )
    go_repository(
        name = "com_github_bsm_go_vlq",
        importpath = "github.com/bsm/go-vlq",
        sum = "h1:D64GF/Xr5zSUnM3q1Jylzo4sK7szhP/ON+nb2DB5XJA=",
        version = "v0.0.0-20150828105119-ec6e8d4f5f4e",
    )
    go_repository(
        name = "com_github_burntsushi_toml",
        importpath = "github.com/BurntSushi/toml",
        sum = "h1:WXkYYl6Yr3qBf1K79EBnL4mak0OimBfB0XUf9Vl28OQ=",
        version = "v0.3.1",
    )
    go_repository(
        name = "com_github_burntsushi_xgb",
        importpath = "github.com/BurntSushi/xgb",
        sum = "h1:1BDTz0u9nC3//pOCMdNH+CiXJVYJh5UQNCOBG7jbELc=",
        version = "v0.0.0-20160522181843-27f122750802",
    )
    go_repository(
        name = "com_github_census_instrumentation_opencensus_proto",
        importpath = "github.com/census-instrumentation/opencensus-proto",
        sum = "h1:glEXhBS5PSLLv4IXzLA5yPRVX4bilULVyxxbrfOtDAk=",
        version = "v0.2.1",
    )
    go_repository(
        name = "com_github_cheggaaa_pb",
        importpath = "github.com/cheggaaa/pb",
        sum = "h1:wIkZHkNfC7R6GI5w7l/PdAdzXzlrbcI3p8OAlnkTsnc=",
        version = "v1.0.27",
    )
    go_repository(
        name = "com_github_christrenkamp_goxpath",
        importpath = "github.com/ChrisTrenkamp/goxpath",
        sum = "h1:y8Gs8CzNfDF5AZvjr+5UyGQvQEBL7pwo+v+wX6q9JI8=",
        version = "v0.0.0-20170922090931-c385f95c6022",
    )
    go_repository(
        name = "com_github_chzyer_logex",
        importpath = "github.com/chzyer/logex",
        sum = "h1:Swpa1K6QvQznwJRcfTfQJmTE72DqScAa40E+fbHEXEE=",
        version = "v1.1.10",
    )
    go_repository(
        name = "com_github_chzyer_test",
        importpath = "github.com/chzyer/test",
        sum = "h1:q763qf9huN11kDQavWsoZXJNW3xEE4JJyHa5Q25/sd8=",
        version = "v0.0.0-20180213035817-a1ea475d72b1",
    )
    go_repository(
        name = "com_github_client9_misspell",
        importpath = "github.com/client9/misspell",
        sum = "h1:ta993UF76GwbvJcIo3Y68y/M3WxlpEHPWIGDkJYwzJI=",
        version = "v0.3.4",
    )
    go_repository(
        name = "com_github_coreos_bbolt",
        importpath = "github.com/coreos/bbolt",
        sum = "h1:HIgH5xUWXT914HCI671AxuTTqjj64UOFr7pHn48LUTI=",
        version = "v1.3.0",
    )
    go_repository(
        name = "com_github_coreos_go_systemd",
        importpath = "github.com/coreos/go-systemd",
        sum = "h1:t5Wuyh53qYyg9eqn4BbnlIT+vmhyww0TatL+zT3uWgI=",
        version = "v0.0.0-20181012123002-c6f51f82210d",
    )
    go_repository(
        name = "com_github_coreos_pkg",
        importpath = "github.com/coreos/pkg",
        sum = "h1:lBNOc5arjvs8E5mO2tbpBpLoyyu8B6e44T7hJy6potg=",
        version = "v0.0.0-20180928190104-399ea9e2e55f",
    )
    go_repository(
        name = "com_github_davecgh_go_spew",
        importpath = "github.com/davecgh/go-spew",
        sum = "h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=",
        version = "v1.1.1",
    )
    go_repository(
        name = "com_github_dgrijalva_jwt_go",
        importpath = "github.com/dgrijalva/jwt-go",
        sum = "h1:7qlOGliEKZXTDg6OTjfoBKDXWrumCAMpl/TFQ4/5kLM=",
        version = "v3.2.0+incompatible",
    )
    go_repository(
        name = "com_github_dimchansky_utfbom",
        importpath = "github.com/dimchansky/utfbom",
        sum = "h1:FcM3g+nofKgUteL8dm/UpdRXNC9KmADgTpLKsu0TRo4=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_dnaeon_go_vcr",
        importpath = "github.com/dnaeon/go-vcr",
        sum = "h1:Dzuw9GtbmllUqEcoHfScT9YpKFUssSiZ5PgZkIGf/YQ=",
        version = "v0.0.0-20180920040454-5637cf3d8a31",
    )
    go_repository(
        name = "com_github_dylanmei_iso8601",
        importpath = "github.com/dylanmei/iso8601",
        sum = "h1:812NGQDBcqquTfH5Yeo7lwR0nzx/cKdsmf3qMjPURUI=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_dylanmei_winrmtest",
        importpath = "github.com/dylanmei/winrmtest",
        sum = "h1:r1oACdS2XYiAWcfF8BJXkoU8l1J71KehGR+d99yWEDA=",
        version = "v0.0.0-20190225150635-99b7fe2fddf1",
    )
    go_repository(
        name = "com_github_envoyproxy_go_control_plane",
        importpath = "github.com/envoyproxy/go-control-plane",
        sum = "h1:uE6Fp4fOcAJdc1wTQXLJ+SYistkbG1dNoi6Zs1+Ybvk=",
        version = "v0.8.0",
    )
    go_repository(
        name = "com_github_envoyproxy_protoc_gen_validate",
        importpath = "github.com/envoyproxy/protoc-gen-validate",
        sum = "h1:EQciDnbrYxy13PgWoY8AqoxGiPrpgBZ1R8UNe3ddc+A=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_ghodss_yaml",
        importpath = "github.com/ghodss/yaml",
        sum = "h1:wQHKEahhL6wmXdzwWG11gIVCkOv05bNOh+Rxn0yngAk=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_go_kit_kit",
        importpath = "github.com/go-kit/kit",
        sum = "h1:Wz+5lgoB0kkuqLEc6NVmwRknTKP6dTGbSqvhZtBI/j0=",
        version = "v0.8.0",
    )
    go_repository(
        name = "com_github_go_logfmt_logfmt",
        importpath = "github.com/go-logfmt/logfmt",
        sum = "h1:MP4Eh7ZCb31lleYCFuwm0oe4/YGak+5l1vA2NOE80nA=",
        version = "v0.4.0",
    )
    go_repository(
        name = "com_github_go_stack_stack",
        importpath = "github.com/go-stack/stack",
        sum = "h1:5SgMzNM5HxrEjV0ww2lTmX6E2Izsfxas4+YHWRs3Lsk=",
        version = "v1.8.0",
    )
    go_repository(
        name = "com_github_go_test_deep",
        importpath = "github.com/go-test/deep",
        sum = "h1:ZrJSEWsXzPOxaZnFteGEfooLba+ju3FYIbOrS+rQd68=",
        version = "v1.0.3",
    )
    go_repository(
        name = "com_github_gogo_protobuf",
        importpath = "github.com/gogo/protobuf",
        sum = "h1:/s5zKNz0uPFCZ5hddgPdo2TK2TVrUNMn0OOX8/aZMTE=",
        version = "v1.2.1",
    )
    go_repository(
        name = "com_github_golang_glog",
        importpath = "github.com/golang/glog",
        sum = "h1:VKtxabqXZkF25pY9ekfRL6a582T4P37/31XEstQ5p58=",
        version = "v0.0.0-20160126235308-23def4e6c14b",
    )
    go_repository(
        name = "com_github_golang_groupcache",
        importpath = "github.com/golang/groupcache",
        sum = "h1:u4bArs140e9+AfE52mFHOXVFnOSBJBRlzTHrOPLOIhE=",
        version = "v0.0.0-20180513044358-24b0969c4cb7",
    )
    go_repository(
        name = "com_github_golang_mock",
        importpath = "github.com/golang/mock",
        sum = "h1:qGJ6qTW+x6xX/my+8YUVl4WNpX9B7+/l2tRsHGZ7f2s=",
        version = "v1.3.1",
    )
    go_repository(
        name = "com_github_golang_snappy",
        importpath = "github.com/golang/snappy",
        sum = "h1:Qgr9rKW7uDUkrbSmQeiDsGa8SjGyCOGtuasMWwvp2P4=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_google_btree",
        importpath = "github.com/google/btree",
        sum = "h1:0udJVsspx3VBr5FwtLhQQtuAsVc79tTq0ocGIPAU6qo=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_google_go_cmp",
        importpath = "github.com/google/go-cmp",
        sum = "h1:Xye71clBPdm5HgqGwUkwhbynsUJZhDbS20FvLhQ2izg=",
        version = "v0.3.1",
    )
    go_repository(
        name = "com_github_google_go_querystring",
        importpath = "github.com/google/go-querystring",
        sum = "h1:Xkwi/a1rcvNg1PPYe5vI8GbeBY/jrVuDX5ASuANWTrk=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_google_martian",
        importpath = "github.com/google/martian",
        sum = "h1:/CP5g8u/VJHijgedC/Legn3BAbAaWPgecwXBIDzw5no=",
        version = "v2.1.0+incompatible",
    )
    go_repository(
        name = "com_github_google_pprof",
        importpath = "github.com/google/pprof",
        sum = "h1:Jnx61latede7zDD3DiiP4gmNz33uK0U5HDUaF0a/HVQ=",
        version = "v0.0.0-20190515194954-54271f7e092f",
    )
    go_repository(
        name = "com_github_google_uuid",
        importpath = "github.com/google/uuid",
        sum = "h1:Gkbcsh/GbpXz7lPftLA3P6TYMwjCLYm83jiFQZF/3gY=",
        version = "v1.1.1",
    )
    go_repository(
        name = "com_github_googleapis_gax_go_v2",
        importpath = "github.com/googleapis/gax-go/v2",
        sum = "h1:sjZBwGj9Jlw33ImPtvFviGYvseOtDM7hkSKB7+Tv3SM=",
        version = "v2.0.5",
    )
    go_repository(
        name = "com_github_gophercloud_gophercloud",
        importpath = "github.com/gophercloud/gophercloud",
        sum = "h1:Pu+HW4kcQozw0QyrTTgLE+3RXNqFhQNNzhbnoLFL83c=",
        version = "v0.0.0-20190208042652-bc37892e1968",
    )
    go_repository(
        name = "com_github_gophercloud_utils",
        importpath = "github.com/gophercloud/utils",
        sum = "h1:OgCNGSnEalfkRpn//WGJHhpo7fkP+LhTpvEITZ7CkK4=",
        version = "v0.0.0-20190128072930-fbb6ab446f01",
    )
    go_repository(
        name = "com_github_gopherjs_gopherjs",
        importpath = "github.com/gopherjs/gopherjs",
        sum = "h1:EGx4pi6eqNxGaHF6qqu48+N2wcFQ5qg5FXgOdqsJ5d8=",
        version = "v0.0.0-20181017120253-0766667cb4d1",
    )
    go_repository(
        name = "com_github_gorilla_websocket",
        importpath = "github.com/gorilla/websocket",
        sum = "h1:WDFjx/TMzVgy9VdMMQi2K2Emtwi2QcUQsztZ/zLaH/Q=",
        version = "v1.4.0",
    )
    go_repository(
        name = "com_github_grpc_ecosystem_go_grpc_middleware",
        importpath = "github.com/grpc-ecosystem/go-grpc-middleware",
        sum = "h1:Iju5GlWwrvL6UBg4zJJt3btmonfrMlCDdsejg4CZE7c=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_grpc_ecosystem_go_grpc_prometheus",
        importpath = "github.com/grpc-ecosystem/go-grpc-prometheus",
        sum = "h1:Ovs26xHkKqVztRpIrF/92BcuyuQ/YW4NSIpoGtfXNho=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_grpc_ecosystem_grpc_gateway",
        importpath = "github.com/grpc-ecosystem/grpc-gateway",
        sum = "h1:2+KSC78XiO6Qy0hIjfc1OD9H+hsaJdJlb8Kqsd41CTE=",
        version = "v1.8.5",
    )
    go_repository(
        name = "com_github_hashicorp_aws_sdk_go_base",
        importpath = "github.com/hashicorp/aws-sdk-go-base",
        sum = "h1:zH9hNUdsS+2G0zJaU85ul8D59BGnZBaKM+KMNPAHGwk=",
        version = "v0.4.0",
    )
    go_repository(
        name = "com_github_hashicorp_consul_api",
        importpath = "github.com/hashicorp/consul/api",
        sum = "h1:HXNYlRkkM/t+Y/Yhxtwcy02dlYwIaoxzvxPnS+cqy78=",
        version = "v1.3.0",
    )
    go_repository(
        name = "com_github_hashicorp_consul_sdk",
        importpath = "github.com/hashicorp/consul/sdk",
        sum = "h1:UOxjlb4xVNF93jak1mzzoBatyFju9nrkxpVwIp/QqxQ=",
        version = "v0.3.0",
    )
    go_repository(
        name = "com_github_hashicorp_go_azure_helpers",
        importpath = "github.com/hashicorp/go-azure-helpers",
        sum = "h1:KhjDnQhCqEMKlt4yH00MCevJQPJ6LkHFdSveXINO6vE=",
        version = "v0.10.0",
    )
    go_repository(
        name = "com_github_hashicorp_go_checkpoint",
        importpath = "github.com/hashicorp/go-checkpoint",
        sum = "h1:MFYpPZCnQqQTE18jFwSII6eUQrD/oxMFp3mlgcqk5mU=",
        version = "v0.5.0",
    )
    go_repository(
        name = "com_github_hashicorp_go_cleanhttp",
        importpath = "github.com/hashicorp/go-cleanhttp",
        sum = "h1:dH3aiDG9Jvb5r5+bYHsikaOUIpcM0xvgMXVoDkXMzJM=",
        version = "v0.5.1",
    )
    go_repository(
        name = "com_github_hashicorp_go_getter",
        importpath = "github.com/hashicorp/go-getter",
        sum = "h1:ENHNi8494porjD0ZhIrjlAHnveSFhY7hvOJrV/fsKkw=",
        version = "v1.4.0",
    )
    go_repository(
        name = "com_github_hashicorp_go_hclog",
        importpath = "github.com/hashicorp/go-hclog",
        sum = "h1:CG6TE5H9/JXsFWJCfoIVpKFIkFe6ysEuHirp4DxCsHI=",
        version = "v0.9.2",
    )
    go_repository(
        name = "com_github_hashicorp_go_immutable_radix",
        importpath = "github.com/hashicorp/go-immutable-radix",
        sum = "h1:AKDB1HM5PWEA7i4nhcpwOrO2byshxBjXVn/J/3+z5/0=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_hashicorp_go_msgpack",
        importpath = "github.com/hashicorp/go-msgpack",
        sum = "h1:i9R9JSrqIz0QVLz3sz+i3YJdT7TTSLcfLLzJi9aZTuI=",
        version = "v0.5.5",
    )
    go_repository(
        name = "com_github_hashicorp_go_net",
        importpath = "github.com/hashicorp/go.net",
        sum = "h1:sNCoNyDEvN1xa+X0baata4RdcpKwcMS6DH+xwfqPgjw=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_hashicorp_go_plugin",
        importpath = "github.com/hashicorp/go-plugin",
        sum = "h1:4OtAfUGbnKC6yS48p0CtMX2oFYtzFZVv6rok3cRWgnE=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_github_hashicorp_go_retryablehttp",
        importpath = "github.com/hashicorp/go-retryablehttp",
        sum = "h1:1BZvpawXoJCWX6pNtow9+rpEj+3itIlutiqnntI6jOE=",
        version = "v0.5.4",
    )
    go_repository(
        name = "com_github_hashicorp_go_rootcerts",
        build_directives = [
            "gazelle:exclude",
        ],
        importpath = "github.com/hashicorp/go-rootcerts",
        patch_args = ["-p1"],
        patches = ["@protoconf//third_party:go_rootcerts_fix_darwin_cross_build.patch"],
        sum = "h1:DMo4fmknnz0E0evoNYnV48RjWndOsmd6OW+09R3cEP8=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_github_hashicorp_go_safetemp",
        importpath = "github.com/hashicorp/go-safetemp",
        sum = "h1:2HR189eFNrjHQyENnQMMpCiBAsRxzbTMIgBhEyExpmo=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_hashicorp_go_slug",
        importpath = "github.com/hashicorp/go-slug",
        sum = "h1:/jAo8dNuLgSImoLXaX7Od7QB4TfYCVPam+OpAt5bZqc=",
        version = "v0.4.1",
    )
    go_repository(
        name = "com_github_hashicorp_go_sockaddr",
        importpath = "github.com/hashicorp/go-sockaddr",
        sum = "h1:ztczhD1jLxIRjVejw8gFomI1BQZOe2WoVOu0SyteCQc=",
        version = "v1.0.2",
    )
    go_repository(
        name = "com_github_hashicorp_go_syslog",
        importpath = "github.com/hashicorp/go-syslog",
        sum = "h1:KaodqZuhUoZereWVIYmpUgZysurB1kBLX2j0MwMrUAE=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_hashicorp_go_tfe",
        importpath = "github.com/hashicorp/go-tfe",
        sum = "h1:7XZ/ZoPyYoeuNXaWWW0mJOq016y0qb7I4Q0P/cagyu8=",
        version = "v0.3.27",
    )
    go_repository(
        name = "com_github_hashicorp_go_uuid",
        importpath = "github.com/hashicorp/go-uuid",
        sum = "h1:fv1ep09latC32wFoVwnqcnKJGnMSdBanPczbHAYm1BE=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_github_hashicorp_go_version",
        importpath = "github.com/hashicorp/go-version",
        sum = "h1:3vNe/fWF5CBgRIguda1meWhsZHy3m8gCJ5wx+dIzX/E=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_hashicorp_golang_lru",
        importpath = "github.com/hashicorp/golang-lru",
        sum = "h1:0hERBMJE1eitiLkihrMvRVBYAkpHzc/J3QdDN+dAcgU=",
        version = "v0.5.1",
    )
    go_repository(
        name = "com_github_hashicorp_hcl",
        importpath = "github.com/hashicorp/hcl",
        sum = "h1:0Anlzjpi4vEasTeNFn2mLJgTSwt0+6sfsiTG8qcWGx4=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_hashicorp_hcl2",
        importpath = "github.com/hashicorp/hcl2",
        sum = "h1:JImQpEeUQ+0DPFMaWzLA0GdUNPaUlCXLpfiqkSZBUfc=",
        version = "v0.0.0-20190821123243-0c888d1241f6",
    )
    go_repository(
        name = "com_github_hashicorp_hcl_v2",
        importpath = "github.com/hashicorp/hcl/v2",
        sum = "h1:efQznTz+ydmQXq3BOnRa3AXzvCeTq1P4dKj/z5GLlY8=",
        version = "v2.0.0",
    )
    go_repository(
        name = "com_github_hashicorp_hil",
        importpath = "github.com/hashicorp/hil",
        sum = "h1:2yzhWGdgQUWZUCNK+AoO35V+HTsgEmcM4J9IkArh7PI=",
        version = "v0.0.0-20190212112733-ab17b08d6590",
    )
    go_repository(
        name = "com_github_hashicorp_logutils",
        importpath = "github.com/hashicorp/logutils",
        sum = "h1:dLEQVugN8vlakKOUE3ihGLTZJRB4j+M2cdTm/ORI65Y=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_hashicorp_mdns",
        importpath = "github.com/hashicorp/mdns",
        sum = "h1:XFSOubp8KWB+Jd2PDyaX5xUd5bhSP/+pTDZVDMzZJM8=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_github_hashicorp_memberlist",
        importpath = "github.com/hashicorp/memberlist",
        sum = "h1:AYBsgJOW9gab/toO5tEB8lWetVgDKZycqkebJ8xxpqM=",
        version = "v0.1.5",
    )
    go_repository(
        name = "com_github_hashicorp_serf",
        importpath = "github.com/hashicorp/serf",
        sum = "h1:ZynDUIQiA8usmRgPdGPHFdPnb1wgGI9tK3mO9hcAJjc=",
        version = "v0.8.5",
    )
    go_repository(
        name = "com_github_hashicorp_terraform",
        importpath = "github.com/hashicorp/terraform",
        sum = "h1:U9gd/12wfT0Q7JYM43Hob6rcirICKCnxSDY+sJlYh6A=",
        version = "v0.12.18",
    )
    go_repository(
        name = "com_github_hashicorp_terraform_config_inspect",
        importpath = "github.com/hashicorp/terraform-config-inspect",
        sum = "h1:fTkL0YwjohGyN7AqsDhz6bwcGBpT+xBqi3Qhpw58Juw=",
        version = "v0.0.0-20190821133035-82a99dc22ef4",
    )
    go_repository(
        name = "com_github_hashicorp_terraform_svchost",
        importpath = "github.com/hashicorp/terraform-svchost",
        sum = "h1:hjyO2JsNZUKT1ym+FAdlBEkGPevazYsmVgIMw7dVELg=",
        version = "v0.0.0-20191011084731-65d371908596",
    )
    go_repository(
        name = "com_github_hashicorp_vault",
        importpath = "github.com/hashicorp/vault",
        sum = "h1:4x0lHxui/ZRp/B3E0Auv1QNBJpzETqHR2kQD3mHSBJU=",
        version = "v0.10.4",
    )
    go_repository(
        name = "com_github_hashicorp_yamux",
        importpath = "github.com/hashicorp/yamux",
        sum = "h1:kJCB4vdITiW1eC1vq2e6IsrXKrZit1bv/TDYFGMp4BQ=",
        version = "v0.0.0-20181012175058-2f1d1f20f75d",
    )
    go_repository(
        name = "com_github_hpcloud_tail",
        importpath = "github.com/hpcloud/tail",
        sum = "h1:nfCOvKYfkgYP8hkirhJocXT2+zOD8yUNjXaWfTlyFKI=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_jessevdk_go_flags",
        importpath = "github.com/jessevdk/go-flags",
        sum = "h1:4IU2WS7AumrZ/40jfhf4QVDMsQwqA7VEHozFRrGARJA=",
        version = "v1.4.0",
    )
    go_repository(
        name = "com_github_jmespath_go_jmespath",
        importpath = "github.com/jmespath/go-jmespath",
        sum = "h1:pmfjZENx5imkbgOkpRUYLnmbU7UEFbjtDA2hxJ1ichM=",
        version = "v0.0.0-20180206201540-c2b33e8439af",
    )
    go_repository(
        name = "com_github_jonboulle_clockwork",
        importpath = "github.com/jonboulle/clockwork",
        sum = "h1:VKV+ZcuP6l3yW9doeqz6ziZGgcynBVQO+obU0+0hcPo=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_joyent_triton_go",
        importpath = "github.com/joyent/triton-go",
        sum = "h1:JHCT6xuyPUrbbgAPE/3dqlvUKzRHMNuTBKKUb6OeR/k=",
        version = "v0.0.0-20180628001255-830d2b111e62",
    )
    go_repository(
        name = "com_github_json_iterator_go",
        importpath = "github.com/json-iterator/go",
        sum = "h1:gL2yXlmiIo4+t+y32d4WGwOjKGYcGOuyrg46vadswDE=",
        version = "v1.1.5",
    )
    go_repository(
        name = "com_github_jstemmer_go_junit_report",
        importpath = "github.com/jstemmer/go-junit-report",
        sum = "h1:rBMNdlhTLzJjJSDIjNEXX1Pz3Hmwmz91v+zycvx9PJc=",
        version = "v0.0.0-20190106144839-af01ea7f8024",
    )
    go_repository(
        name = "com_github_jtolds_gls",
        importpath = "github.com/jtolds/gls",
        sum = "h1:fSuqC+Gmlu6l/ZYAoZzx2pyucC8Xza35fpRVWLVmUEE=",
        version = "v4.2.1+incompatible",
    )
    go_repository(
        name = "com_github_julienschmidt_httprouter",
        importpath = "github.com/julienschmidt/httprouter",
        sum = "h1:TDTW5Yz1mjftljbcKqRcrYhd4XeOoI98t+9HbQbYf7g=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_kardianos_osext",
        importpath = "github.com/kardianos/osext",
        sum = "h1:iQTw/8FWTuc7uiaSepXwyf3o52HaUYcV+Tu66S3F5GA=",
        version = "v0.0.0-20190222173326-2bc1f35cddc0",
    )
    go_repository(
        name = "com_github_keybase_go_crypto",
        importpath = "github.com/keybase/go-crypto",
        sum = "h1:NARVGAAgEXvoMeNPHhPFt1SBt1VMznA3Gnz9d0qj+co=",
        version = "v0.0.0-20161004153544-93f5b35093ba",
    )
    go_repository(
        name = "com_github_konsorten_go_windows_terminal_sequences",
        importpath = "github.com/konsorten/go-windows-terminal-sequences",
        sum = "h1:mweAR1A6xJ3oS2pRaGiHgQ4OO8tzTaLawm8vnODuwDk=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_github_kr_logfmt",
        importpath = "github.com/kr/logfmt",
        sum = "h1:T+h1c/A9Gawja4Y9mFVWj2vyii2bbUNDw3kt9VxK2EY=",
        version = "v0.0.0-20140226030751-b84e30acd515",
    )
    go_repository(
        name = "com_github_kr_pretty",
        importpath = "github.com/kr/pretty",
        sum = "h1:L/CwN0zerZDmRFUapSPitk6f+Q3+0za1rQkzVuMiMFI=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_kr_pty",
        importpath = "github.com/kr/pty",
        sum = "h1:VkoXIwSboBpnk99O/KFauAEILuNHv5DVFKZMBN/gUgw=",
        version = "v1.1.1",
    )
    go_repository(
        name = "com_github_kr_text",
        importpath = "github.com/kr/text",
        sum = "h1:45sCR5RtlFHMR4UwH9sdQ5TC8v0qDQCHnXt+kaKSTVE=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_kylelemons_godebug",
        importpath = "github.com/kylelemons/godebug",
        sum = "h1:RPNrshWIDI6G2gRW9EHilWtl7Z6Sb1BR0xunSBf0SNc=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_lib_pq",
        importpath = "github.com/lib/pq",
        sum = "h1:X5PMW56eZitiTeO7tKzZxFCSpbFZJtkMMooicw2us9A=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_lusis_go_artifactory",
        importpath = "github.com/lusis/go-artifactory",
        sum = "h1:wnfcqULT+N2seWf6y4yHzmi7GD2kNx4Ute0qArktD48=",
        version = "v0.0.0-20160115162124-7e4ce345df82",
    )
    go_repository(
        name = "com_github_masterzen_simplexml",
        importpath = "github.com/masterzen/simplexml",
        sum = "h1:SmVbOZFWAlyQshuMfOkiAx1f5oUTsOGG5IXplAEYeeM=",
        version = "v0.0.0-20160608183007-4572e39b1ab9",
    )
    go_repository(
        name = "com_github_masterzen_winrm",
        importpath = "github.com/masterzen/winrm",
        sum = "h1:/1RFh2SLCJ+tEnT73+Fh5R2AO89sQqs8ba7o+hx1G0Y=",
        version = "v0.0.0-20190223112901-5e5c9a7fe54b",
    )
    go_repository(
        name = "com_github_mattn_go_colorable",
        importpath = "github.com/mattn/go-colorable",
        sum = "h1:/bC9yWikZXAL9uJdulbSfyVNIR3n3trXl+v8+1sx8mU=",
        version = "v0.1.2",
    )
    go_repository(
        name = "com_github_mattn_go_runewidth",
        importpath = "github.com/mattn/go-runewidth",
        sum = "h1:2BvfKmzob6Bmd4YsL0zygOqfdFnK7GR4QL06Do4/p7Y=",
        version = "v0.0.4",
    )
    go_repository(
        name = "com_github_mattn_go_shellwords",
        importpath = "github.com/mattn/go-shellwords",
        sum = "h1:xmZZyxuP+bYKAKkA9ABYXVNJ+G/Wf3R8d8vAP3LDJJk=",
        version = "v1.0.4",
    )
    go_repository(
        name = "com_github_matttproud_golang_protobuf_extensions",
        importpath = "github.com/matttproud/golang_protobuf_extensions",
        sum = "h1:4hp9jkHxhMHkqkrB3Ix0jegS5sx/RkqARlsWZ6pIwiU=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_github_miekg_dns",
        importpath = "github.com/miekg/dns",
        sum = "h1:WMhc1ik4LNkTg8U9l3hI1LvxKmIL+f1+WV/SZtCbDDA=",
        version = "v1.1.12",
    )
    go_repository(
        name = "com_github_mitchellh_colorstring",
        importpath = "github.com/mitchellh/colorstring",
        sum = "h1:62I3jR2EmQ4l5rM/4FEfDWcRD+abF5XlKShorW5LRoQ=",
        version = "v0.0.0-20190213212951-d06e56a500db",
    )
    go_repository(
        name = "com_github_mitchellh_copystructure",
        importpath = "github.com/mitchellh/copystructure",
        sum = "h1:Laisrj+bAB6b/yJwB5Bt3ITZhGJdqmxquMKeZ+mmkFQ=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_mitchellh_go_linereader",
        importpath = "github.com/mitchellh/go-linereader",
        sum = "h1:GRiLv4rgyqjqzxbhJke65IYUf4NCOOvrPOJbV/sPxkM=",
        version = "v0.0.0-20190213213312-1b945b3263eb",
    )
    go_repository(
        name = "com_github_mitchellh_go_testing_interface",
        importpath = "github.com/mitchellh/go-testing-interface",
        sum = "h1:fzU/JVNcaqHQEcVFAKeR41fkiLdIPrefOvVG1VZ96U0=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_mitchellh_go_wordwrap",
        importpath = "github.com/mitchellh/go-wordwrap",
        sum = "h1:6GlHJ/LTGMrIJbwgdqdl2eEH8o+Exx/0m8ir9Gns0u4=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_mitchellh_gox",
        importpath = "github.com/mitchellh/gox",
        sum = "h1:lfGJxY7ToLJQjHHwi0EX6uYBdK78egf954SQl13PQJc=",
        version = "v0.4.0",
    )
    go_repository(
        name = "com_github_mitchellh_hashstructure",
        importpath = "github.com/mitchellh/hashstructure",
        sum = "h1:ZkRJX1CyOoTkar7p/mLS5TZU4nJ1Rn/F8u9dGS02Q3Y=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_mitchellh_iochan",
        importpath = "github.com/mitchellh/iochan",
        sum = "h1:C+X3KsSTLFVBr/tK1eYN/vs4rJcvsiLU338UhYPJWeY=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_mitchellh_mapstructure",
        importpath = "github.com/mitchellh/mapstructure",
        sum = "h1:fmNYVwqnSfB9mZU6OS2O6GsXM+wcskZDuKQzvN1EDeE=",
        version = "v1.1.2",
    )
    go_repository(
        name = "com_github_mitchellh_panicwrap",
        importpath = "github.com/mitchellh/panicwrap",
        sum = "h1:67zIyVakCIvcs69A0FGfZjBdPleaonSgGlXRSRlb6fE=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_mitchellh_prefixedio",
        importpath = "github.com/mitchellh/prefixedio",
        sum = "h1:eD92Am0Qf3rqhsOeA1zwBHSfRkoHrt4o6uORamdmJP8=",
        version = "v0.0.0-20190213213902-5733675afd51",
    )
    go_repository(
        name = "com_github_mitchellh_reflectwalk",
        importpath = "github.com/mitchellh/reflectwalk",
        sum = "h1:FVzMWA5RllMAKIdUSC8mdWo3XtwoecrH79BY70sEEpE=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_github_modern_go_concurrent",
        importpath = "github.com/modern-go/concurrent",
        sum = "h1:TRLaZ9cD/w8PVh93nsPXa1VrQ6jlwL5oN8l14QlcNfg=",
        version = "v0.0.0-20180306012644-bacd9c7ef1dd",
    )
    go_repository(
        name = "com_github_mwitkow_go_conntrack",
        importpath = "github.com/mwitkow/go-conntrack",
        sum = "h1:F9x/1yl3T2AeKLr2AMdilSD8+f9bvMnNN8VS5iDtovc=",
        version = "v0.0.0-20161129095857-cc309e4a2223",
    )
    go_repository(
        name = "com_github_nu7hatch_gouuid",
        importpath = "github.com/nu7hatch/gouuid",
        sum = "h1:VhgPp6v9qf9Agr/56bj7Y/xa04UccTW04VP0Qed4vnQ=",
        version = "v0.0.0-20131221200532-179d4d0c4d8d",
    )
    go_repository(
        name = "com_github_oklog_run",
        importpath = "github.com/oklog/run",
        sum = "h1:Ru7dDtJNOyC66gQ5dQmaCa0qIsAUFY3sFpK1Xk8igrw=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_onsi_ginkgo",
        importpath = "github.com/onsi/ginkgo",
        sum = "h1:VkHVNpR4iVnU8XQR6DBm8BqYjN7CRzw+xKUbVVbbW9w=",
        version = "v1.8.0",
    )
    go_repository(
        name = "com_github_onsi_gomega",
        importpath = "github.com/onsi/gomega",
        sum = "h1:izbySO9zDPmjJ8rDjLvkA2zJHIo+HkYXHnf7eN7SSyo=",
        version = "v1.5.0",
    )
    go_repository(
        name = "com_github_packer_community_winrmcp",
        importpath = "github.com/packer-community/winrmcp",
        sum = "h1:m3CEgv3ah1Rhy82L+c0QG/U3VyY1UsvsIdkh0/rU97Y=",
        version = "v0.0.0-20180102160824-81144009af58",
    )
    go_repository(
        name = "com_github_pascaldekloe_goe",
        importpath = "github.com/pascaldekloe/goe",
        sum = "h1:cBOtyMzM9HTpWjXfbbunk26uA6nG3a8n06Wieeh0MwY=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_pkg_browser",
        importpath = "github.com/pkg/browser",
        sum = "h1:49lOXmGaUpV9Fz3gd7TFZY106KVlPVa5jcYD1gaQf98=",
        version = "v0.0.0-20180916011732-0a3d74bf9ce4",
    )
    go_repository(
        name = "com_github_pkg_errors",
        importpath = "github.com/pkg/errors",
        sum = "h1:iURUrRGxPUNPdy5/HRSm+Yj6okJ6UtLINN0Q9M4+h3I=",
        version = "v0.8.1",
    )
    go_repository(
        name = "com_github_pmezard_go_difflib",
        importpath = "github.com/pmezard/go-difflib",
        sum = "h1:4DBwDE0NGyQoBHbLQYPwSUPoCMWR5BEzIk/f1lZbAQM=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_prometheus_client_golang",
        importpath = "github.com/prometheus/client_golang",
        sum = "h1:9iH4JKXLzFbOAdtqv/a+j8aewx2Y8lAjAydhbaScPF8=",
        version = "v0.9.3",
    )
    go_repository(
        name = "com_github_prometheus_client_model",
        importpath = "github.com/prometheus/client_model",
        sum = "h1:gQz4mCbXsO+nc9n1hCxHcGA3Zx3Eo+UHZoInFGUIXNM=",
        version = "v0.0.0-20190812154241-14fe0d1b01d4",
    )
    go_repository(
        name = "com_github_prometheus_common",
        importpath = "github.com/prometheus/common",
        sum = "h1:7etb9YClo3a6HjLzfl6rIQaU+FDfi0VSX39io3aQ+DM=",
        version = "v0.4.0",
    )
    go_repository(
        name = "com_github_prometheus_procfs",
        importpath = "github.com/prometheus/procfs",
        sum = "h1:Z5QMcUKnQw7ouB1wDuyZM6TL/rm+brJcNk6Ai8ut3zM=",
        version = "v0.0.0-20190519111021-9935e8e0588d",
    )
    go_repository(
        name = "com_github_rogpeppe_fastuuid",
        importpath = "github.com/rogpeppe/fastuuid",
        sum = "h1:gu+uRPtBe88sKxUCEXRoeCvVG90TJmwhiqRpvdhQFng=",
        version = "v0.0.0-20150106093220-6724a57986af",
    )
    go_repository(
        name = "com_github_ryanuber_columnize",
        importpath = "github.com/ryanuber/columnize",
        sum = "h1:j1Wcmh8OrK4Q7GXY+V7SVSY8nUWQxHW5TkBe7YUl+2s=",
        version = "v2.1.0+incompatible",
    )
    go_repository(
        name = "com_github_satori_go_uuid",
        importpath = "github.com/satori/go.uuid",
        sum = "h1:0uYX9dsZ2yD7q2RtLRtPSdGDWzjeM3TbMJP9utgA0ww=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_sean__seed",
        importpath = "github.com/sean-/seed",
        sum = "h1:nn5Wsu0esKSJiIVhscUtVbo7ada43DJhG55ua/hjS5I=",
        version = "v0.0.0-20170313163322-e2103e2c3529",
    )
    go_repository(
        name = "com_github_sergi_go_diff",
        importpath = "github.com/sergi/go-diff",
        sum = "h1:Kpca3qRNrduNnOQeazBd0ysaKrUJiIuISHxogkT9RPQ=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_sirupsen_logrus",
        importpath = "github.com/sirupsen/logrus",
        sum = "h1:juTguoYk5qI21pwyTXY3B3Y5cOTH3ZUyZCg1v/mihuo=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_smartystreets_assertions",
        importpath = "github.com/smartystreets/assertions",
        sum = "h1:zE9ykElWQ6/NYmHa3jpm/yHnI4xSofP+UP6SpjHcSeM=",
        version = "v0.0.0-20180927180507-b2de0cb4f26d",
    )
    go_repository(
        name = "com_github_smartystreets_goconvey",
        importpath = "github.com/smartystreets/goconvey",
        sum = "h1:E+gaaifzi2xF65PbDmuKI3PhLWY6G5opMLniFq8vmXA=",
        version = "v0.0.0-20190222223459-a17d461953aa",
    )
    go_repository(
        name = "com_github_soheilhy_cmux",
        importpath = "github.com/soheilhy/cmux",
        sum = "h1:0HKaf1o97UwFjHH9o5XsHUOF+tqmdA7KEzXLpiyaw0E=",
        version = "v0.1.4",
    )
    go_repository(
        name = "com_github_spf13_afero",
        importpath = "github.com/spf13/afero",
        sum = "h1:qgMbHoJbPbw579P+1zVY+6n4nIFuIchaIjzZ/I/Yq8M=",
        version = "v1.2.1",
    )
    go_repository(
        name = "com_github_spf13_pflag",
        importpath = "github.com/spf13/pflag",
        sum = "h1:zPAT6CGy6wXeQ7NtTnaTerfKOsV6V6F8agHXFiazDkg=",
        version = "v1.0.3",
    )
    go_repository(
        name = "com_github_stretchr_testify",
        importpath = "github.com/stretchr/testify",
        sum = "h1:2E4SXV/wtOkTonXsotYi4li6zVWxYlZuYNCXe9XRJyk=",
        version = "v1.4.0",
    )
    go_repository(
        name = "com_github_svanharmelen_jsonapi",
        importpath = "github.com/svanharmelen/jsonapi",
        sum = "h1:Z4EH+5EffvBEhh37F0C0DnpklTMh00JOkjW5zK3ofBI=",
        version = "v0.0.0-20180618144545-0c0828c3f16d",
    )
    go_repository(
        name = "com_github_terraform_providers_terraform_provider_openstack",
        importpath = "github.com/terraform-providers/terraform-provider-openstack",
        sum = "h1:adpjqej+F8BAX9dHmuPF47sUIkgifeqBu6p7iCsyj0Y=",
        version = "v1.15.0",
    )
    go_repository(
        name = "com_github_tmc_grpc_websocket_proxy",
        importpath = "github.com/tmc/grpc-websocket-proxy",
        sum = "h1:lYIiVDtZnyTWlNwiAxLj0bbpTcx1BWCFhXjfsvmPdNc=",
        version = "v0.0.0-20171017195756-830351dc03c6",
    )
    go_repository(
        name = "com_github_ugorji_go",
        importpath = "github.com/ugorji/go",
        sum = "h1:j4s+tAvLfL3bZyefP2SEWmhBzmuIlH/eqNuPdFPgngw=",
        version = "v1.1.4",
    )
    go_repository(
        name = "com_github_ulikunitz_xz",
        importpath = "github.com/ulikunitz/xz",
        sum = "h1:pFrO0lVpTBXLpYw+pnLj6TbvHuyjXMfjGeCwSqCVwok=",
        version = "v0.5.5",
    )
    go_repository(
        name = "com_github_unknwon_com",
        importpath = "github.com/Unknwon/com",
        sum = "h1:tuQ7w+my8a8mkwN7x2TSd7OzTjkZ7rAeSyH4xncuAMI=",
        version = "v0.0.0-20151008135407-28b053d5a292",
    )
    go_repository(
        name = "com_github_vmihailenco_msgpack",
        importpath = "github.com/vmihailenco/msgpack",
        sum = "h1:RMF1enSPeKTlXrXdOcqjFUElywVZjjC6pqse21bKbEU=",
        version = "v4.0.1+incompatible",
    )
    go_repository(
        name = "com_github_xanzy_ssh_agent",
        importpath = "github.com/xanzy/ssh-agent",
        sum = "h1:TCbipTQL2JiiCprBWx9frJ2eJlCYT00NmctrHxVAr70=",
        version = "v0.2.1",
    )
    go_repository(
        name = "com_github_xiang90_probing",
        importpath = "github.com/xiang90/probing",
        sum = "h1:MPPkRncZLN9Kh4MEFmbnK4h3BD7AUmskWv2+EeZJCCs=",
        version = "v0.0.0-20160813154853-07dd2e8dfe18",
    )
    go_repository(
        name = "com_github_xlab_treeprint",
        importpath = "github.com/xlab/treeprint",
        sum = "h1:Jpn2j6wHkC9wJv5iMfJhKqrZJx3TahFx+7sbZ7zQdxs=",
        version = "v0.0.0-20161029104018-1d6e34225557",
    )
    go_repository(
        name = "com_github_zclconf_go_cty",
        importpath = "github.com/zclconf/go-cty",
        sum = "h1:sPHsy7ADcIZQP3vILvTjrh74ZA175TFP5vqiNK1UmlI=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_zclconf_go_cty_yaml",
        importpath = "github.com/zclconf/go-cty-yaml",
        sum = "h1:up11wlgAaDvlAGENcFDnZgkn0qUJurso7k6EpURKNF8=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_google_cloud_go",
        build_directives = [
            "gazelle:go_visibility @com_google_cloud_go_storage//:__subpackages__",
        ],
        importpath = "cloud.google.com/go",
        sum = "h1:lRi0CHyU+ytlvylOlFKKq0af6JncuyoRh1J+QJBqQx0=",
        version = "v0.45.1",
    )
    go_repository(
        name = "com_google_cloud_go_bigquery",
        importpath = "cloud.google.com/go/bigquery",
        sum = "h1:hL+ycaJpVE9M7nLoiXb/Pn10ENE2u+oddxbD8uu0ZVU=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_google_cloud_go_datastore",
        importpath = "cloud.google.com/go/datastore",
        sum = "h1:Kt+gOPPp2LEPWp8CSfxhsM8ik9CcyE/gYu+0r+RnZvM=",
        version = "v1.0.0",
    )
    go_repository(
        name = "in_gopkg_alecthomas_kingpin_v2",
        importpath = "gopkg.in/alecthomas/kingpin.v2",
        sum = "h1:jMFz6MfLP0/4fUyZle81rXUoxOBFi19VUFKVDOQfozc=",
        version = "v2.2.6",
    )
    go_repository(
        name = "in_gopkg_check_v1",
        importpath = "gopkg.in/check.v1",
        sum = "h1:qIbj1fsPNlZgppZ+VLlY7N33q108Sa+fhmuc+sWQYwY=",
        version = "v1.0.0-20180628173108-788fd7840127",
    )
    go_repository(
        name = "in_gopkg_cheggaaa_pb_v1",
        importpath = "gopkg.in/cheggaaa/pb.v1",
        sum = "h1:n1tBJnnK2r7g9OW2btFH91V92STTUevLXYFb8gy9EMk=",
        version = "v1.0.28",
    )
    go_repository(
        name = "in_gopkg_fsnotify_v1",
        importpath = "gopkg.in/fsnotify.v1",
        sum = "h1:xOHLXZwVvI9hhs+cLKq5+I5onOuwQLhQwiu63xxlHs4=",
        version = "v1.4.7",
    )
    go_repository(
        name = "in_gopkg_ini_v1",
        importpath = "gopkg.in/ini.v1",
        sum = "h1:7N3gPTt50s8GuLortA00n8AqRTk75qOP98+mTPpgzRk=",
        version = "v1.42.0",
    )
    go_repository(
        name = "in_gopkg_resty_v1",
        importpath = "gopkg.in/resty.v1",
        sum = "h1:CuXP0Pjfw9rOuY6EP+UvtNvt5DSqHpIxILZKT/quCZI=",
        version = "v1.12.0",
    )
    go_repository(
        name = "in_gopkg_tomb_v1",
        importpath = "gopkg.in/tomb.v1",
        sum = "h1:uRGJdciOHaEIrze2W8Q3AKkepLTh2hOroT7a+7czfdQ=",
        version = "v1.0.0-20141024135613-dd632973f1e7",
    )
    go_repository(
        name = "in_gopkg_yaml_v2",
        importpath = "gopkg.in/yaml.v2",
        sum = "h1:ZCJp+EgiOT7lHqUV2J862kp8Qj64Jo6az82+3Td9dZw=",
        version = "v2.2.2",
    )
    go_repository(
        name = "io_k8s_sigs_yaml",
        importpath = "sigs.k8s.io/yaml",
        sum = "h1:4A07+ZFc2wgJwo8YNlQpr1rVlgUDlxXHhPJciaPY5gs=",
        version = "v1.1.0",
    )
    go_repository(
        name = "io_opencensus_go",
        importpath = "go.opencensus.io",
        sum = "h1:C9hSCOW830chIVkdja34wa6Ky+IzWllkUinR+BtRZd4=",
        version = "v0.22.0",
    )
    go_repository(
        name = "io_rsc_binaryregexp",
        importpath = "rsc.io/binaryregexp",
        sum = "h1:HfqmD5MEmC0zvwBuF187nq9mdnXjXsSivRiXN7SmRkE=",
        version = "v0.2.0",
    )
    go_repository(
        name = "net_howett_plist",
        importpath = "howett.net/plist",
        sum = "h1:jhnBjNi9UFpfpl8YZhA9CrOqpnJdvzuiHsl/dnxl11M=",
        version = "v0.0.0-20181124034731-591f970eefbb",
    )
    go_repository(
        name = "org_golang_google_api",
        importpath = "google.golang.org/api",
        sum = "h1:jbyannxz0XFD3zdjgrSUsaJbgpH4eTrkdhRChkHPfO8=",
        version = "v0.9.0",
    )
    go_repository(
        name = "org_golang_google_appengine",
        importpath = "google.golang.org/appengine",
        sum = "h1:QzqyMA1tlu6CgqCDUtU9V+ZKhLFT2dkJuANu5QaxI3I=",
        version = "v1.6.1",
    )
    go_repository(
        name = "org_golang_google_genproto",
        importpath = "google.golang.org/genproto",
        sum = "h1:1x8rC5/IgdLMPbPTvlQTN28+rcy8XL9Q19UWUMDyqYs=",
        version = "v0.0.0-20191223191004-3caeed10a8bf",
    )
    go_repository(
        name = "org_golang_x_crypto",
        importpath = "golang.org/x/crypto",
        sum = "h1:HuIa8hRrWRSrqYzx1qI49NNxhdi2PrY7gxVSq1JjLDc=",
        version = "v0.0.0-20190701094942-4def268fd1a4",
    )
    go_repository(
        name = "org_golang_x_exp",
        importpath = "golang.org/x/exp",
        sum = "h1:OeRHuibLsmZkFj773W4LcfAGsSxJgfPONhr8cmO+eLA=",
        version = "v0.0.0-20190510132918-efd6b22b2522",
    )
    go_repository(
        name = "org_golang_x_image",
        importpath = "golang.org/x/image",
        sum = "h1:KYGJGHOQy8oSi1fDlSpcZF0+juKwk/hEMv5SiwHogR0=",
        version = "v0.0.0-20190227222117-0694c2d4d067",
    )
    go_repository(
        name = "org_golang_x_lint",
        importpath = "golang.org/x/lint",
        sum = "h1:QzoH/1pFpZguR8NrRHLcO6jKqfv2zpuSqZLgdm7ZmjI=",
        version = "v0.0.0-20190409202823-959b441ac422",
    )
    go_repository(
        name = "org_golang_x_mobile",
        importpath = "golang.org/x/mobile",
        sum = "h1:Tus/Y4w3V77xDsGwKUC8a/QrV7jScpU557J77lFffNs=",
        version = "v0.0.0-20190312151609-d3739f865fa6",
    )
    go_repository(
        name = "org_golang_x_oauth2",
        importpath = "golang.org/x/oauth2",
        sum = "h1:SVwTIAaPC2U/AvvLNZ2a7OVsmBpC8L5BlwK1whH3hm0=",
        version = "v0.0.0-20190604053449-0f29369cfe45",
    )
    go_repository(
        name = "org_golang_x_sync",
        importpath = "golang.org/x/sync",
        sum = "h1:8gQV6CLnAEikrhgkHFbMAEhagSSnXWGV915qUMm9mrU=",
        version = "v0.0.0-20190423024810-112230192c58",
    )
    go_repository(
        name = "org_golang_x_sys",
        importpath = "golang.org/x/sys",
        sum = "h1:Vco5b+cuG5NNfORVxZy6bYZQ7rsigisU1WQFkvQ0L5E=",
        version = "v0.0.0-20191002063906-3421d5a6bb1c",
    )
    go_repository(
        name = "org_golang_x_time",
        importpath = "golang.org/x/time",
        sum = "h1:SvFZT6jyqRaOeXpc5h/JSfZenJ2O330aBsf7JfSUXmQ=",
        version = "v0.0.0-20190308202827-9d24e82272b4",
    )
    go_repository(
        name = "org_golang_x_tools",
        importpath = "golang.org/x/tools",
        sum = "h1:PhtdWYteEBebOX7KXm4qkIAVSUTHQ883/2hRB92r9lk=",
        version = "v0.0.0-20190909030654-5b82db07426d",
    )
    go_repository(
        name = "org_uber_go_atomic",
        importpath = "go.uber.org/atomic",
        sum = "h1:cxzIVoETapQEqDhQu3QfnvXAV4AlzcvUCxkVUFw3+EU=",
        version = "v1.4.0",
    )
    go_repository(
        name = "org_uber_go_multierr",
        importpath = "go.uber.org/multierr",
        sum = "h1:HoEmRHQPVSqub6w2z2d2EOVs2fjyFRGyofhKuyDq0QI=",
        version = "v1.1.0",
    )
    go_repository(
        name = "org_uber_go_zap",
        importpath = "go.uber.org/zap",
        sum = "h1:XCJQEf3W6eZaVwhRBof6ImoYGJSITeKWsyeh3HFu/5o=",
        version = "v1.9.1",
    )
    go_repository(
        name = "com_github_azure_go_autorest",
        importpath = "github.com/Azure/go-autorest",
        sum = "h1:nhKI/bvazIs3C3TFGoSqKY6hZ8f5od5mb5/UcS6HVIY=",
        version = "v10.15.3+incompatible",
    )
    go_repository(
        name = "com_github_circonus_labs_circonus_gometrics",
        importpath = "github.com/circonus-labs/circonus-gometrics",
        sum = "h1:C29Ae4G5GtYyYMm1aztcyj/J5ckgJm2zwdDajFbx1NY=",
        version = "v2.3.1+incompatible",
    )
    go_repository(
        name = "com_github_circonus_labs_circonusllhist",
        importpath = "github.com/circonus-labs/circonusllhist",
        sum = "h1:TJH+oke8D16535+jHExHj4nQvzlZrj7ug5D7I/orNUA=",
        version = "v0.1.3",
    )
    go_repository(
        name = "com_github_coredns_coredns",
        importpath = "github.com/coredns/coredns",
        sum = "h1:bAFHrSsBeTeRG5W3Nf2su3lUGw7Npw2UKeCJm/3A638=",
        version = "v1.1.2",
    )
    go_repository(
        name = "com_github_datadog_datadog_go",
        importpath = "github.com/DataDog/datadog-go",
        sum = "h1:V5BKkxACZLjzHjSgBbr2gvLA2Ae49yhc6CSY7MLy5k4=",
        version = "v2.2.0+incompatible",
    )
    go_repository(
        name = "com_github_denverdino_aliyungo",
        importpath = "github.com/denverdino/aliyungo",
        sum = "h1:lrWnAyy/F72MbxIxFUzKmcMCdt9Oi8RzpAxzTNQHD7o=",
        version = "v0.0.0-20170926055100-d3308649c661",
    )
    go_repository(
        name = "com_github_digitalocean_godo",
        importpath = "github.com/digitalocean/godo",
        sum = "h1:uW1/FcvZE/hoixnJcnlmIUvTVNdZCLjRLzmDtRi1xXY=",
        version = "v1.10.0",
    )
    go_repository(
        name = "com_github_docker_go_connections",
        importpath = "github.com/docker/go-connections",
        sum = "h1:3lOnM9cSzgGwx8VfK/NGOW5fLQ0GjIlCkaktF+n1M6o=",
        version = "v0.3.0",
    )
    go_repository(
        name = "com_github_elazarl_go_bindata_assetfs",
        importpath = "github.com/elazarl/go-bindata-assetfs",
        sum = "h1:ZoRgc53qJCfSLimXqJDrmBhnt5GChDsExMCK7t48o0Y=",
        version = "v0.0.0-20160803192304-e1a2a7ec64b0",
    )
    go_repository(
        name = "com_github_fatih_structs",
        importpath = "github.com/fatih/structs",
        sum = "h1:Q7juDM0QtcnhCpeyLGQKyg4TOIghuNXrkL32pHAUMxo=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_go_ini_ini",
        importpath = "github.com/go-ini/ini",
        sum = "h1:Mujh4R/dH6YL8bxuISne3xX2+qcQ9p0IxKAP6ExWoUo=",
        version = "v1.25.4",
    )
    go_repository(
        name = "com_github_go_ldap_ldap",
        importpath = "github.com/go-ldap/ldap",
        sum = "h1:kD5HQcAzlQ7yrhfn+h+MSABeAy/jAJhvIJ/QDllP44g=",
        version = "v3.0.2+incompatible",
    )
    go_repository(
        name = "com_github_go_ole_go_ole",
        importpath = "github.com/go-ole/go-ole",
        sum = "h1:2lOsA72HgjxAuMlKpFiCbHTvu44PIVkZ5hqm3RSdI/E=",
        version = "v1.2.1",
    )
    go_repository(
        name = "com_github_gogo_googleapis",
        importpath = "github.com/gogo/googleapis",
        sum = "h1:kFkMAZBNAn4j7K0GiZr8cRYzejq68VbheufiV3YuyFI=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_google_gofuzz",
        importpath = "github.com/google/gofuzz",
        sum = "h1:+RRA9JqSOZFfKrOeqr2z77+8R2RKyh8PG66dcu1V0ck=",
        version = "v0.0.0-20170612174753-24818f796faf",
    )
    go_repository(
        name = "com_github_googleapis_gnostic",
        importpath = "github.com/googleapis/gnostic",
        sum = "h1:l6N3VoaVzTncYYW+9yOz2LJJammFZGBO13sqgEhpy9g=",
        version = "v0.2.0",
    )
    go_repository(
        name = "com_github_gregjones_httpcache",
        importpath = "github.com/gregjones/httpcache",
        sum = "h1:pdN6V1QBWetyv/0+wjACpqVH+eVULgEjkurDLq3goeM=",
        version = "v0.0.0-20180305231024-9cad4c3443a7",
    )
    go_repository(
        name = "com_github_hashicorp_go_bexpr",
        importpath = "github.com/hashicorp/go-bexpr",
        sum = "h1:ijMXI4qERbzxbCnkxmfUtwMyjrrk3y+Vt0MxojNCbBs=",
        version = "v0.1.2",
    )
    go_repository(
        name = "com_github_hashicorp_go_discover",
        importpath = "github.com/hashicorp/go-discover",
        sum = "h1:SynRxs8h2h7lLSA5py5a3WWkYpImhREtju0CuRd97wc=",
        version = "v0.0.0-20190403160810-22221edb15cd",
    )
    go_repository(
        name = "com_github_hashicorp_go_memdb",
        importpath = "github.com/hashicorp/go-memdb",
        sum = "h1:yxxFgVz31vFoKKTtRUNbXLNe4GFnbLKqg+0N7yG42L8=",
        version = "v0.0.0-20180223233045-1289e7fffe71",
    )
    go_repository(
        name = "com_github_hashicorp_go_raftchunking",
        importpath = "github.com/hashicorp/go-raftchunking",
        sum = "h1:moEnaG3gcwsWNyIBJoD5PCByE+Ewkqxh6N05CT+MbwA=",
        version = "v0.6.1",
    )
    go_repository(
        name = "com_github_hashicorp_net_rpc_msgpackrpc",
        importpath = "github.com/hashicorp/net-rpc-msgpackrpc",
        sum = "h1:lc3c72qGlIMDqQpQH82Y4vaglRMMFdJbziYWriR4UcE=",
        version = "v0.0.0-20151116020338-a14192a58a69",
    )
    go_repository(
        name = "com_github_hashicorp_raft",
        importpath = "github.com/hashicorp/raft",
        sum = "h1:HJr7UE1x/JrJSc9Oy6aDBHtNHUUBHjcQjTgvUVihoZs=",
        version = "v1.1.1",
    )
    go_repository(
        name = "com_github_hashicorp_raft_boltdb",
        importpath = "github.com/hashicorp/raft-boltdb",
        sum = "h1:xykPFhrBAS2J0VBzVa5e80b5ZtYuNQtgXjN40qBZlD4=",
        version = "v0.0.0-20171010151810-6e5ba93211ea",
    )
    go_repository(
        name = "com_github_hashicorp_vault_api",
        importpath = "github.com/hashicorp/vault/api",
        sum = "h1:j08Or/wryXT4AcHj1oCbMd7IijXcKzYUGw59LGu9onU=",
        version = "v1.0.4",
    )
    go_repository(
        name = "com_github_hashicorp_vault_sdk",
        importpath = "github.com/hashicorp/vault/sdk",
        sum = "h1:mOEPeOhT7jl0J4AMl1E705+BcmeRs1VmKNb9F0sMLy8=",
        version = "v0.1.13",
    )
    go_repository(
        name = "com_github_hashicorp_vic",
        importpath = "github.com/hashicorp/vic",
        sum = "h1:O/pT5C1Q3mVXMyuqg7yuAWUg/jMZR1/0QTzTRdNR6Uw=",
        version = "v1.5.1-0.20190403131502-bbfe86ec9443",
    )
    go_repository(
        name = "com_github_imdario_mergo",
        importpath = "github.com/imdario/mergo",
        sum = "h1:xTNEAn+kxVO7dTZGu0CegyqKZmoWFI0rF8UxjlB2d28=",
        version = "v0.3.6",
    )
    go_repository(
        name = "com_github_jarcoal_httpmock",
        importpath = "github.com/jarcoal/httpmock",
        sum = "h1:FjHUJJ7oBW4G/9j1KzlHaXL09LyMVM9rupS39lncbXk=",
        version = "v0.0.0-20180424175123-9c70cfe4a1da",
    )
    go_repository(
        name = "com_github_kisielk_errcheck",
        importpath = "github.com/kisielk/errcheck",
        sum = "h1:reN85Pxc5larApoH1keMBiu2GWtPqXQ1nc9gx+jOU+E=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_kisielk_gotool",
        importpath = "github.com/kisielk/gotool",
        sum = "h1:AV2c/EiW3KqPNT9ZKl07ehoAGi4C5/01Cfbblndcapg=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_microsoft_go_winio",
        importpath = "github.com/Microsoft/go-winio",
        sum = "h1:M3NHMuPgMSUPdE5epwNUHlRPSVzHs8HpRTrVXhR0myo=",
        version = "v0.4.3",
    )
    go_repository(
        name = "com_github_nicolai86_scaleway_sdk",
        importpath = "github.com/nicolai86/scaleway-sdk",
        sum = "h1:BQ1HW7hr4IVovMwWg0E0PYcyW8CzqDcVmaew9cujU4s=",
        version = "v1.10.2-0.20180628010248-798f60e20bb2",
    )
    go_repository(
        name = "com_github_nytimes_gziphandler",
        importpath = "github.com/NYTimes/gziphandler",
        sum = "h1:iLrQrdwjDd52kHDA5op2UBJFjmOb9g+7scBan4RN8F0=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_github_packethost_packngo",
        importpath = "github.com/packethost/packngo",
        sum = "h1:vwpFWvAO8DeIZfFeqASzZfsxuWPno9ncAebBEP0N3uE=",
        version = "v0.1.1-0.20180711074735-b9cb5096f54c",
    )
    go_repository(
        name = "com_github_peterbourgon_diskv",
        importpath = "github.com/peterbourgon/diskv",
        sum = "h1:UBdAOUP5p4RWqPBg048CAvpKN+vxiaj6gdUUzhl4XmI=",
        version = "v2.0.1+incompatible",
    )
    go_repository(
        name = "com_github_pierrec_lz4",
        importpath = "github.com/pierrec/lz4",
        sum = "h1:2xWsjqPFWcplujydGg4WmhC/6fZqK42wMM8aXeqhl0I=",
        version = "v2.0.5+incompatible",
    )
    go_repository(
        name = "com_github_renier_xmlrpc",
        importpath = "github.com/renier/xmlrpc",
        sum = "h1:Wdi9nwnhFNAlseAOekn6B5G/+GMtks9UKbvRU/CMM/o=",
        version = "v0.0.0-20170708154548-ce4a1a486c03",
    )
    go_repository(
        name = "com_github_ryanuber_go_glob",
        importpath = "github.com/ryanuber/go-glob",
        sum = "h1:iQh3xXAumdQ+4Ufa5b25cRpC5TYKlno6hsv6Cb3pkBk=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_shirou_gopsutil",
        importpath = "github.com/shirou/gopsutil",
        sum = "h1:1Ge4j/3uB2rxzPWD3TC+daeCw+w91z8UCUL/7WH5gn8=",
        version = "v0.0.0-20181107111621-48177ef5f880",
    )
    go_repository(
        name = "com_github_shirou_w32",
        importpath = "github.com/shirou/w32",
        sum = "h1:udFKJ0aHUL60LboW/A+DfgoHVedieIzIXE8uylPue0U=",
        version = "v0.0.0-20160930032740-bb4de0191aa4",
    )
    go_repository(
        name = "com_github_softlayer_softlayer_go",
        importpath = "github.com/softlayer/softlayer-go",
        sum = "h1:bVQRCxQvfjNUeRqaY/uT0tFuvuFY0ulgnczuR684Xic=",
        version = "v0.0.0-20180806151055-260589d94c7d",
    )
    go_repository(
        name = "com_github_stackexchange_wmi",
        importpath = "github.com/StackExchange/wmi",
        sum = "h1:fLjPD/aNc3UIOA6tDi6QXUemppXK3P9BI7mr2hd6gx8=",
        version = "v0.0.0-20180116203802-5d049714c4a6",
    )
    go_repository(
        name = "com_github_tent_http_link_go",
        importpath = "github.com/tent/http-link-go",
        sum = "h1:/Bsw4C+DEdqPjt8vAqaC9LAqpAQnaCQQqmolqq3S1T4=",
        version = "v0.0.0-20130702225549-ac974c61c2f9",
    )
    go_repository(
        name = "com_github_tv42_httpunix",
        importpath = "github.com/tv42/httpunix",
        sum = "h1:G3dpKMzFDjgEh2q1Z7zUUtKa8ViPtH+ocF0bE0g00O8=",
        version = "v0.0.0-20150427012821-b75d8614f926",
    )
    go_repository(
        name = "com_github_vmware_govmomi",
        importpath = "github.com/vmware/govmomi",
        sum = "h1:f7QxSmP7meCtoAmiKZogvVbLInT+CZx6Px6K5rYsJZo=",
        version = "v0.18.0",
    )
    go_repository(
        name = "in_gopkg_airbrake_gobrake_v2",
        importpath = "gopkg.in/airbrake/gobrake.v2",
        sum = "h1:7z2uVWwn7oVeeugY1DtlPAy5H+KYgB1KeKTnqjNatLo=",
        version = "v2.0.9",
    )
    go_repository(
        name = "in_gopkg_asn1_ber_v1",
        importpath = "gopkg.in/asn1-ber.v1",
        sum = "h1:TxyelI5cVkbREznMhfzycHdkp5cLA7DpE+GKjSslYhM=",
        version = "v1.0.0-20181015200546-f715ec2f112d",
    )
    go_repository(
        name = "in_gopkg_gemnasium_logrus_airbrake_hook_v2",
        importpath = "gopkg.in/gemnasium/logrus-airbrake-hook.v2",
        sum = "h1:OAj3g0cR6Dx/R07QgQe8wkA9RNjB2u4i700xBkIT4e0=",
        version = "v2.1.2",
    )
    go_repository(
        name = "in_gopkg_inf_v0",
        importpath = "gopkg.in/inf.v0",
        sum = "h1:73M5CoZyi3ZLMOyDlQh031Cx6N9NDJ2Vvfl76EDAgDc=",
        version = "v0.9.1",
    )
    go_repository(
        name = "in_gopkg_square_go_jose_v2",
        importpath = "gopkg.in/square/go-jose.v2",
        sum = "h1:SK5KegNXmKmqE342YYN2qPHEnUYeoMiXXl1poUlI+o4=",
        version = "v2.3.1",
    )
    go_repository(
        name = "io_istio_gogo_genproto",
        importpath = "istio.io/gogo-genproto",
        sum = "h1:m6oJj4SHNNgTcsQG0Wgx881zrYo8LkYy9w6SE3Gk8gM=",
        version = "v0.0.0-20190124151557-6d926a6e6feb",
    )
    go_repository(
        name = "io_k8s_api",
        importpath = "k8s.io/api",
        sum = "h1:9MWtbqhwTyDvF4cS1qAhxDb9Mi8taXiAu+5nEacl7gY=",
        version = "v0.0.0-20190325185214-7544f9db76f6",
    )
    go_repository(
        name = "io_k8s_apimachinery",
        importpath = "k8s.io/apimachinery",
        sum = "h1:Q4RZrHNtlC/mSdC1sTrcZ5RchC/9vxLVj57pWiCBKv4=",
        version = "v0.0.0-20190223001710-c182ff3b9841",
    )
    go_repository(
        name = "io_k8s_client_go",
        importpath = "k8s.io/client-go",
        sum = "h1:tTI4hRmb1DRMl4fG6Vclfdi6nTM82oIrTT7HfitmxC4=",
        version = "v8.0.0+incompatible",
    )
    go_repository(
        name = "io_etcd_go_etcd",
        build_directives = [
            "gazelle:proto disable",
        ],
        build_file_generation = "on",
        importpath = "go.etcd.io/etcd",
        sum = "h1:5aomL5mqoKHxw6NG+oYgsowk8tU8aOalo2IdZxdWHkw=",
        version = "v3.3.18+incompatible",
    )
    go_repository(
        name = "com_github_etcd_io_etcd",
        build_directives = [
            "gazelle:proto disable",
        ],
        build_file_generation = "on",
        importpath = "github.com/etcd-io/etcd",
        sum = "h1:iBOwlpFcpg+GJEiZ2ODqwkcu2UEK3y4aKFa3d8+LneM=",
        version = "v3.3.18+incompatible",
    )
    go_repository(
        name = "com_github_coreos_go_systemd_v22",
        importpath = "github.com/coreos/go-systemd/v22",
        sum = "h1:XJIw/+VlJ+87J+doOxznsAWIdmWuViOVhkQamW5YV28=",
        version = "v22.0.0",
    )
    go_repository(
        name = "com_github_qri_io_starlib",
        importpath = "github.com/qri-io/starlib",
        sum = "h1:aXcKv4p8WRoGElC+ZpfyKMP1OmT2KP5+iISPfXJXk6M=",
        version = "v0.4.1",
    )
    go_repository(
        name = "cc_mvdan_interfacer",
        importpath = "mvdan.cc/interfacer",
        sum = "h1:WX1yoOaKQfddO/mLzdV4wptyWgoH/6hwLs7QHTixo0I=",
        version = "v0.0.0-20180901003855-c20040233aed",
    )
    go_repository(
        name = "cc_mvdan_lint",
        importpath = "mvdan.cc/lint",
        sum = "h1:DxJ5nJdkhDlLok9K6qO+5290kphDJbHOQO1DFFFTeBo=",
        version = "v0.0.0-20170908181259-adc824a0674b",
    )
    go_repository(
        name = "cc_mvdan_unparam",
        importpath = "mvdan.cc/unparam",
        sum = "h1:duVSyluuJA+u0BnkcLR01smoLrGgDTfWt5c8ODYG8fU=",
        version = "v0.0.0-20190209190245-fbb59629db34",
    )
    go_repository(
        name = "com_github_360entsecgroup_skylar_excelize",
        importpath = "github.com/360EntSecGroup-Skylar/excelize",
        sum = "h1:l55mJb6rkkaUzOpSsgEeKYtS6/0gHwBYyfo5Jcjv/Ks=",
        version = "v1.4.1",
    )
    go_repository(
        name = "com_github_aead_siphash",
        importpath = "github.com/aead/siphash",
        sum = "h1:FwHfE/T45KPKYuuSAKyyvE+oPWcaQ+CUmFW0bPlM+kg=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_github_andreasbriese_bbloom",
        importpath = "github.com/AndreasBriese/bbloom",
        sum = "h1:PqzgE6kAMi81xWQA2QIVxjWkFHptGgC547vchpUbtFo=",
        version = "v0.0.0-20180913140656-343706a395b7",
    )
    go_repository(
        name = "com_github_andybalholm_cascadia",
        importpath = "github.com/andybalholm/cascadia",
        sum = "h1:hOCXnnZ5A+3eVDX8pvgl4kofXv2ELss0bKcqRySc45o=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_bifurcation_mint",
        importpath = "github.com/bifurcation/mint",
        sum = "h1:ArS0Fye/ZH2QmKVSj4JEf+9ARkXNe18jcFkmr4XPRiw=",
        version = "v0.0.0-20181105073638-824af6541065",
    )
    go_repository(
        name = "com_github_bren2010_proquint",
        importpath = "github.com/bren2010/proquint",
        sum = "h1:QgeLLoPD3kRVmeu/1al9iIpIANMi9O1zXFm8BnYGCJg=",
        version = "v0.0.0-20160323162903-38337c27106d",
    )
    go_repository(
        name = "com_github_btcsuite_btcd",
        importpath = "github.com/btcsuite/btcd",
        sum = "h1:m0N5Vg5nP3zEz8TREZpwX3gt4Biw3/8fbIf4A3hO96g=",
        version = "v0.0.0-20190427004231-96897255fd17",
    )
    go_repository(
        name = "com_github_btcsuite_btclog",
        importpath = "github.com/btcsuite/btclog",
        sum = "h1:bAs4lUbRJpnnkd9VhRV3jjAVU7DJVjMaK+IsvSeZvFo=",
        version = "v0.0.0-20170628155309-84c8d2346e9f",
    )
    go_repository(
        name = "com_github_btcsuite_btcutil",
        importpath = "github.com/btcsuite/btcutil",
        sum = "h1:yJzD/yFppdVCf6ApMkVy8cUxV0XrxdP9rVf6D87/Mng=",
        version = "v0.0.0-20190425235716-9e5f4b9a998d",
    )
    go_repository(
        name = "com_github_btcsuite_go_socks",
        importpath = "github.com/btcsuite/go-socks",
        sum = "h1:R/opQEbFEy9JGkIguV40SvRY1uliPX8ifOvi6ICsFCw=",
        version = "v0.0.0-20170105172521-4720035b7bfd",
    )
    go_repository(
        name = "com_github_btcsuite_goleveldb",
        importpath = "github.com/btcsuite/goleveldb",
        sum = "h1:Tvd0BfvqX9o823q1j2UZ/epQo09eJh6dTcRp79ilIN4=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_btcsuite_snappy_go",
        importpath = "github.com/btcsuite/snappy-go",
        sum = "h1:ZxaA6lo2EpxGddsA8JwWOcxlzRybb444sgmeJQMJGQE=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_btcsuite_websocket",
        importpath = "github.com/btcsuite/websocket",
        sum = "h1:R8vQdOQdZ9Y3SkEwmHoWBmX1DNXhXZqlTpq6s4tyJGc=",
        version = "v0.0.0-20150119174127-31079b680792",
    )
    go_repository(
        name = "com_github_btcsuite_winsvc",
        importpath = "github.com/btcsuite/winsvc",
        sum = "h1:J9B4L7e3oqhXOcm+2IuNApwzQec85lE+QaikUcCs+dk=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_cenkalti_backoff",
        importpath = "github.com/cenkalti/backoff",
        sum = "h1:tKJnvO2kl0zmb/jA5UKAt4VoEVw1qxKWjE/Bpp46npY=",
        version = "v2.1.1+incompatible",
    )
    go_repository(
        name = "com_github_cespare_xxhash",
        importpath = "github.com/cespare/xxhash",
        sum = "h1:a6HrQnmkObjyL+Gs60czilIUGqrzKutQD6XZog3p+ko=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_cheekybits_genny",
        importpath = "github.com/cheekybits/genny",
        sum = "h1:uGGa4nei+j20rOSeDeP5Of12XVm7TGUd4dJA9RDitfE=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_cskr_pubsub",
        importpath = "github.com/cskr/pubsub",
        sum = "h1:vlOzMhl6PFn60gRlTQQsIfVwaPB/B/8MziK8FhEPt/0=",
        version = "v1.0.2",
    )
    go_repository(
        name = "com_github_davidlazar_go_crypto",
        importpath = "github.com/davidlazar/go-crypto",
        sum = "h1:6xT9KW8zLC5IlbaIF5Q7JNieBoACT7iW0YTxQHR0in0=",
        version = "v0.0.0-20170701192655-dcfb0a7ac018",
    )
    go_repository(
        name = "com_github_dgraph_io_badger",
        importpath = "github.com/dgraph-io/badger",
        sum = "h1:7KPp6xv5+wymkVUbkAnZZXvmDrJlf09m/7u1HG5lAYA=",
        version = "v2.0.0-rc.2+incompatible",
    )
    go_repository(
        name = "com_github_dgryski_go_farm",
        importpath = "github.com/dgryski/go-farm",
        sum = "h1:dDxpBYafY/GYpcl+LS4Bn3ziLPuEdGRkRjYAbSlWxSA=",
        version = "v0.0.0-20190104051053-3adb47b1fb0f",
    )
    go_repository(
        name = "com_github_dgryski_go_sip13",
        importpath = "github.com/dgryski/go-sip13",
        sum = "h1:RMLoZVzv4GliuWafOuPuQDKSm1SJph7uCRnnS61JAn4=",
        version = "v0.0.0-20181026042036-e10d5fee7954",
    )
    go_repository(
        name = "com_github_dustin_go_humanize",
        importpath = "github.com/dustin/go-humanize",
        sum = "h1:VSnTsYCnlFHaM2/igO1h6X3HA71jcobQuxemgkq4zYo=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_dustmop_soup",
        importpath = "github.com/dustmop/soup",
        sum = "h1:44fmjqDtdCiUNlSjJVp+w1AOs6na3Y6Ai0aIeseFjkI=",
        version = "v1.1.2-0.20190516214245-38228baa104e",
    )
    go_repository(
        name = "com_github_elgris_jsondiff",
        importpath = "github.com/elgris/jsondiff",
        sum = "h1:QV0ZrfBLpFc2KDk+a4LJefDczXnonRwrYrQJY/9L4dA=",
        version = "v0.0.0-20160530203242-765b5c24c302",
    )
    go_repository(
        name = "com_github_facebookgo_atomicfile",
        importpath = "github.com/facebookgo/atomicfile",
        sum = "h1:BBso6MBKW8ncyZLv37o+KNyy0HrrHgfnOaGQC2qvN+A=",
        version = "v0.0.0-20151019160806-2de1f203e7d5",
    )
    go_repository(
        name = "com_github_fd_go_nat",
        importpath = "github.com/fd/go-nat",
        sum = "h1:DPyQ97sxA9ThrWYRPcWUz/z9TnpTIGRYODIQc/dy64M=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_go_check_check",
        importpath = "github.com/go-check/check",
        sum = "h1:0gkP6mzaMqkmpcJYCFOLkIBwI7xFExG03bbkOkCvUPI=",
        version = "v0.0.0-20180628173108-788fd7840127",
    )
    go_repository(
        name = "com_github_go_critic_go_critic",
        importpath = "github.com/go-critic/go-critic",
        sum = "h1:djv/qAomOVj8voCHt0M0OYwR/4vfDq1zNKSPKjJCexs=",
        version = "v0.3.5-0.20190526074819-1df300866540",
    )
    go_repository(
        name = "com_github_go_lintpack_lintpack",
        importpath = "github.com/go-lintpack/lintpack",
        sum = "h1:DI5mA3+eKdWeJ40nU4d6Wc26qmdG8RCi/btYq0TuRN0=",
        version = "v0.5.2",
    )
    go_repository(
        name = "com_github_go_toolsmith_astcast",
        importpath = "github.com/go-toolsmith/astcast",
        sum = "h1:JojxlmI6STnFVG9yOImLeGREv8W2ocNUM+iOhR6jE7g=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_go_toolsmith_astcopy",
        importpath = "github.com/go-toolsmith/astcopy",
        sum = "h1:OMgl1b1MEpjFQ1m5ztEO06rz5CUd3oBv9RF7+DyvdG8=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_go_toolsmith_astequal",
        importpath = "github.com/go-toolsmith/astequal",
        sum = "h1:4zxD8j3JRFNyLN46lodQuqz3xdKSrur7U/sr0SDS/gQ=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_go_toolsmith_astfmt",
        importpath = "github.com/go-toolsmith/astfmt",
        sum = "h1:A0vDDXt+vsvLEdbMFJAUBI/uTbRw1ffOPnxsILnFL6k=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_go_toolsmith_astinfo",
        importpath = "github.com/go-toolsmith/astinfo",
        sum = "h1:wP6mXeB2V/d1P1K7bZ5vDUO3YqEzcvOREOxZPEu3gVI=",
        version = "v0.0.0-20180906194353-9809ff7efb21",
    )
    go_repository(
        name = "com_github_go_toolsmith_astp",
        importpath = "github.com/go-toolsmith/astp",
        sum = "h1:alXE75TXgcmupDsMK1fRAy0YUzLzqPVvBKoyWV+KPXg=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_go_toolsmith_pkgload",
        importpath = "github.com/go-toolsmith/pkgload",
        sum = "h1:4DFWWMXVfbcN5So1sBNW9+yeiMqLFGl1wFLTL5R0Tgg=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_go_toolsmith_strparse",
        importpath = "github.com/go-toolsmith/strparse",
        sum = "h1:Vcw78DnpCAKlM20kSbAyO4mPfJn/lyYA4BJUDxe2Jb4=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_go_toolsmith_typep",
        importpath = "github.com/go-toolsmith/typep",
        sum = "h1:zKymWyA1TRYvqYrYDrfEMZULyrhcnGY3x7LDKU2XQaA=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_gobwas_glob",
        importpath = "github.com/gobwas/glob",
        sum = "h1:A4xDbljILXROh+kObIiy5kIaPYD8e96x1tgBhUI5J+Y=",
        version = "v0.2.3",
    )
    go_repository(
        name = "com_github_golangci_check",
        importpath = "github.com/golangci/check",
        sum = "h1:23T5iq8rbUYlhpt5DB4XJkc6BU31uODLD1o1gKvZmD0=",
        version = "v0.0.0-20180506172741-cfe4005ccda2",
    )
    go_repository(
        name = "com_github_golangci_dupl",
        importpath = "github.com/golangci/dupl",
        sum = "h1:w8hkcTqaFpzKqonE9uMCefW1WDie15eSP/4MssdenaM=",
        version = "v0.0.0-20180902072040-3e9179ac440a",
    )
    go_repository(
        name = "com_github_golangci_errcheck",
        importpath = "github.com/golangci/errcheck",
        sum = "h1:YYWNAGTKWhKpcLLt7aSj/odlKrSrelQwlovBpDuf19w=",
        version = "v0.0.0-20181223084120-ef45e06d44b6",
    )
    go_repository(
        name = "com_github_golangci_go_misc",
        importpath = "github.com/golangci/go-misc",
        sum = "h1:9kfjN3AdxcbsZBf8NjltjWihK2QfBBBZuv91cMFfDHw=",
        version = "v0.0.0-20180628070357-927a3d87b613",
    )
    go_repository(
        name = "com_github_golangci_go_tools",
        importpath = "github.com/golangci/go-tools",
        sum = "h1:/7detzz5stiXWPzkTlPTzkBEIIE4WGpppBJYjKqBiPI=",
        version = "v0.0.0-20190318055746-e32c54105b7c",
    )
    go_repository(
        name = "com_github_golangci_goconst",
        importpath = "github.com/golangci/goconst",
        sum = "h1:pe9JHs3cHHDQgOFXJJdYkK6fLz2PWyYtP4hthoCMvs8=",
        version = "v0.0.0-20180610141641-041c5f2b40f3",
    )
    go_repository(
        name = "com_github_golangci_gocyclo",
        importpath = "github.com/golangci/gocyclo",
        sum = "h1:J2XAy40+7yz70uaOiMbNnluTg7gyQhtGqLQncQh+4J8=",
        version = "v0.0.0-20180528134321-2becd97e67ee",
    )
    go_repository(
        name = "com_github_golangci_gofmt",
        importpath = "github.com/golangci/gofmt",
        sum = "h1:0OkFarm1Zy2CjCiDKfK9XHgmc2wbDlRMD2hD8anAJHU=",
        version = "v0.0.0-20181222123516-0b8337e80d98",
    )
    go_repository(
        name = "com_github_golangci_golangci_lint",
        importpath = "github.com/golangci/golangci-lint",
        sum = "h1:XmQgfcLofSG/6AsQuQqmLizB+3GggD+o6ObBG9L+VMM=",
        version = "v1.18.0",
    )
    go_repository(
        name = "com_github_golangci_gosec",
        importpath = "github.com/golangci/gosec",
        sum = "h1:fUdgm/BdKvwOHxg5AhNbkNRp2mSy8sxTXyBVs/laQHo=",
        version = "v0.0.0-20190211064107-66fb7fc33547",
    )
    go_repository(
        name = "com_github_golangci_ineffassign",
        importpath = "github.com/golangci/ineffassign",
        sum = "h1:gLLhTLMk2/SutryVJ6D4VZCU3CUqr8YloG7FPIBWFpI=",
        version = "v0.0.0-20190609212857-42439a7714cc",
    )
    go_repository(
        name = "com_github_golangci_lint_1",
        importpath = "github.com/golangci/lint-1",
        sum = "h1:En/tZdwhAn0JNwLuXzP3k2RVtMqMmOEK7Yu/g3tmtJE=",
        version = "v0.0.0-20190420132249-ee948d087217",
    )
    go_repository(
        name = "com_github_golangci_maligned",
        importpath = "github.com/golangci/maligned",
        sum = "h1:kNY3/svz5T29MYHubXix4aDDuE3RWHkPvopM/EDv/MA=",
        version = "v0.0.0-20180506175553-b1d89398deca",
    )
    go_repository(
        name = "com_github_golangci_misspell",
        importpath = "github.com/golangci/misspell",
        sum = "h1:EL/O5HGrF7Jaq0yNhBLucz9hTuRzj2LdwGBOaENgxIk=",
        version = "v0.0.0-20180809174111-950f5d19e770",
    )
    go_repository(
        name = "com_github_golangci_prealloc",
        importpath = "github.com/golangci/prealloc",
        sum = "h1:leSNB7iYzLYSSx3J/s5sVf4Drkc68W2wm4Ixh/mr0us=",
        version = "v0.0.0-20180630174525-215b22d4de21",
    )
    go_repository(
        name = "com_github_golangci_revgrep",
        importpath = "github.com/golangci/revgrep",
        sum = "h1:HVfrLniijszjS1aiNg8JbBMO2+E1WIQ+j/gL4SQqGPg=",
        version = "v0.0.0-20180526074752-d9c87f5ffaf0",
    )
    go_repository(
        name = "com_github_golangci_unconvert",
        importpath = "github.com/golangci/unconvert",
        sum = "h1:zwtduBRr5SSWhqsYNgcuWO2kFlpdOZbP0+yRjmvPGys=",
        version = "v0.0.0-20180507085042-28b1c447d1f4",
    )
    go_repository(
        name = "com_github_gostaticanalysis_analysisutil",
        importpath = "github.com/gostaticanalysis/analysisutil",
        sum = "h1:JVnpOZS+qxli+rgVl98ILOXVNbW+kb5wcxeGx8ShUIw=",
        version = "v0.0.0-20190318220348-4088753ea4d3",
    )
    go_repository(
        name = "com_github_gxed_go_shellwords",
        importpath = "github.com/gxed/go-shellwords",
        sum = "h1:2TP32H4TAklZUdz84oj95BJhVnIrRasyx2j1cqH5K38=",
        version = "v1.0.3",
    )
    go_repository(
        name = "com_github_gxed_hashland_keccakpg",
        importpath = "github.com/gxed/hashland/keccakpg",
        sum = "h1:wrk3uMNaMxbXiHibbPO4S0ymqJMm41WiudyFSs7UnsU=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_gxed_hashland_murmur3",
        importpath = "github.com/gxed/hashland/murmur3",
        sum = "h1:SheiaIt0sda5K+8FLz952/1iWS9zrnKsEJaOJu4ZbSc=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_gxed_pubsub",
        importpath = "github.com/gxed/pubsub",
        sum = "h1:TF4mX7zXpeyz/xintezebSa7ZDxAGBnqDwcoobvaz2o=",
        version = "v0.0.0-20180201040156-26ebdf44f824",
    )
    go_repository(
        name = "com_github_hsanjuan_go_libp2p_gostream",
        importpath = "github.com/hsanjuan/go-libp2p-gostream",
        sum = "h1:9dIgHQPR0VWxhOyTZrbgLzTx0xvZ5rTpmhG9huGEPjY=",
        version = "v0.0.31",
    )
    go_repository(
        name = "com_github_hsanjuan_go_libp2p_http",
        importpath = "github.com/hsanjuan/go-libp2p-http",
        sum = "h1:hviJbUD3h1Ez2FYTUdnRjrkAzn/9i2V/cLZpFPgnuP8=",
        version = "v0.0.2",
    )
    go_repository(
        name = "com_github_huin_goupnp",
        importpath = "github.com/huin/goupnp",
        sum = "h1:wg75sLpL6DZqwHQN6E1Cfk6mtfzS45z8OV+ic+DtHRo=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_huin_goutil",
        importpath = "github.com/huin/goutil",
        sum = "h1:vlNjIqmUZ9CMAWsbURYl3a6wZbw7q5RHVvlXTNS/Bs8=",
        version = "v0.0.0-20170803182201-1ca381bf3150",
    )
    go_repository(
        name = "com_github_inconshreveable_mousetrap",
        importpath = "github.com/inconshreveable/mousetrap",
        sum = "h1:Z8tu5sraLXCXIcARxBp/8cbvlwVa7Z1NHg9XEKhtSvM=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_ipfs_bbloom",
        importpath = "github.com/ipfs/bbloom",
        sum = "h1:s7KkiBPfxCeDVo47KySjK0ACPc5GJRUxFpdyWEuDjhw=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_dir_index_html",
        importpath = "github.com/ipfs/dir-index-html",
        sum = "h1:cKdOjJBKJgewgcv97zOlqsNOle52N91d3nAkbQadnuY=",
        version = "v1.0.3",
    )
    go_repository(
        name = "com_github_ipfs_go_bitswap",
        importpath = "github.com/ipfs/go-bitswap",
        sum = "h1:BJwx7Kh5W845l10bOckkAJiNrT6XXWNaE8neK7H57q4=",
        version = "v0.0.7",
    )
    go_repository(
        name = "com_github_ipfs_go_block_format",
        importpath = "github.com/ipfs/go-block-format",
        sum = "h1:qPDvcP19izTjU8rgo6p7gTXZlkMkF5bz5G3fqIsSCPE=",
        version = "v0.0.2",
    )
    go_repository(
        name = "com_github_ipfs_go_blockservice",
        importpath = "github.com/ipfs/go-blockservice",
        sum = "h1:40OvwrxeudTAlUGUAKNYnNPcwQeLtXedjzTWecnUinQ=",
        version = "v0.0.3",
    )
    go_repository(
        name = "com_github_ipfs_go_cid",
        importpath = "github.com/ipfs/go-cid",
        sum = "h1:tuuKaZPU1M6HcejsO3AcYWW8sZ8MTvyxfc4uqB4eFE8=",
        version = "v0.0.2",
    )
    go_repository(
        name = "com_github_ipfs_go_cidutil",
        importpath = "github.com/ipfs/go-cidutil",
        sum = "h1:CNOboQf1t7Qp0nuNh8QMmhJs0+Q//bRL1axtCnIB1Yo=",
        version = "v0.0.2",
    )
    go_repository(
        name = "com_github_ipfs_go_datastore",
        importpath = "github.com/ipfs/go-datastore",
        sum = "h1:q3OfiOZV5rlsK1H5V8benjeUApRfMGs4Mrhmr6NriQo=",
        version = "v0.0.5",
    )
    go_repository(
        name = "com_github_ipfs_go_detect_race",
        importpath = "github.com/ipfs/go-detect-race",
        sum = "h1:qX/xay2W3E4Q1U7d9lNs1sU9nvguX0a7319XbyQ6cOk=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ds_badger",
        importpath = "github.com/ipfs/go-ds-badger",
        sum = "h1:sVYE2YlCzltznTZeAP1S+bp3qipz7VzogfZDtf6tGq0=",
        version = "v0.0.3",
    )
    go_repository(
        name = "com_github_ipfs_go_ds_flatfs",
        importpath = "github.com/ipfs/go-ds-flatfs",
        sum = "h1:1zujtU5bPBH6B8roE+TknKIbBCrpau865xUk0dH3x2A=",
        version = "v0.0.2",
    )
    go_repository(
        name = "com_github_ipfs_go_ds_leveldb",
        importpath = "github.com/ipfs/go-ds-leveldb",
        sum = "h1:P5HB59Zblym0B5XYOeEyw3YtPtbpIqQCavCSWaWEEk8=",
        version = "v0.0.2",
    )
    go_repository(
        name = "com_github_ipfs_go_ds_measure",
        importpath = "github.com/ipfs/go-ds-measure",
        sum = "h1:PrCueug+yZLkDCOthZTXKinuoCal/GvlAT7cNxzr03g=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_go_fs_lock",
        importpath = "github.com/ipfs/go-fs-lock",
        sum = "h1:XHX8uW4jQBYWHj59XXcjg7BHlHxV9ZOYs6Y43yb7/l0=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs",
        importpath = "github.com/ipfs/go-ipfs",
        sum = "h1:KB4k3U90cesx60MwHEOqUoSCquZ+JXXHNdw0HIKBusc=",
        version = "v0.4.21",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_addr",
        importpath = "github.com/ipfs/go-ipfs-addr",
        sum = "h1:DpDFybnho9v3/a1dzJ5KnWdThWD1HrFLpQ+tWIyBaFI=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_blockstore",
        importpath = "github.com/ipfs/go-ipfs-blockstore",
        sum = "h1:O9n3PbmTYZoNhkgkEyrXTznbmktIXif62xLX+8dPHzc=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_blocksutil",
        importpath = "github.com/ipfs/go-ipfs-blocksutil",
        sum = "h1:Eh/H4pc1hsvhzsQoMEP3Bke/aW5P5rVM1IWFJMcGIPQ=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_chunker",
        importpath = "github.com/ipfs/go-ipfs-chunker",
        sum = "h1:cHUUxKFQ99pozdahi+uSC/3Y6HeRpi9oTeUHbE27SEw=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_cmds",
        importpath = "github.com/ipfs/go-ipfs-cmds",
        sum = "h1:ZMo0ZeQOr10ZKY4yxYA3lRHUbnF/ZYcV9cpU0IrlGFI=",
        version = "v0.0.8",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_config",
        importpath = "github.com/ipfs/go-ipfs-config",
        sum = "h1:Ep4tRdP1iVK76BgOprD9B/qtOEdpno+1Xb57BqydgGk=",
        version = "v0.0.3",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_delay",
        importpath = "github.com/ipfs/go-ipfs-delay",
        sum = "h1:r/UXYyRcddO6thwOnhiznIAiSvxMECGgtv35Xs1IeRQ=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_ds_help",
        importpath = "github.com/ipfs/go-ipfs-ds-help",
        sum = "h1:QBg+Ts2zgeemK/dB0saiF/ykzRGgfoFMT90Rzo0OnVU=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_exchange_interface",
        importpath = "github.com/ipfs/go-ipfs-exchange-interface",
        sum = "h1:LJXIo9W7CAmugqI+uofioIpRb6rY30GUu7G6LUfpMvM=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_exchange_offline",
        importpath = "github.com/ipfs/go-ipfs-exchange-offline",
        sum = "h1:P56jYKZF7lDDOLx5SotVh5KFxoY6C81I1NSHW1FxGew=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_files",
        importpath = "github.com/ipfs/go-ipfs-files",
        sum = "h1:ME+QnC3uOyla1ciRPezDW0ynQYK2ikOh9OCKAEg4uUA=",
        version = "v0.0.3",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_flags",
        importpath = "github.com/ipfs/go-ipfs-flags",
        sum = "h1:OH5cEkJYL0QgA+bvD55TNG9ud8HA2Nqaav47b2c/UJk=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_posinfo",
        importpath = "github.com/ipfs/go-ipfs-posinfo",
        sum = "h1:Esoxj+1JgSjX0+ylc0hUmJCOv6V2vFoZiETLR6OtpRs=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_pq",
        importpath = "github.com/ipfs/go-ipfs-pq",
        sum = "h1:zgUotX8dcAB/w/HidJh1zzc1yFq6Vm8J7T2F4itj/RU=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_routing",
        importpath = "github.com/ipfs/go-ipfs-routing",
        sum = "h1:394mZeTLcbM/LDO12PneBYvkZAUA+nRnmC0lAzDXKOY=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_util",
        importpath = "github.com/ipfs/go-ipfs-util",
        sum = "h1:Wz9bL2wB2YBJqggkA4dD7oSmqB4cAnpNbGrlHJulv50=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ipld_cbor",
        importpath = "github.com/ipfs/go-ipld-cbor",
        sum = "h1:amzFztBQQQ69UA5+f7JRfoXF/z2l//MGfEDHVkS20+s=",
        version = "v0.0.2",
    )
    go_repository(
        name = "com_github_ipfs_go_ipld_format",
        importpath = "github.com/ipfs/go-ipld-format",
        sum = "h1:OVAGlyYT6JPZ0pEfGntFPS40lfrDmaDbQwNHEY2G9Zs=",
        version = "v0.0.2",
    )
    go_repository(
        name = "com_github_ipfs_go_ipld_git",
        importpath = "github.com/ipfs/go-ipld-git",
        sum = "h1:dn5Quu9lgjkSqkc9CaTsRjzg90kaIitj9wENtigVMH8=",
        version = "v0.0.2",
    )
    go_repository(
        name = "com_github_ipfs_go_ipns",
        importpath = "github.com/ipfs/go-ipns",
        sum = "h1:5vX0+ehF55YWxE8Pmf4eB8szcP+fh24AXnvCkOmSLCc=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_go_log",
        importpath = "github.com/ipfs/go-log",
        sum = "h1:9XTUN/rW64BCG1YhPK9Hoy3q8nr4gOmHHBpgFdfw6Lc=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_go_merkledag",
        importpath = "github.com/ipfs/go-merkledag",
        sum = "h1:A5DlOMzqTRDVmdgkf3dzCKCFmVWH4Zqwb0cbYXUs+Ro=",
        version = "v0.0.3",
    )
    go_repository(
        name = "com_github_ipfs_go_metrics_interface",
        importpath = "github.com/ipfs/go-metrics-interface",
        sum = "h1:j+cpbjYvu4R8zbleSs36gvB7jR+wsL2fGD6n0jO4kdg=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_go_metrics_prometheus",
        importpath = "github.com/ipfs/go-metrics-prometheus",
        sum = "h1:9i2iljLg12S78OhC6UAiXi176xvQGiZaGVF1CUVdE+s=",
        version = "v0.0.2",
    )
    go_repository(
        name = "com_github_ipfs_go_mfs",
        importpath = "github.com/ipfs/go-mfs",
        sum = "h1:Xjqk0jAhgwhMHO39oH4jqP1QkeAGqDelxa814voygN0=",
        version = "v0.0.7",
    )
    go_repository(
        name = "com_github_ipfs_go_path",
        importpath = "github.com/ipfs/go-path",
        sum = "h1:zG/id80tV51XAfvCsRJIEGQSHGuTDBi8RWrtr3EfcfY=",
        version = "v0.0.4",
    )
    go_repository(
        name = "com_github_ipfs_go_peertaskqueue",
        importpath = "github.com/ipfs/go-peertaskqueue",
        sum = "h1:i0JprfjjILYcWM1xguO/1MCS8XKVxLSl+ECEVr6i8nw=",
        version = "v0.0.4",
    )
    go_repository(
        name = "com_github_ipfs_go_todocounter",
        importpath = "github.com/ipfs/go-todocounter",
        sum = "h1:kITWA5ZcQZfrUnDNkRn04Xzh0YFaDFXsoO2A81Eb6Lw=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_go_unixfs",
        importpath = "github.com/ipfs/go-unixfs",
        sum = "h1:mQ6KS3NK4GA9hyUpGdGItqt5llzyIx0Qy2UxC/A7bEo=",
        version = "v0.0.6",
    )
    go_repository(
        name = "com_github_ipfs_go_verifcid",
        importpath = "github.com/ipfs/go-verifcid",
        sum = "h1:m2HI7zIuR5TFyQ1b79Da5N9dnnCP1vcu2QqawmWlK2E=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_hang_fds",
        importpath = "github.com/ipfs/hang-fds",
        sum = "h1:KGAxiGtJPT3THVRNT6yxgpdFPeX4ZemUjENOt6NlOn4=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_interface_go_ipfs_core",
        importpath = "github.com/ipfs/interface-go-ipfs-core",
        sum = "h1:nmEYOfK6QRf3VIdggoZ4rmbKXcC2g6cEdU13Z1CvmL4=",
        version = "v0.0.8",
    )
    go_repository(
        name = "com_github_ipfs_iptb",
        importpath = "github.com/ipfs/iptb",
        sum = "h1:YFYTrCkLMRwk/35IMyC6+yjoQSHTEcNcefBStLJzgvo=",
        version = "v1.4.0",
    )
    go_repository(
        name = "com_github_ipfs_iptb_plugins",
        importpath = "github.com/ipfs/iptb-plugins",
        sum = "h1:JZp4h/+7f00dY4Epr8gzF+VqKITXmVGsZabvmZp7E9I=",
        version = "v0.0.2",
    )
    go_repository(
        name = "com_github_jackpal_gateway",
        importpath = "github.com/jackpal/gateway",
        sum = "h1:qzXWUJfuMdlLMtt0a3Dgt+xkWQiA5itDEITVJtuSwMc=",
        version = "v1.0.5",
    )
    go_repository(
        name = "com_github_jackpal_go_nat_pmp",
        importpath = "github.com/jackpal/go-nat-pmp",
        sum = "h1:i0LektDkO1QlrTm/cSuP+PyBCDnYvjPLGl4LdWEMiaA=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_github_jbenet_go_base58",
        importpath = "github.com/jbenet/go-base58",
        sum = "h1:4zOlv2my+vf98jT1nQt4bT/yKWUImevYPJ2H344CloE=",
        version = "v0.0.0-20150317085156-6237cf65f3a6",
    )
    go_repository(
        name = "com_github_jbenet_go_cienv",
        importpath = "github.com/jbenet/go-cienv",
        sum = "h1:Vc/s0QbQtoxX8MwwSLWWh+xNNZvM3Lw7NsTcHrvvhMc=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_jbenet_go_context",
        importpath = "github.com/jbenet/go-context",
        sum = "h1:BQSFePA1RWJOlocH6Fxy8MmwDt+yVQYULKfN0RoTN8A=",
        version = "v0.0.0-20150711004518-d14ea06fba99",
    )
    go_repository(
        name = "com_github_jbenet_go_is_domain",
        importpath = "github.com/jbenet/go-is-domain",
        sum = "h1:11r5MSptcNFZyBoqubBQnVMUKRWLuRjL1banaIk+iYo=",
        version = "v1.0.2",
    )
    go_repository(
        name = "com_github_jbenet_go_random",
        importpath = "github.com/jbenet/go-random",
        sum = "h1:uUx61FiAa1GI6ZmVd2wf2vULeQZIKG66eybjNXKYCz4=",
        version = "v0.0.0-20190219211222-123a90aedc0c",
    )
    go_repository(
        name = "com_github_jbenet_go_random_files",
        importpath = "github.com/jbenet/go-random-files",
        sum = "h1:fHCa28iw+qaRWZK4IqrntHxXALD5kKr/ESrpOCRRdrg=",
        version = "v0.0.0-20190219210431-31b3f20ebded",
    )
    go_repository(
        name = "com_github_jbenet_go_temp_err_catcher",
        importpath = "github.com/jbenet/go-temp-err-catcher",
        sum = "h1:vhC1OXXiT9R2pczegwz6moDvuRpggaroAXhPIseh57A=",
        version = "v0.0.0-20150120210811-aac704a3f4f2",
    )
    go_repository(
        name = "com_github_jbenet_goprocess",
        importpath = "github.com/jbenet/goprocess",
        sum = "h1:YKyIEECS/XvcfHtBzxtjBBbWK+MbvA6dG8ASiqwvr10=",
        version = "v0.1.3",
    )
    go_repository(
        name = "com_github_jinzhu_copier",
        importpath = "github.com/jinzhu/copier",
        sum = "h1:sHsPfNMAG70QAvKbddQ0uScZCHQoZsT5NykGRCeeeIs=",
        version = "v0.0.0-20180308034124-7e38e58719c3",
    )
    go_repository(
        name = "com_github_jrick_logrotate",
        importpath = "github.com/jrick/logrotate",
        sum = "h1:lQ1bL/n9mBNeIXoTUoYRlK4dHuNJVofX9oWqBtPnSzI=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_k0kubun_colorstring",
        importpath = "github.com/k0kubun/colorstring",
        sum = "h1:uC1QfSlInpQF+M0ao65imhwqKnz3Q2z/d8PWZRMQvDM=",
        version = "v0.0.0-20150214042306-9440f1994b88",
    )
    go_repository(
        name = "com_github_kkdai_bstream",
        importpath = "github.com/kkdai/bstream",
        sum = "h1:n1NeQ3SgUHyISrjFFoO5dR748Is8dBL9qpaTNfphQrs=",
        version = "v0.0.0-20181106074824-b3251f7901ec",
    )
    go_repository(
        name = "com_github_klauspost_compress",
        importpath = "github.com/klauspost/compress",
        sum = "h1:8VMb5+0wMgdBykOV96DwNwKFQ+WTI4pzYURP99CcB9E=",
        version = "v1.4.1",
    )
    go_repository(
        name = "com_github_klauspost_cpuid",
        importpath = "github.com/klauspost/cpuid",
        sum = "h1:NMpwD2G9JSFOE1/TJjGSo5zG7Yb2bTe7eq1jH+irmeE=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_koron_go_ssdp",
        importpath = "github.com/koron/go-ssdp",
        sum = "h1:wxtKgYHEncAU00muMD06dzLiahtGM1eouRNOzVV7tdQ=",
        version = "v0.0.0-20180514024734-4a0ed625a78b",
    )
    go_repository(
        name = "com_github_kubuxu_go_os_helper",
        importpath = "github.com/Kubuxu/go-os-helper",
        sum = "h1:EJiD2VUQyh5A9hWJLmc6iWg6yIcJ7jpBcwC8GMGXfDk=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_kubuxu_gocovmerge",
        importpath = "github.com/Kubuxu/gocovmerge",
        sum = "h1:HNhzThEtZW714v8Eda8sWWRcu9WSzJC+oCyjRjvZgRA=",
        version = "v0.0.0-20161216165753-7ecaa51963cd",
    )
    go_repository(
        name = "com_github_libp2p_go_addr_util",
        importpath = "github.com/libp2p/go-addr-util",
        sum = "h1:TpTQm9cXVRVSKsYbgQ7GKc3KbbHVTnbostgGaDEP+88=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_libp2p_go_buffer_pool",
        importpath = "github.com/libp2p/go-buffer-pool",
        sum = "h1:QNK2iAFa8gjAe1SPz6mHSMuCcjs+X1wlHzeOSqcmlfs=",
        version = "v0.0.2",
    )
    go_repository(
        name = "com_github_libp2p_go_conn_security",
        importpath = "github.com/libp2p/go-conn-security",
        sum = "h1:4kMMrqrt9EUNCNjX1xagSJC+bq16uqjMe9lk1KBMVNs=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_libp2p_go_conn_security_multistream",
        importpath = "github.com/libp2p/go-conn-security-multistream",
        sum = "h1:Ykz0lnNjxk+0SdslUmlLNyrleqdpS1S/VW+dxFdt74Y=",
        version = "v0.0.2",
    )
    go_repository(
        name = "com_github_libp2p_go_flow_metrics",
        importpath = "github.com/libp2p/go-flow-metrics",
        sum = "h1:0gxuFd2GuK7IIP5pKljLwps6TvcuYgvG7Atqi3INF5s=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p",
        importpath = "github.com/libp2p/go-libp2p",
        sum = "h1:tkDnM7iwrz9OSRRb8YAV4HYjv8TKsAxyxrV2sES9/Aw=",
        version = "v0.0.28",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_autonat",
        importpath = "github.com/libp2p/go-libp2p-autonat",
        sum = "h1:OCStANLLpeyQeWFUuqZJ7aS9+Bx0/uoVb1PtLA9fGTQ=",
        version = "v0.0.6",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_autonat_svc",
        importpath = "github.com/libp2p/go-libp2p-autonat-svc",
        sum = "h1:bTom7QFAkJMXiA8ibSsKQ2+LKEHsXZz2IAWYolg/YYg=",
        version = "v0.0.5",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_blankhost",
        importpath = "github.com/libp2p/go-libp2p-blankhost",
        sum = "h1:/mZuuiwntNR8RywnCFlGHLKrKLYne+qciBpQXWqp5fk=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_circuit",
        importpath = "github.com/libp2p/go-libp2p-circuit",
        sum = "h1:vd9vZDy+LDssTvUuxIqnYUOAK2hfHoSQO2xjWhPVEmc=",
        version = "v0.0.8",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_connmgr",
        importpath = "github.com/libp2p/go-libp2p-connmgr",
        sum = "h1:oEUriPO/qWTvfHRPEU4HdNlNhYigdueOs2X3UZCdbYM=",
        version = "v0.0.6",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_crypto",
        importpath = "github.com/libp2p/go-libp2p-crypto",
        sum = "h1:TTdJ4y6Uoa6NxQcuEaVkQfFRcQeCE2ReDk8Ok4I0Fyw=",
        version = "v0.0.2",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_daemon",
        importpath = "github.com/libp2p/go-libp2p-daemon",
        sum = "h1:SpY/vQtIEPMbWv50kXZ9zSV116j0Qz/UkGe8xgOToyw=",
        version = "v0.0.6",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_discovery",
        importpath = "github.com/libp2p/go-libp2p-discovery",
        sum = "h1:/kZwOVmcUHvB94zegSJYnUA9EvT1g8APoQJb5FHyT1c=",
        version = "v0.0.4",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_host",
        importpath = "github.com/libp2p/go-libp2p-host",
        sum = "h1:BB/1Z+4X0rjKP5lbQTmjEjLbDVbrcmLOlA6QDsN5/j4=",
        version = "v0.0.3",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_interface_connmgr",
        importpath = "github.com/libp2p/go-libp2p-interface-connmgr",
        sum = "h1:KG/KNYL2tYzXAfMvQN5K1aAGTYSYUMJ1prgYa2/JI1E=",
        version = "v0.0.5",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_interface_pnet",
        importpath = "github.com/libp2p/go-libp2p-interface-pnet",
        sum = "h1:7GnzRrBTJHEsofi1ahFdPN9Si6skwXQE9UqR2S+Pkh8=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_kad_dht",
        importpath = "github.com/libp2p/go-libp2p-kad-dht",
        sum = "h1:ReMb41jJrngvXnU5Tirf74bBkXx4M9ne5QyFQPeNYtw=",
        version = "v0.0.13",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_kbucket",
        importpath = "github.com/libp2p/go-libp2p-kbucket",
        sum = "h1:ZrvW3qCM+lAuv7nrNts/zfEiClq+GZe8OIzX4Vb3Dwo=",
        version = "v0.1.1",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_loggables",
        importpath = "github.com/libp2p/go-libp2p-loggables",
        sum = "h1:HVww9oAnINIxbt69LJNkxD8lnbfgteXR97Xm4p3l9ps=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_metrics",
        importpath = "github.com/libp2p/go-libp2p-metrics",
        sum = "h1:yumdPC/P2VzINdmcKZd0pciSUCpou+s0lwYCjBbzQZU=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_mplex",
        importpath = "github.com/libp2p/go-libp2p-mplex",
        sum = "h1:lSPS1VJ36P01gGO//KgcsmSah5uoC3X9r7WY5j+iP4c=",
        version = "v0.1.1",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_nat",
        importpath = "github.com/libp2p/go-libp2p-nat",
        sum = "h1:+KXK324yaY701On8a0aGjTnw8467kW3ExKcqW2wwmyw=",
        version = "v0.0.4",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_net",
        importpath = "github.com/libp2p/go-libp2p-net",
        sum = "h1:qP06u4TYXfl7uW/hzqPhlVVTSA2nw1B/bHBJaUnbh6M=",
        version = "v0.0.2",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_netutil",
        importpath = "github.com/libp2p/go-libp2p-netutil",
        sum = "h1:LgD6+skofkOx8z6odD9+MZHKjupv3ng1u6KRhaADTnA=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_peer",
        importpath = "github.com/libp2p/go-libp2p-peer",
        sum = "h1:qGCWD1a+PyZcna6htMPo26jAtqirVnJ5NvBQIKV7rRY=",
        version = "v0.1.1",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_peerstore",
        importpath = "github.com/libp2p/go-libp2p-peerstore",
        sum = "h1:RgX/djPFXqZGktW0j2eF4NAX0pzDsCot45jO2GewC+g=",
        version = "v0.0.6",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_pnet",
        importpath = "github.com/libp2p/go-libp2p-pnet",
        sum = "h1:2e5d15M8XplUKsU4Fqrll5eDfqGg/7mHUufLkhbfKHM=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_protocol",
        importpath = "github.com/libp2p/go-libp2p-protocol",
        sum = "h1:+zkEmZ2yFDi5adpVE3t9dqh/N9TbpFWywowzeEzBbLM=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_pubsub",
        importpath = "github.com/libp2p/go-libp2p-pubsub",
        sum = "h1:DKVoDac2u1Dr8gX7B1HFjCrXkMxr8tfbnt0Fk1mmkgk=",
        version = "v0.0.3",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_pubsub_router",
        importpath = "github.com/libp2p/go-libp2p-pubsub-router",
        sum = "h1:2EF+8nueIsA9Unpj1MxdlS9+dX29kwCxSttchMMfXsI=",
        version = "v0.0.3",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_quic_transport",
        importpath = "github.com/libp2p/go-libp2p-quic-transport",
        sum = "h1:FGEPXsjpY9K6P3iMtJQPKGl45eXickBY1+xSJ84lVVI=",
        version = "v0.0.3",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_record",
        importpath = "github.com/libp2p/go-libp2p-record",
        sum = "h1:zN7AS3X46qmwsw5JLxdDuI43cH5UYwovKxHPjKBYQxw=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_routing",
        importpath = "github.com/libp2p/go-libp2p-routing",
        sum = "h1:hPMAWktf9rYi3ME4MG48qE7dq1ofJxiQbfdvpNntjhc=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_routing_helpers",
        importpath = "github.com/libp2p/go-libp2p-routing-helpers",
        sum = "h1:SLX7eDQE8Xo197NwNM/hM7WnH3w6fSGY9+G9HkiYwqQ=",
        version = "v0.0.2",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_secio",
        importpath = "github.com/libp2p/go-libp2p-secio",
        sum = "h1:h3fPeDrej7bvvARnC2oSjAfcLZOaS4REZKgWCRQNpE4=",
        version = "v0.0.3",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_swarm",
        importpath = "github.com/libp2p/go-libp2p-swarm",
        sum = "h1:gE0P/v2h+KEXtAi9YTw2UBOSODJ4m9VuuJ+ktc2LVUo=",
        version = "v0.0.6",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_tls",
        importpath = "github.com/libp2p/go-libp2p-tls",
        sum = "h1:UIslpmpKDbjEymuidtP2D9up00GfWrOs6eyTKf83uBA=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_transport",
        importpath = "github.com/libp2p/go-libp2p-transport",
        sum = "h1:pV6+UlRxyDpASSGD+60vMvdifSCby6JkJDfi+yUMHac=",
        version = "v0.0.5",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_transport_upgrader",
        importpath = "github.com/libp2p/go-libp2p-transport-upgrader",
        sum = "h1:uGMOd14BL1oFlfb/cGfOxPjiTKBhzWV4aMjjoCF1Z1o=",
        version = "v0.0.4",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_yamux",
        importpath = "github.com/libp2p/go-libp2p-yamux",
        sum = "h1:HmKvv2jWJ4GEm3iP7cEKjuw0POa6rK+Hcsu1FBKzpLc=",
        version = "v0.1.3",
    )
    go_repository(
        name = "com_github_libp2p_go_maddr_filter",
        importpath = "github.com/libp2p/go-maddr-filter",
        sum = "h1:hx8HIuuwk34KePddrp2mM5ivgPkZ09JH4AvsALRbFUs=",
        version = "v0.0.4",
    )
    go_repository(
        name = "com_github_libp2p_go_mplex",
        importpath = "github.com/libp2p/go-mplex",
        sum = "h1:043XJ3Zr7/Oz5cfyUaJwxUZyP02TngTpt4oq8R5UizQ=",
        version = "v0.0.4",
    )
    go_repository(
        name = "com_github_libp2p_go_msgio",
        importpath = "github.com/libp2p/go-msgio",
        sum = "h1:ivPvEKHxmVkTClHzg6RXTYHqaJQ0V9cDbq+6lKb3UV0=",
        version = "v0.0.2",
    )
    go_repository(
        name = "com_github_libp2p_go_nat",
        importpath = "github.com/libp2p/go-nat",
        sum = "h1:l6fKV+p0Xa354EqQOQP+d8CivdLM4kl5GxC1hSc/UeI=",
        version = "v0.0.3",
    )
    go_repository(
        name = "com_github_libp2p_go_reuseport",
        importpath = "github.com/libp2p/go-reuseport",
        sum = "h1:7PhkfH73VXfPJYKQ6JwS5I/eVcoyYi9IMNGc6FWpFLw=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_libp2p_go_reuseport_transport",
        importpath = "github.com/libp2p/go-reuseport-transport",
        sum = "h1:WglMwyXyBu61CMkjCCtnmqNqnjib0GIEjMiHTwR/KN4=",
        version = "v0.0.2",
    )
    go_repository(
        name = "com_github_libp2p_go_stream_muxer",
        importpath = "github.com/libp2p/go-stream-muxer",
        sum = "h1:Ce6e2Pyu+b5MC1k3eeFtAax0pW4gc6MosYSLV05UeLw=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_libp2p_go_stream_muxer_multistream",
        importpath = "github.com/libp2p/go-stream-muxer-multistream",
        sum = "h1:DhHqb4nu1fQv/vQKeLAaZGmhLsUA4SF77IdYJiWE1d4=",
        version = "v0.1.1",
    )
    go_repository(
        name = "com_github_libp2p_go_tcp_transport",
        importpath = "github.com/libp2p/go-tcp-transport",
        sum = "h1:2iRu994wCT/iEz62F+c60FUoSkijNEQ0q2Itc+79XlQ=",
        version = "v0.0.4",
    )
    go_repository(
        name = "com_github_libp2p_go_testutil",
        importpath = "github.com/libp2p/go-testutil",
        sum = "h1:Xg+O0G2HIMfHqBOBDcMS1iSZJ3GEcId4qOxCQvsGZHk=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_libp2p_go_ws_transport",
        importpath = "github.com/libp2p/go-ws-transport",
        sum = "h1:3wt9ed0gIUrne627XHvPMTwG4/AUpsLDy4TGQi2EyQ0=",
        version = "v0.0.4",
    )
    go_repository(
        name = "com_github_libp2p_go_yamux",
        importpath = "github.com/libp2p/go-yamux",
        sum = "h1:xX8A36vpXb59frIzWFdEgptLMsOANMFq2K7fPRlunYI=",
        version = "v1.2.3",
    )
    go_repository(
        name = "com_github_logrusorgru_aurora",
        importpath = "github.com/logrusorgru/aurora",
        sum = "h1:9MlwzLdW7QSDrhDjFlsEYmxpFyIoXmYRon3dt0io31k=",
        version = "v0.0.0-20181002194514-a7b3b318ed4e",
    )
    go_repository(
        name = "com_github_lucas_clemente_aes12",
        importpath = "github.com/lucas-clemente/aes12",
        sum = "h1:sSeNEkJrs+0F9TUau0CgWTTNEwF23HST3Eq0A+QIx+A=",
        version = "v0.0.0-20171027163421-cd47fb39b79f",
    )
    go_repository(
        name = "com_github_lucas_clemente_quic_go",
        importpath = "github.com/lucas-clemente/quic-go",
        sum = "h1:zasajC848Dqq/+WqfqBCkmPw+YHNe1MBts/z7y7nXf4=",
        version = "v0.11.1",
    )
    go_repository(
        name = "com_github_lucas_clemente_quic_go_certificates",
        importpath = "github.com/lucas-clemente/quic-go-certificates",
        sum = "h1:zqEC1GJZFbGZA0tRyNZqRjep92K5fujFtFsu5ZW7Aug=",
        version = "v0.0.0-20160823095156-d2f86524cced",
    )
    go_repository(
        name = "com_github_magiconair_properties",
        importpath = "github.com/magiconair/properties",
        sum = "h1:U+1DqNen04MdEPgFiIwdOUiqZ8qPa37xgogX/sd3+54=",
        version = "v1.7.6",
    )
    go_repository(
        name = "com_github_marten_seemann_qtls",
        importpath = "github.com/marten-seemann/qtls",
        sum = "h1:0yWJ43C62LsZt08vuQJDK1uC1czUc3FJeCLPoNAI4vA=",
        version = "v0.2.3",
    )
    go_repository(
        name = "com_github_mattn_goveralls",
        importpath = "github.com/mattn/goveralls",
        sum = "h1:7eJB6EqsPhRVxvwEXGnqdO2sJI0PTsrWoTMXEk9/OQc=",
        version = "v0.0.2",
    )
    go_repository(
        name = "com_github_mgutz_ansi",
        importpath = "github.com/mgutz/ansi",
        sum = "h1:j7+1HpAFS1zy5+Q4qx1fWh90gTKwiN4QCGoY9TWyyO4=",
        version = "v0.0.0-20170206155736-9520e82c474b",
    )
    go_repository(
        name = "com_github_minio_blake2b_simd",
        importpath = "github.com/minio/blake2b-simd",
        sum = "h1:lYpkrQH5ajf0OXOcUbGjvZxxijuBwbbmlSxLiuofa+g=",
        version = "v0.0.0-20160723061019-3f5f724cb5b1",
    )
    go_repository(
        name = "com_github_minio_sha256_simd",
        importpath = "github.com/minio/sha256-simd",
        sum = "h1:l16XLUUJ34wIz+RIvLhSwGvLvKyy+W598b135bJN6mg=",
        version = "v0.0.0-20190328051042-05b4dd3047e5",
    )
    go_repository(
        name = "com_github_mitchellh_go_ps",
        importpath = "github.com/mitchellh/go-ps",
        sum = "h1:kw1v0NlnN+GZcU8Ma8CLF2Zzgjfx95gs3/GN3vYAPpo=",
        version = "v0.0.0-20170309133038-4fdf99ab2936",
    )
    go_repository(
        name = "com_github_mohae_deepcopy",
        importpath = "github.com/mohae/deepcopy",
        sum = "h1:RWengNIwukTxcDr9M+97sNutRR1RKhG96O6jWumTTnw=",
        version = "v0.0.0-20170929034955-c48cc78d4826",
    )
    go_repository(
        name = "com_github_mozilla_tls_observatory",
        importpath = "github.com/mozilla/tls-observatory",
        sum = "h1:Q0XH6Ql1+Z6YbUKyWyI0sD8/9yH0U8x86yA8LuWMJwY=",
        version = "v0.0.0-20180409132520-8791a200eb40",
    )
    go_repository(
        name = "com_github_mr_tron_base58",
        importpath = "github.com/mr-tron/base58",
        sum = "h1:ZEw4I2EgPKDJ2iEw0cNmLB3ROrEmkOtXIkaG7wZg+78=",
        version = "v1.1.2",
    )
    go_repository(
        name = "com_github_multiformats_go_base32",
        importpath = "github.com/multiformats/go-base32",
        sum = "h1:tw5+NhuwaOjJCC5Pp82QuXbrmLzWg7uxlMFp8Nq/kkI=",
        version = "v0.0.3",
    )
    go_repository(
        name = "com_github_multiformats_go_multiaddr",
        importpath = "github.com/multiformats/go-multiaddr",
        sum = "h1:WgMSI84/eRLdbptXMkMWDXPjPq7SPLIgGUVm2eroyU4=",
        version = "v0.0.4",
    )
    go_repository(
        name = "com_github_multiformats_go_multiaddr_dns",
        importpath = "github.com/multiformats/go-multiaddr-dns",
        sum = "h1:/Bbsgsy3R6e3jf2qBahzNHzww6usYaZ0NhNH3sqdFS8=",
        version = "v0.0.2",
    )
    go_repository(
        name = "com_github_multiformats_go_multiaddr_net",
        importpath = "github.com/multiformats/go-multiaddr-net",
        sum = "h1:76O59E3FavvHqNg7jvzWzsPSW5JSi/ek0E4eiDVbg9g=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_multiformats_go_multibase",
        importpath = "github.com/multiformats/go-multibase",
        sum = "h1:PN9/v21eLywrFWdFNsFKaU04kLJzuYzmrJR+ubhT9qA=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_multiformats_go_multicodec",
        importpath = "github.com/multiformats/go-multicodec",
        sum = "h1:4u6lcjbE4VVVoigU4QJSSVYsGVP4j2jtDkR8lPwOrLE=",
        version = "v0.1.6",
    )
    go_repository(
        name = "com_github_multiformats_go_multihash",
        importpath = "github.com/multiformats/go-multihash",
        sum = "h1:1wxmCvTXAifAepIMyF39vZinRw5sbqjPs/UIi93+uik=",
        version = "v0.0.5",
    )
    go_repository(
        name = "com_github_multiformats_go_multistream",
        importpath = "github.com/multiformats/go-multistream",
        sum = "h1:rNgWgFyzRSTI9L+xISrz7kN5MdNXoEcoIeeCH05wLKA=",
        version = "v0.0.4",
    )
    go_repository(
        name = "com_github_nbutton23_zxcvbn_go",
        importpath = "github.com/nbutton23/zxcvbn-go",
        sum = "h1:Ri1EhipkbhWsffPJ3IPlrb4SkTOPa2PfRXp3jchBczw=",
        version = "v0.0.0-20171102151520-eafdab6b0663",
    )
    go_repository(
        name = "com_github_oklog_ulid",
        importpath = "github.com/oklog/ulid",
        sum = "h1:EGfNDEx6MqHz8B3uNV6QAib1UR2Lm97sHi3ocA6ESJ4=",
        version = "v1.3.1",
    )
    go_repository(
        name = "com_github_oneofone_xxhash",
        importpath = "github.com/OneOfOne/xxhash",
        sum = "h1:KMrpdQIwFcEqXDklaen+P1axHaj9BSKzvpUUfnHldSE=",
        version = "v1.2.2",
    )
    go_repository(
        name = "com_github_openpeedeep_depguard",
        importpath = "github.com/OpenPeeDeeP/depguard",
        sum = "h1:k9QF73nrHT3nPLz3lu6G5s+3Hi8Je36ODr1F5gjAXXM=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_opentracing_opentracing_go",
        importpath = "github.com/opentracing/opentracing-go",
        sum = "h1:pWlfV3Bxv7k65HYwkikxat0+s3pV4bsqf19k25Ur8rU=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_paulmach_orb",
        importpath = "github.com/paulmach/orb",
        sum = "h1:Wa1nzU269Zv7V9paVEY1COWW8FCqv4PC/KJRbJSimpM=",
        version = "v0.1.3",
    )
    go_repository(
        name = "com_github_pborman_uuid",
        importpath = "github.com/pborman/uuid",
        sum = "h1:J7Q5mO4ysT1dv8hyrUGHb9+ooztCXu1D8MY8DZYsu3g=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_pelletier_go_toml",
        importpath = "github.com/pelletier/go-toml",
        sum = "h1:cmiOvKzEunMsAxyhXSzpL5Q1CRKpVv0KQsnAIcSEVYM=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_polydawn_refmt",
        importpath = "github.com/polydawn/refmt",
        sum = "h1:bzMe+2coZJYHnhGgVlcQKuRy4FSny4ds8dLQjw5P1XE=",
        version = "v0.0.0-20190221155625-df39d6c2d992",
    )
    go_repository(
        name = "com_github_prometheus_tsdb",
        importpath = "github.com/prometheus/tsdb",
        sum = "h1:YZcsG11NqnK4czYLrWd9mpEuAJIHVQLwdrleYfszMAA=",
        version = "v0.7.1",
    )
    go_repository(
        name = "com_github_puerkitobio_goquery",
        importpath = "github.com/PuerkitoBio/goquery",
        sum = "h1:uGvmFXOA73IKluu/F84Xd1tt/z07GYm8X49XKHP7EJk=",
        version = "v1.5.0",
    )
    go_repository(
        name = "com_github_qri_io_compare",
        importpath = "github.com/qri-io/compare",
        sum = "h1:A/MRx3uEnJ/iMjfJY1VOqH9CYs9zFSEYaFVeXuGfmis=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_qri_io_dataset",
        importpath = "github.com/qri-io/dataset",
        sum = "h1:BiOaxMD2LLY/c6Fn/6HdITmkX9xEBBG9Jz4X6ud8CxY=",
        version = "v0.1.1",
    )
    go_repository(
        name = "com_github_qri_io_dsdiff",
        importpath = "github.com/qri-io/dsdiff",
        sum = "h1:w1XG/l9AvXEiAMRlccxRttPbT/5PdIfrA402XuzOo4U=",
        version = "v0.1.1",
    )
    go_repository(
        name = "com_github_qri_io_jsonpointer",
        importpath = "github.com/qri-io/jsonpointer",
        sum = "h1:OcTtTmorodUCRc2CZhj/ZwOET8zVj6uo0ArEmzoThZI=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_qri_io_jsonschema",
        importpath = "github.com/qri-io/jsonschema",
        sum = "h1:t//Doa/gvMqJ0bDhG7PGIKfaWGGxRVaffp+bcvBGGEk=",
        version = "v0.1.1",
    )
    go_repository(
        name = "com_github_qri_io_qfs",
        importpath = "github.com/qri-io/qfs",
        sum = "h1:XgDhon13qLlwJHI2m0FAc2EhZEcrqMPwizvgfQXHzB8=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_qri_io_varname",
        importpath = "github.com/qri-io/varName",
        sum = "h1:dFP5qZHrxnn5fNoMbjfpMCRBYDrOsoyls7R07r+emk0=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_quasilyte_go_consistent",
        importpath = "github.com/Quasilyte/go-consistent",
        sum = "h1:1EBYo2g4jmrQYzTN/GXwdkZGDJpMDdygqXq99yllHPM=",
        version = "v0.0.0-20181230194409-8f8379e70f99",
    )
    go_repository(
        name = "com_github_quasilyte_go_consistent",
        importpath = "github.com/quasilyte/go-consistent",
        sum = "h1:JoUA0uz9U0FVFq5p4LjEq4C0VgQ0El320s3Ms0V4eww=",
        version = "v0.0.0-20190521200055-c6f3937de18c",
    )
    go_repository(
        name = "com_github_rogpeppe_go_internal",
        importpath = "github.com/rogpeppe/go-internal",
        sum = "h1:g0fH8RicVgNl+zVZDCDfbdWxAWoAEJyI7I3TZYXFiig=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_rs_cors",
        importpath = "github.com/rs/cors",
        sum = "h1:G9tHG9lebljV9mfp9SNPDL36nCDxmo3zTlAf1YgvzmI=",
        version = "v1.6.0",
    )
    go_repository(
        name = "com_github_shurcool_go",
        importpath = "github.com/shurcooL/go",
        sum = "h1:MZM7FHLqUHYI0Y/mQAt3d2aYa0SiNms/hFqC9qJYolM=",
        version = "v0.0.0-20180423040247-9e1955d9fb6e",
    )
    go_repository(
        name = "com_github_shurcool_go_goon",
        importpath = "github.com/shurcooL/go-goon",
        sum = "h1:llrF3Fs4018ePo4+G/HV/uQUqEI1HMDjCeOf2V6puPc=",
        version = "v0.0.0-20170922171312-37c2f522c041",
    )
    go_repository(
        name = "com_github_sourcegraph_go_diff",
        importpath = "github.com/sourcegraph/go-diff",
        sum = "h1:gO6i5zugwzo1RVTvgvfwCOSVegNuvnNi6bAD1QCmkHs=",
        version = "v0.5.1",
    )
    go_repository(
        name = "com_github_spacemonkeygo_openssl",
        importpath = "github.com/spacemonkeygo/openssl",
        sum = "h1:/eS3yfGjQKG+9kayBkj0ip1BGhq6zJ3eaVksphxAaek=",
        version = "v0.0.0-20181017203307-c2dcc5cca94a",
    )
    go_repository(
        name = "com_github_spacemonkeygo_spacelog",
        importpath = "github.com/spacemonkeygo/spacelog",
        sum = "h1:RC6RW7j+1+HkWaX/Yh71Ee5ZHaHYt7ZP4sQgUrm6cDU=",
        version = "v0.0.0-20180420211403-2296661a0572",
    )
    go_repository(
        name = "com_github_spaolacci_murmur3",
        importpath = "github.com/spaolacci/murmur3",
        sum = "h1:7c1g84S4BPRrfL5Xrdp6fOJ206sU9y293DDHaoy0bLI=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_spf13_cast",
        importpath = "github.com/spf13/cast",
        sum = "h1:HHl1DSRbEQN2i8tJmtS6ViPyHx35+p51amrdsiTCrkg=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_spf13_cobra",
        importpath = "github.com/spf13/cobra",
        sum = "h1:NfkwRbgViGoyjBKsLI0QMDcuMnhM+SBg3T0cGfpvKDE=",
        version = "v0.0.2",
    )
    go_repository(
        name = "com_github_spf13_jwalterweatherman",
        importpath = "github.com/spf13/jwalterweatherman",
        sum = "h1:2ZXvIUGghLpdTVHR1UfvfrzoVlZaE/yOWC5LueIHZig=",
        version = "v0.0.0-20180109140146-7c0cea34c8ec",
    )
    go_repository(
        name = "com_github_spf13_viper",
        importpath = "github.com/spf13/viper",
        sum = "h1:Ncr3ZIuJn322w2k1qmzXDnkLAdQMlJqBa9kfAH+irso=",
        version = "v1.0.2",
    )
    go_repository(
        name = "com_github_stebalien_go_bitfield",
        importpath = "github.com/Stebalien/go-bitfield",
        sum = "h1:X3kbSSPUaJK60wV2hjOPZwmpljr6VGCqdq4cBLhbQBo=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_syndtr_goleveldb",
        importpath = "github.com/syndtr/goleveldb",
        sum = "h1:fBdIW9lB4Iz0n9khmH8w27SJ3QEJ7+IgjPEwGSZiFdE=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_texttheater_golang_levenshtein",
        importpath = "github.com/texttheater/golang-levenshtein",
        sum = "h1:T5PdfK/M1xyrHwynxMIVMWLS7f/qHwfslZphxtGnw7s=",
        version = "v0.0.0-20180516184445-d188e65d659e",
    )
    go_repository(
        name = "com_github_timakin_bodyclose",
        importpath = "github.com/timakin/bodyclose",
        sum = "h1:AmoEvWAO3nDx1MEcMzPh+GzOOIA5Znpv6++c7bePPY0=",
        version = "v0.0.0-20190721030226-87058b9bfcec",
    )
    go_repository(
        name = "com_github_ultraware_funlen",
        importpath = "github.com/ultraware/funlen",
        sum = "h1:UeC9tpM4wNWzUJfan8z9sFE4QCzjjzlCZmuJN+aOkH0=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_urfave_cli",
        importpath = "github.com/urfave/cli",
        sum = "h1:fDqGv3UG/4jbVl/QkFwEdddtEDjh/5Ov6X+0B/3bPaw=",
        version = "v1.20.0",
    )
    go_repository(
        name = "com_github_valyala_bytebufferpool",
        importpath = "github.com/valyala/bytebufferpool",
        sum = "h1:GqA5TC/0021Y/b9FG4Oi9Mr3q7XYx6KllzawFIhcdPw=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_valyala_fasthttp",
        importpath = "github.com/valyala/fasthttp",
        sum = "h1:dzZJf2IuMiclVjdw0kkT+f9u4YdrapbNyGAN47E/qnk=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_valyala_quicktemplate",
        importpath = "github.com/valyala/quicktemplate",
        sum = "h1:C58y/wN0FMTi2PR0n3onltemfFabany53j7M6SDDB8k=",
        version = "v1.1.1",
    )
    go_repository(
        name = "com_github_valyala_tcplisten",
        importpath = "github.com/valyala/tcplisten",
        sum = "h1:0R4NLDRDZX6JcmhJgXi5E4b8Wg84ihbmUKp/GvSPEzc=",
        version = "v0.0.0-20161114210144-ceec8f93295a",
    )
    go_repository(
        name = "com_github_warpfork_go_wish",
        importpath = "github.com/warpfork/go-wish",
        sum = "h1:qOpVTI+BrstcjTZLm2Yz/3sOnqkzj3FQoh0g+E5s3Gc=",
        version = "v0.0.0-20180510122957-5ad1f5abf436",
    )
    go_repository(
        name = "com_github_whyrusleeping_base32",
        importpath = "github.com/whyrusleeping/base32",
        sum = "h1:BCPnHtcboadS0DvysUuJXZ4lWVv5Bh5i7+tbIyi+ck4=",
        version = "v0.0.0-20170828182744-c30ac30633cc",
    )
    go_repository(
        name = "com_github_whyrusleeping_cbor",
        importpath = "github.com/whyrusleeping/cbor",
        sum = "h1:5HZfQkwe0mIfyDmc1Em5GqlNRzcdtlv4HTNmdpt7XH0=",
        version = "v0.0.0-20171005072247-63513f603b11",
    )
    go_repository(
        name = "com_github_whyrusleeping_chunker",
        importpath = "github.com/whyrusleeping/chunker",
        sum = "h1:jQa4QT2UP9WYv2nzyawpKMOCl+Z/jW7djv2/J50lj9E=",
        version = "v0.0.0-20181014151217-fe64bd25879f",
    )
    go_repository(
        name = "com_github_whyrusleeping_go_ctrlnet",
        importpath = "github.com/whyrusleeping/go-ctrlnet",
        sum = "h1:c23eYhe7i8MG6dUSPzyIDDy5+cWOoZMovPamBKqrjYQ=",
        version = "v0.0.0-20180313164037-f564fbbdaa95",
    )
    go_repository(
        name = "com_github_whyrusleeping_go_keyspace",
        importpath = "github.com/whyrusleeping/go-keyspace",
        sum = "h1:EKhdznlJHPMoKr0XTrX+IlJs1LH3lyx2nfr1dOlZ79k=",
        version = "v0.0.0-20160322163242-5b898ac5add1",
    )
    go_repository(
        name = "com_github_whyrusleeping_go_logging",
        importpath = "github.com/whyrusleeping/go-logging",
        sum = "h1:9lDbC6Rz4bwmou+oE6Dt4Cb2BGMur5eR/GYptkKUVHo=",
        version = "v0.0.0-20170515211332-0457bb6b88fc",
    )
    go_repository(
        name = "com_github_whyrusleeping_go_notifier",
        importpath = "github.com/whyrusleeping/go-notifier",
        sum = "h1:M/lL30eFZTKnomXY6huvM6G0+gVquFNf6mxghaWlFUg=",
        version = "v0.0.0-20170827234753-097c5d47330f",
    )
    go_repository(
        name = "com_github_whyrusleeping_go_smux_multiplex",
        importpath = "github.com/whyrusleeping/go-smux-multiplex",
        sum = "h1:iqksILj8STw03EJQe7Laj4ubnw+ojOyik18cd5vPL1o=",
        version = "v3.0.16+incompatible",
    )
    go_repository(
        name = "com_github_whyrusleeping_go_smux_multistream",
        importpath = "github.com/whyrusleeping/go-smux-multistream",
        sum = "h1:BdYHctE9HJZLquG9tpTdwWcbG4FaX6tVKPGjCGgiVxo=",
        version = "v2.0.2+incompatible",
    )
    go_repository(
        name = "com_github_whyrusleeping_go_smux_yamux",
        importpath = "github.com/whyrusleeping/go-smux-yamux",
        sum = "h1:nVkExQ7pYlN9e45LcqTCOiDD0904fjtm0flnHZGbXkw=",
        version = "v2.0.9+incompatible",
    )
    go_repository(
        name = "com_github_whyrusleeping_go_sysinfo",
        importpath = "github.com/whyrusleeping/go-sysinfo",
        sum = "h1:ctS9Anw/KozviCCtK6VWMz5kPL9nbQzbQY4yfqlIV4M=",
        version = "v0.0.0-20190219211824-4a357d4b90b1",
    )
    go_repository(
        name = "com_github_whyrusleeping_mafmt",
        importpath = "github.com/whyrusleeping/mafmt",
        sum = "h1:TCghSl5kkwEE0j+sU/gudyhVMRlpBin8fMBBHg59EbA=",
        version = "v1.2.8",
    )
    go_repository(
        name = "com_github_whyrusleeping_mdns",
        importpath = "github.com/whyrusleeping/mdns",
        sum = "h1:nMCC9Pwz1pxfC1Y6mYncdk+kq8d5aLx0Q+/gyZGE44M=",
        version = "v0.0.0-20180901202407-ef14215e6b30",
    )
    go_repository(
        name = "com_github_whyrusleeping_multiaddr_filter",
        importpath = "github.com/whyrusleeping/multiaddr-filter",
        sum = "h1:E9S12nwJwEOXe2d6gT6qxdvqMnNq+VnSsKPgm2ZZNds=",
        version = "v0.0.0-20160516205228-e903e4adabd7",
    )
    go_repository(
        name = "com_github_whyrusleeping_tar_utils",
        importpath = "github.com/whyrusleeping/tar-utils",
        sum = "h1:GGsyl0dZ2jJgVT+VvWBf/cNijrHRhkrTjkmp5wg7li0=",
        version = "v0.0.0-20180509141711-8c6c8ba81d5c",
    )
    go_repository(
        name = "com_github_whyrusleeping_timecache",
        importpath = "github.com/whyrusleeping/timecache",
        sum = "h1:lYbXeSvJi5zk5GLKVuid9TVjS9a0OmLIDKTfoZBL6Ow=",
        version = "v0.0.0-20160911033111-cfcb2f1abfee",
    )
    go_repository(
        name = "com_github_whyrusleeping_yamux",
        importpath = "github.com/whyrusleeping/yamux",
        sum = "h1:PzUrk7/Z0g/N5V4/+DesmKXYcCToALgj+SbATgs0B34=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_yudai_gojsondiff",
        importpath = "github.com/yudai/gojsondiff",
        sum = "h1:27cbfqXLVEJ1o8I6v3y9lg8Ydm53EKqHXAOMxEGlCOA=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_yudai_golcs",
        importpath = "github.com/yudai/golcs",
        sum = "h1:BHyfKlQyqbsFN5p3IfnEUduWvb9is428/nNb5L3U01M=",
        version = "v0.0.0-20170316035057-ecda9a501e82",
    )
    go_repository(
        name = "com_github_yudai_pp",
        importpath = "github.com/yudai/pp",
        sum = "h1:Q4//iY4pNF6yPLZIigmvcl7k/bPgrcTPIFIcmawg5bI=",
        version = "v2.0.1+incompatible",
    )
    go_repository(
        name = "com_sourcegraph_sqs_pbtypes",
        importpath = "sourcegraph.com/sqs/pbtypes",
        sum = "h1:JPJh2pk3+X4lXAkZIk2RuE/7/FoK9maXw+TNPJhVS/c=",
        version = "v0.0.0-20180604144634-d3ebe8f20ae4",
    )
    go_repository(
        name = "in_gopkg_errgo_v2",
        importpath = "gopkg.in/errgo.v2",
        sum = "h1:0vLT13EuvQ0hNvakwLuFZ/jYrLp5F3kcWHXdRggjCE8=",
        version = "v2.1.0",
    )
    go_repository(
        name = "org_bazil_fuse",
        importpath = "bazil.org/fuse",
        sum = "h1:FNCRpXiquG1aoyqcIWVFmpTSKVcx2bQD38uZZeGtdlw=",
        version = "v0.0.0-20180421153158-65cc252bf669",
    )
    go_repository(
        name = "org_go4",
        importpath = "go4.org",
        sum = "h1:JkRdGP3zvTtTbabWSAC6n67ka30y7gOzWAah4XYJSfw=",
        version = "v0.0.0-20190313082347-94abd6928b1d",
    )
    go_repository(
        name = "org_golang_x_xerrors",
        importpath = "golang.org/x/xerrors",
        sum = "h1:9zdDQZ7Thm29KFXgAX/+yaf3eVbP7djjWp/dXAppNCc=",
        version = "v0.0.0-20190717185122-a985d3407aa7",
    )
    go_repository(
        name = "org_uber_go_dig",
        importpath = "go.uber.org/dig",
        sum = "h1:E5/L92iQTNJTjfgJF2KgU+/JpMaiuvK2DHLBj0+kSZk=",
        version = "v1.7.0",
    )
    go_repository(
        name = "org_uber_go_fx",
        importpath = "go.uber.org/fx",
        sum = "h1:7OAz8ucp35AU8eydejpYG7QrbE8rLKzGhHbZlJi5LYY=",
        version = "v1.9.0",
    )
    go_repository(
        name = "org_uber_go_goleak",
        importpath = "go.uber.org/goleak",
        sum = "h1:G3eWbSNIskeRqtsN/1uI5B+eP73y3JUuBsv9AZjehb4=",
        version = "v0.10.0",
    )
    go_repository(
        name = "tools_gotest",
        importpath = "gotest.tools",
        sum = "h1:5USw7CrJBYKqjg9R7QlA6jzqZKEAtvW82aNmsxxGPxw=",
        version = "v2.1.0+incompatible",
    )
    go_repository(
        name = "tools_gotest_gotestsum",
        importpath = "gotest.tools/gotestsum",
        sum = "h1:LdVJDg3RHrci4MbupUgSkwPCikz4kTzDHWtUahDAleY=",
        version = "v0.3.4",
    )
