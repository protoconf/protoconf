load("@bazel_gazelle//:deps.bzl", "go_repository")

def deps():
    go_repository(
        name = "com_github_hashicorp_go_plugin",
        importpath = "github.com/hashicorp/go-plugin",
        sum = "h1:4OtAfUGbnKC6yS48p0CtMX2oFYtzFZVv6rok3cRWgnE=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_github_hashicorp_terraform",
        importpath = "github.com/hashicorp/terraform",
        sum = "h1:LACXUTZvAGf8W/6wehHjOgi6YEMN7ejDUpnpll2qbJ0=",
        version = "v0.12.13",
    )
    go_repository(
        name = "com_github_zclconf_go_cty",
        importpath = "github.com/zclconf/go-cty",
        sum = "h1:uJwc9HiBOCpoKIObTQaLR+tsEXx1HBHnOsOOpcdhZgw=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_mitchellh_go_testing_interface",
        importpath = "github.com/mitchellh/go-testing-interface",
        sum = "h1:fzU/JVNcaqHQEcVFAKeR41fkiLdIPrefOvVG1VZ96U0=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_hashicorp_go_hclog",
        importpath = "github.com/hashicorp/go-hclog",
        sum = "h1:b86HUuA126IcSHyC55WjPo7KtCOVeTCKIjr+3lBhPxI=",
        version = "v0.10.0",
    )
    go_repository(
        name = "com_github_oklog_run",
        importpath = "github.com/oklog/run",
        sum = "h1:Ru7dDtJNOyC66gQ5dQmaCa0qIsAUFY3sFpK1Xk8igrw=",
        version = "v1.0.0",
    )
    go_repository(
        name = "org_golang_x_crypto",
        importpath = "golang.org/x/crypto",
        sum = "h1:pXVtWnwHkrWD9ru3sDxY/qFK/bfc0egRovX91EjWjf4=",
        version = "v0.0.0-20191112222119-e1110fd1c708",
    )
    go_repository(
        name = "com_github_hashicorp_go_version",
        importpath = "github.com/hashicorp/go-version",
        sum = "h1:3vNe/fWF5CBgRIguda1meWhsZHy3m8gCJ5wx+dIzX/E=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_hashicorp_yamux",
        importpath = "github.com/hashicorp/yamux",
        sum = "h1:W+SIwDdl3+jXWeidYySAgzytE3piq6GumXeBjFBG67c=",
        version = "v0.0.0-20190923154419-df201c70410d",
    )
    go_repository(
        name = "com_github_hashicorp_terraform_svchost",
        importpath = "github.com/hashicorp/terraform-svchost",
        sum = "h1:hjyO2JsNZUKT1ym+FAdlBEkGPevazYsmVgIMw7dVELg=",
        version = "v0.0.0-20191011084731-65d371908596",
    )
    go_repository(
        name = "com_github_hashicorp_go_getter",
        importpath = "github.com/hashicorp/go-getter",
        sum = "h1:ENHNi8494porjD0ZhIrjlAHnveSFhY7hvOJrV/fsKkw=",
        version = "v1.4.0",
    )
    go_repository(
        name = "com_github_hashicorp_hcl2",
        importpath = "github.com/hashicorp/hcl2",
        sum = "h1:PFfGModn55JA0oBsvFghhj0v93me+Ctr3uHC/UmFAls=",
        version = "v0.0.0-20191002203319-fb75b3253c80",
    )
    go_repository(
        name = "com_github_hashicorp_hcl",
        importpath = "github.com/hashicorp/hcl",
        sum = "h1:0Anlzjpi4vEasTeNFn2mLJgTSwt0+6sfsiTG8qcWGx4=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_hashicorp_hcl_v2",
        importpath = "github.com/hashicorp/hcl/v2",
        sum = "h1:efQznTz+ydmQXq3BOnRa3AXzvCeTq1P4dKj/z5GLlY8=",
        version = "v2.0.0",
    )
    go_repository(
        name = "com_github_agext_levenshtein",
        importpath = "github.com/agext/levenshtein",
        sum = "h1:0S/Yg6LYmFJ5stwQeRp6EeOcCbj7xiqQSdNelsXvaqE=",
        version = "v1.2.2",
    )
    go_repository(
        name = "com_github_vmihailenco_msgpack",
        importpath = "github.com/vmihailenco/msgpack",
        sum = "h1:dSLoQfGFAo3F6OoNhwUmLwVgaUXK79GlxNBwueZn0xI=",
        version = "v4.0.4+incompatible",
    )
    go_repository(
        name = "com_github_mitchellh_reflectwalk",
        importpath = "github.com/mitchellh/reflectwalk",
        sum = "h1:FVzMWA5RllMAKIdUSC8mdWo3XtwoecrH79BY70sEEpE=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_github_hashicorp_hil",
        importpath = "github.com/hashicorp/hil",
        sum = "h1:T1Q6ag9tCwun16AW+XK3tAql24P4uTGUMIn1/92WsQQ=",
        version = "v0.0.0-20190212132231-97b3a9cdfa93",
    )
    go_repository(
        name = "com_github_hashicorp_go_uuid",
        importpath = "github.com/hashicorp/go-uuid",
        sum = "h1:fv1ep09latC32wFoVwnqcnKJGnMSdBanPczbHAYm1BE=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_github_mitchellh_copystructure",
        importpath = "github.com/mitchellh/copystructure",
        sum = "h1:Laisrj+bAB6b/yJwB5Bt3ITZhGJdqmxquMKeZ+mmkFQ=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_mitchellh_mapstructure",
        importpath = "github.com/mitchellh/mapstructure",
        sum = "h1:fmNYVwqnSfB9mZU6OS2O6GsXM+wcskZDuKQzvN1EDeE=",
        version = "v1.1.2",
    )
    go_repository(
        name = "com_github_mitchellh_go_wordwrap",
        importpath = "github.com/mitchellh/go-wordwrap",
        sum = "h1:6GlHJ/LTGMrIJbwgdqdl2eEH8o+Exx/0m8ir9Gns0u4=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_apparentlymart_go_textseg",
        importpath = "github.com/apparentlymart/go-textseg",
        sum = "h1:rRmlIsPEEhUTIKQb7T++Nz/A5Q6C9IuX2wFoYVvnCs0=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_hashicorp_go_cleanhttp",
        importpath = "github.com/hashicorp/go-cleanhttp",
        sum = "h1:dH3aiDG9Jvb5r5+bYHsikaOUIpcM0xvgMXVoDkXMzJM=",
        version = "v0.5.1",
    )
    go_repository(
        name = "com_github_blang_semver",
        importpath = "github.com/blang/semver",
        sum = "h1:cQNTCjp13qL8KC3Nbxr/y2Bqb63oX6wdnnjpJbkM4JQ=",
        version = "v3.5.1+incompatible",
    )
    go_repository(
        name = "com_github_zclconf_go_cty_yaml",
        importpath = "github.com/zclconf/go-cty-yaml",
        sum = "h1:up11wlgAaDvlAGENcFDnZgkn0qUJurso7k6EpURKNF8=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_github_spf13_afero",
        importpath = "github.com/spf13/afero",
        sum = "h1:5jhuqJyZCZf2JRofRvN/nIFgIWNzPa3/Vz8mYylgbWc=",
        version = "v1.2.2",
    )
    go_repository(
        name = "com_github_mitchellh_hashstructure",
        importpath = "github.com/mitchellh/hashstructure",
        sum = "h1:ZkRJX1CyOoTkar7p/mLS5TZU4nJ1Rn/F8u9dGS02Q3Y=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_google_go_cmp",
        importpath = "github.com/google/go-cmp",
        sum = "h1:Xye71clBPdm5HgqGwUkwhbynsUJZhDbS20FvLhQ2izg=",
        version = "v0.3.1",
    )
    go_repository(
        name = "com_github_google_uuid",
        importpath = "github.com/google/uuid",
        sum = "h1:Gkbcsh/GbpXz7lPftLA3P6TYMwjCLYm83jiFQZF/3gY=",
        version = "v1.1.1",
    )
    go_repository(
        name = "com_github_apparentlymart_go_cidr",
        importpath = "github.com/apparentlymart/go-cidr",
        sum = "h1:NmIwLZ/KdsjIUlhf+/Np40atNXm/+lZ5txfTJ/SpF+U=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_github_bmatcuk_doublestar",
        importpath = "github.com/bmatcuk/doublestar",
        sum = "h1:2bNwBOmhyFEFcoB3tGvTD5xanq+4kyOZlB8wFYbMjkk=",
        version = "v1.1.5",
    )
    go_repository(
        name = "com_github_aws_aws_sdk_go",
        importpath = "github.com/aws/aws-sdk-go",
        sum = "h1:8muvpP+Bq5e0CDkM9PDZ6tN74fVUq5v3zSCRaZ93ykM=",
        version = "v1.25.33",
    )
    go_repository(
        name = "com_github_bgentry_go_netrc",
        importpath = "github.com/bgentry/go-netrc",
        sum = "h1:xDfNPAt8lFiC1UJrqV3uuy861HCTo708pDMbjHHdCas=",
        version = "v0.0.0-20140422174119-9fd32a8b3d3d",
    )
    go_repository(
        name = "org_golang_x_oauth2",
        importpath = "golang.org/x/oauth2",
        sum = "h1:SVwTIAaPC2U/AvvLNZ2a7OVsmBpC8L5BlwK1whH3hm0=",
        version = "v0.0.0-20190604053449-0f29369cfe45",
    )
    go_repository(
        name = "com_github_hashicorp_go_safetemp",
        importpath = "github.com/hashicorp/go-safetemp",
        sum = "h1:2HR189eFNrjHQyENnQMMpCiBAsRxzbTMIgBhEyExpmo=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_ulikunitz_xz",
        importpath = "github.com/ulikunitz/xz",
        sum = "h1:jGHAfXawEGZQ3blwU5wnWKQJvAraT7Ftq9EXjnXYgt8=",
        version = "v0.5.6",
    )
    go_repository(
        name = "org_golang_google_api",
        importpath = "google.golang.org/api",
        sum = "h1:Q3Ui3V3/CVinFWFiW39Iw0kMuVrRzYX0wN6OPFp0lTA=",
        version = "v0.13.0",
    )
    go_repository(
        name = "com_github_googleapis_gax_go_v2",
        importpath = "github.com/googleapis/gax-go/v2",
        sum = "h1:sjZBwGj9Jlw33ImPtvFviGYvseOtDM7hkSKB7+Tv3SM=",
        version = "v2.0.5",
    )
    go_repository(
        name = "com_github_jmespath_go_jmespath",
        importpath = "github.com/jmespath/go-jmespath",
        sum = "h1:pmfjZENx5imkbgOkpRUYLnmbU7UEFbjtDA2hxJ1ichM=",
        version = "v0.0.0-20180206201540-c2b33e8439af",
    )
    go_repository(
        name = "io_opencensus_go",
        importpath = "go.opencensus.io",
        sum = "h1:75k/FF0Q2YM8QYo07VPddOLBslDt1MZOdEslOHvmzAs=",
        version = "v0.22.2",
    )
    go_repository(
        name = "com_github_golang_groupcache",
        importpath = "github.com/golang/groupcache",
        sum = "h1:uHTyIjqVhYRhLbJ8nIiOJHkEZZ+5YoOsAbD3sk82NiE=",
        version = "v0.0.0-20191027212112-611e8accdfc9",
    )
    go_repository(
        name = "com_google_cloud_go_storage",
        importpath = "cloud.google.com/go/storage",
        sum = "h1:2Ze/3nQD5F+HfL0xOPM2EeawDWs+NPRtzgcre+17iZU=",
        version = "v1.3.0",
    )

    go_repository(
        name = "com_google_cloud_go",
        importpath = "cloud.google.com/go",
        sum = "h1:6ZHYIRlohUdU4LrLHbTsReY1eYy/MoZW1FsEyBuMXsk=",
        version = "v0.48.0",
        build_directives = [
          "gazelle:go_visibility @com_google_cloud_go_storage//:__subpackages__",
        ],
    )
    go_repository(
        name = "com_github_stretchr_testify",
        importpath = "github.com/stretchr/testify",
        sum = "h1:2E4SXV/wtOkTonXsotYi4li6zVWxYlZuYNCXe9XRJyk=",
        version = "v1.4.0",
    )
    go_repository(
        name = "com_github_davecgh_go_spew",
        importpath = "github.com/davecgh/go-spew",
        sum = "h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=",
        version = "v1.1.1",
    )
    go_repository(
        name = "in_gopkg_yaml_v2",
        importpath = "gopkg.in/yaml.v2",
        sum = "h1:ymVxjfMaHvXD8RqPRmzHHsB3VvucivSkIAvJFDI5O3c=",
        version = "v2.2.5",
    )
