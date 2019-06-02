# Protoconf

Based on Configerator by Facebook, see https://research.fb.com/wp-content/uploads/2016/11/holistic-configuration-management-at-facebook.pdf

## Build
```bash
cd src && bazel build //agent //examples/go_client
```
## Trying it yourself
1. Make sure Consul is listening locally on default port (you can achieve this with `consul agent -dev`)
2. Run the agent: `bazel run //agent`
3. Run the example agent client: `bazel run //examples/go_client example/consul/path`
4. Change `example/consul/path` (`bazel run //examples:inserter` or `./examples/inserter/inserter.sh`) and see the client gets notified

## Future Work
* Canary service

## Run CI

* Download `drone-cli` from https://github.com/drone/drone-cli/releases.
* Copy the drone command line to your `$PATH` and make it executable.
* Run: `drone exec --pipeline default`
