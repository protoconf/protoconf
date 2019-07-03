<div align="center"><h1>Protoconf</h1></div>
<div align="center">
  <img alt="logo" src="https://lior2b.github.io/temp/protoconf.svg" height="128">
</div>
<div align="center">
<h4>codify configuration, instant delivery</h4>
</div>

<div align="center">
  <h3>
    <a href="#quick-start">
      Quick start
    </a>
    <span> | </span>
    <a href="#production-setup">
      Production setup
    </a>
    <span> | </span>
    <a href="#build-from-source">
      Build from source
    </a>
    <span> | </span>
    <a href="https://lior2b.github.io/temp/">
      Web IDE
    </a>
  </h3>
</div>

## Introduction
Modern services are comprised of many dynamic variables, that need to be changed regularly. Today, the process is unstructured and error prone. From ML model variables, kill switches, gradual rollout configuration, A/B experiment configuration and more - developers want their code to allow to be configured to the finer details.

**Protoconf is a modern approach to software configuration**, inspired by [Facebook's Configerator](https://research.fb.com/publications/holistic-configuration-management-at-facebook/).

Using Protoconf enables:

* **Code review for configuration changes**
  Enables the battle tested flow of pull-request & code-review. Configuration auditing out of the box (who did what, when?). The repository is the source of truth for the configuration deployed to production.
* **No service restart required to pick up changes**
  Instant delivery of configuration updates. Encourages writing software that doesn't know downtime.
* **Clear representation of complex configuration**
  Write configuration in Starlark (a Python dialect), no more copying & pasting from huge JSON files.
* **Automated validation**
  Config follows a fully-typed (Protobuf) schema. This allows writing validation code in Starlark, to verify your configuration before it is committed.



#### Configuration update flow

<div align="center">
  <img src="https://lior2b.github.io/temp/protoconf_flow.png">
</div>

#### How this looks from the service's eyes

This is roughly how configuration is consumed by a service. This paradigm encourages you to write software that can reconfigure itself in runtime rather than require a restart:

<div align="center">
  <img src="https://lior2b.github.io/temp/protoconf_api.png" width="400">
</div>

As Protoconf uses Protobuf and gRPC, it supports delivering configuration to [all major languages](https://github.com/protocolbuffers/protobuf/blob/master/docs/third_party.md). See also: [Protobuf overview](https://developers.google.com/protocol-buffers/docs/overview).

## Quick start
Step by step instructions to start developing with Protoconf, with an example from an imaginary Python web crawler service. See full example under `examples/`.
1. Install the `protoconf` binary (see [build from source](#build-from-source))

2. Write a Protobuf schema under `protoconf/src/`(syntax guide https://developers.google.com/protocol-buffers/docs/proto3)
	
	1. For example: `protoconf/src/crawler/crawler.proto`
	
	```protobuf
	syntax = "proto3";
	
	message Crawler {
	  string user_agent = 1;
	  int32 http_timeout = 2;
	  bool follow_redirects = 3;
	}
	
	message CrawlerService {
	  repeated Crawler crawlers = 1;
	  enum AdminPermission {
	    READ_WRITE = 0;
	    GOD_MODE = 1;
	  }
	  map<string, AdminPermission> admins = 2;
	  int32 log_level = 3;
	}
	```
	2. Pro tip: adding fields to an existing schema? Don't worry, Protobuf is backward and forward compatible (https://en.wikipedia.org/wiki/Protocol_Buffers)

3. Write validators *(optional)*
     1. Write a Starlark file alongside the `.proto` file, with a `.proto-validator` suffix
     2. For example: `protoconf/src/crawler/crawler.proto-validator`

   ```python
   load("crawler.proto", "Crawler", "CrawlerService")
   
   def test_crawlers_not_empty(cs):
       if len(cs.crawlers) < 1:
           fail("Crawlers can't be empty")
   
   add_validator(CrawlerService, test_crawlers_not_empty)
   
   def test_http_timeout(c):
       MIN_TIMEOUT = 10
       if c.http_timeout < MIN_TIMEOUT:
           fail("Crawler HTTP timeout must be at least %d, got %d" % (MIN_TIMEOUT, c.http_timeout))
   
   add_validator(Crawler, test_http_timeout)
   ```

4. Write a config
	1. A Starlark `.pconf` file. Your code can be modular, export functions (ideally in `.pinc` files), and build a complete custom stack for your configuration needs.
	2. For example: `protoconf/src/crawler/text_crawler.pconf`
	
	   ```python
	   load("crawler.proto", "Crawler", "CrawlerService")
	   
	   def default_crawler():
	       return Crawler(user_agent="Linux", http_timeout=30)
	   
	   def main():
	       crawlers = []
	   
	       for i in range(3):
	           crawler = default_crawler()
	           crawler.http_timeout = 30 + 30*i
	               if i == 0:
	                   crawler.follow_redirects = True
	           crawlers.append(crawler)
	   
	       admins = {'superuser': CrawlerService.AdminPermission.GOD_MODE}
	       return CrawlerService(crawlers=crawlers, admins=admins, log_level=2)
	   ```
	
	3. Compile with `protoconf compile`, this will create a materialized config file under `protoconf/materialized_configs/`
	4. For example: `protoconf compile protoconf/ crawler/text/crawler` will create `protoconf/materialized_config/crawler/text_crawler.materialized_JSON`
	
	```json
	{
	  "protoFile": "crawler/crawler.proto",
	  "value": {
	    "@type": "https://CrawlerService",
	    "admins": {
	      "superuser": "GOD_MODE"
	    },
	    "crawlers": [
	      {
	        "userAgent": "Linux",
	        "httpTimeout": 30,
	        "followRedirects": true
	      },
	      {
	        "userAgent": "Linux",
	        "httpTimeout": 60
	      },
	      {
	        "userAgent": "Linux",
	        "httpTimeout": 90
	      }
	    ],
	    "logLevel": 2
	  }
	}
	```
	
5. Run the Protoconf agent in dev mode
	```bash
	protoconf agent protoconf/
	```
	
6. Prepare your application to work with Protobuf configs coming from Protoconf
	1. Compile your `.proto` schema, this will generate an object to work with.
	   For Python you can use `grpcio-tools`, for example:
	
	   ```bash
	   pip3 install grpcio-tools
	   python3 -m grpc_tools.protoc --python_out=. -I../protoconf/src ../protoconf/src/crawler/crawler.proto
	   ```
	Other languages can use the `protoc` binary (https://developers.google.com/protocol-buffers/docs/tutorials).
	2. Install the Protoconf Python library:
	```bash
	pip3 install -r python/requirements.txt python/
	```
	3. In your code, setup a connection to Protoconf and get the config. See full example under `examples/`. The code mainly consists of:
	```python
	from protoconf import ProtoconfSync
	from crawler.crawler_pb2 import CrawlerService
	
	protoconf = ProtoconfSync()
	crawler_service = protoconf.get_and_subscribe("crawler/text_crawler", CrawlerService, got_config)
	print("config:", crawler_service)
	
	def got_config(new_crawler_service):
	    print("got a new config:", new_crawler_service)
	```
	
7. Commit all changes under `protoconf/` (including the `.materialized_JSON` files)

## Production setup
1. Run your preferred key-value store (e.g. Consul)
2. Run the Protoconf agent: `protoconf agent`
3. Setup a commit hook in your repository server (e.g. Github) that runs `protoconf update` on changed `.materialized_JSON` files

## Build from source
1. Install Bazel: https://docs.bazel.build/versions/master/install.html
2. Clone Protoconf from gitlab: `git clone git@gitlab.com:protoconf/protoconf.git`
3. Build the binary: `cd protoconf && bazel build :protoconf`
4. Copy the binary to your `$PATH`, for example: `sudo cp bazel-bin/agent/linux_amd64_stripped/protoconf /usr/local/bin/`

## Trying the example
1. Make sure Consul is listening locally on default port (you can achieve this with `consul agent -dev`)
2. `cd src`
3. Run the agent: `bazel run //agent`
4. Compile the Protoconf config: `bazel run //compiler "$(pwd)/examples/protoconf" crawler/text_crawler.pconf`
5. Insert the Protoconf config to Consul: `bazel run //inserter "$(pwd)/examples/protoconf" crawler/text_crawler.materialized_JSON`
6. Run the Go client: `bazel run //examples/grpc_clients/go_client`, the client will get the config from the agent and will listen to changes
7. Change the config file at `examples/protoconf/src/crawler/text_crawler.pconf`
8. Repeat steps 4 & 5 to recompile and re-insert the config, observe the client got the updated config


## Run CI
1. Download `drone-cli` from https://github.com/drone/drone-cli/releases.
2. Copy the drone binary to your `$PATH` and make it executable
3. Run: `drone exec --pipeline default`

