# Protoconf

Based on Configerator by Facebook, see https://research.fb.com/wp-content/uploads/2016/11/holistic-configuration-management-at-facebook.pdf

## Build
```bash
cd src && bazel build //...
```
## Trying the example
1. Make sure Consul is listening locally on default port (you can achieve this with `consul agent -dev`)
2. `cd src`
3. Run the agent: `bazel run //agent`
4. Compile the config: `bazel run //compiler "$(pwd)/examples/protoconf" address_book/my_address_book.pconf`
5. Insert the config: `bazel run //inserter "$(pwd)/examples/protoconf" address_book/my_address_book.materialized_JSON`
6. Run the Go client: `bazel run //examples/go_client`, the client will get the config from the agent and will listen to changes
7. Change the config file at `examples/protoconf/src/address_book/my_address_book.pconf`
8. Repeat steps 4 & 5 to recompile and re-insert the config, observe the client got the updated config

#### Bonus - you can find client implementations in the examples directory:
- Python: `cd examples/python_asyncio_client && bazel run :python_asyncio_client`
- Rust `cd examples/rust_client && bazel run :rust_client`
- Java: `cd examples/java_client && bazel run :java_client`
- Ruby: `cd examples/ruby_client && bazel run :ruby_client`

## Future Work
* Canary service

## Run CI

* Download `drone-cli` from https://github.com/drone/drone-cli/releases.
* Copy the drone command line to your `$PATH` and make it executable.
* Run: `drone exec --pipeline default`
