---
id: intro
title: Introduction
sidebar_position: 1
---

# Introduction

protoconf is a configuration service based on Google's Protocol Buffers (Protobuf). It provides a structured, robust, and scalable solution for managing configurations across different services in a microservices environment.

protoconf uses Protobuf to define configuration structures, and Starlark (a Python-like language used in tools like Bazel) for the configuration logic. These technologies provide a strong schema and type checking, ensuring configuration consistency and reducing errors.

## How protoconf Works

In protoconf, configuration is defined as Protobuf messages. The configuration logic is written in Starlark and compiled to produce the final configuration. This can then be consumed by your applications either as a file or via a gRPC service.

protoconf also supports versioning and validation. This means you can have different versions of configuration for different versions of your service and validate your configuration against specific rules. This makes configuration management more flexible and reliable.

The following sections will guide you through installing protoconf, creating your first protoconf configuration, and advanced topics like validation and consuming configuration updates.
