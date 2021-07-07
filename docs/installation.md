# Installation

### On Linux/MacOS

```
export PROTOCONF_VERSION="0.1.4"
export PROTOCONF_OS=$(uname | tr '[A-Z]' '[a-z]')
# change to "arm64" if needed
export PROTOCONF_ARCH="amd64"
curl -LO https://github.com/protoconf/protoconf/releases/download/${PROTOCONF_VERSION}/protoconf-${PROTOCONF_OS}-${PROTOCONF_ARCH}-${PROTOCONF_VERSION}.tar.gz
sudo tar xvf protoconf-${PROTOCONF_OS}-${PROTOCONF_ARCH}-${PROTOCONF_VERSION}.tar.gz -C /usr/local/bin
```

### On Windows

Download from the github [releases page](https://github.com/protoconf/protoconf/releases).

## Validate the installation

```
$ protoconf
2020/04/26 10:16:59 proto: duplicate proto type registered: v1.ConfigSubscriptionRequest
2020/04/26 10:16:59 proto: duplicate proto type registered: v1.ConfigUpdate
Usage: protoconf [--version] [--help] <command> [<args>]

Available commands are:
    agent      Runs a Protoconf agent
    compile    Compile configs
    exec       Watches keys and execute on changes
    import
    insert     Insert a materialized config to the key-value store
    mutate     Write to mutation server
    serve      Runs a server
```
