---
sidebar_position: 1
---

# Terraform

## Installation and Initialization

First, install `protoconf-terraform` using Homebrew:

```shell
brew install protoconf/tap/protoconf-terraform
```

Alternatively, download it from the [Github repository](https://github.com/protoconf/protoconf-terraform).

Next, initialize the workspace:

```shell
protoconf-terraform init
```

Create a `provider.tf` file and include the providers you want to use:

```shell
provider "tls" {}
provider "null" {}
provider "random" {}
```

Then, initialize Terraform:

```shell
terraform init
```

This command downloads all the necessary providers to the local cache.

## Generation

Next, generate the provider schemas:

```shell
protoconf-terraform generate
```

This command connects to the providers, fetches their schemas, writes the schemas into protobuf files, and updates `./src/terraform/v1/terraform.proto` to link these files.

## Usage

Here's an example of how to use the API:

```python
load("//terraform/v1/util.pinc", "util")
load("//terraform/random/provider/v3/random.proto", "Random")
load("//terraform/random/resources/v3/pet.proto", "RandomPet")
load("//terraform/null/provider/v3/null.proto", "Null")
load("//terraform/null/datasources/v3/data.proto", "NullDataSource")

tf = util.Terraform(
    util.Provider(Random()),
    util.Resource(
        "dog",
        RandomPet(),
        lambda dog: util.Output(
            "dog_name", dog.id
        ),
    ),
    util.Data(
        "null_name",
        NullDataSource(),
        lambda data: util.Group(
            util.Output("null_random", data.random),
            util.Output("has_computed_default", data.has_computed_default),
        ),
    ),
    util.Module(
        "ssh_key",
        source="JamesWoolfenden/key/tls",
        version="0.0.6",
        out_dir="/tmp/sshkey",
        then=lambda output: util.Group(
            util.Output("public_key", output("public_key")),
        ),
    ),
)
```

Here's how to interpret the different parts of the example:

- `util.Terraform`: Initializes a new Terraform configuration.
- `util.Provider(Random())`: Adds a provider configuration for the "random" provider.
- `util.Resource`: Creates a new resource of type "random_pet" with the identifier "dog".
- `util.Data`: Creates a new data source of type "null_data_source" with the identifier "null_name".
- `util.Module`: Adds a module with the identifier "ssh_key".

## Exporting Configurations

After creating the `tf` Terraform object, there are two ways to export the configurations:

1. Use external tools like Terraform Cloud, Atlantis, env0 or Spacelift
1. Use `protoconf-terraform run` to watch for config changes in protoconf and apply them immediatly.

### External tools

Use the `.tf.json` config suffix to create Terraform compatible files under `./outputs`. These files can be used to either run locally on the developer's machine or to be picked up later by the developer's favourite Terraform management tool (recommended for infrastructure provisioning with long provisioning times like VPC, compute, clusters).

### Watch for Configs

Use `protoconf-terraform run`: This command needs a `SubscriptionConfig` for protoconf-terraform to know which configs to watch.

```python
load("//protoconf_terraform/config/v1/config.proto", "SubscriptionConfig")

def main():
    return SubscriptionConfig(keys=[
        "example/dog",
        "example/cat",
    ])
```

This approach is recommended for short running provisioning such as Kubernetes resources, DNS updates etc. You can use this approach with the mutation API to create:

- Continuous Deployment pipelines
- Ephemeral deployments (for development purposes)
- Canaries
- Automatic failovers

#### Docker Deployment

Running `protoconf-terraform` in a Docker container involves packaging the necessary files and running the container.

First, you'll need to create a `Dockerfile`. Here's an example `Dockerfile` for `protoconf-terraform`:

```docker
FROM homebrew/brew AS installer

RUN brew install protoconf/tap/protoconf-terraform hashicorp/tap/terraform

FROM alpine
# Add Maintainer Info
LABEL maintainer="Your Name <your.email@example.com>"

# Set the Current Working Directory inside the container
WORKDIR /app

COPY --from=installer /home/linuxbrew/.linuxbrew/bin/terraform /usr/local/bin/terraform
COPY --from=installer /home/linuxbrew/.linuxbrew/bin/protoconf-terraform /usr/local/bin/protoconf-terraform

# Install git.
# Git is required for fetching the dependencies.
RUN apk update && apk add --no-cache git

# Copy proto files
COPY ./src/terraform /app/src/terraform

# This container will be executable
ENTRYPOINT ["protoconf-terraform"]
```

To build the Docker image, navigate to the directory containing your `Dockerfile` and run:

```shell
docker build -t protoconf-terraform .
```

After building the Docker image, you can run the `protoconf-terraform` command within a Docker container using the following command:

```shell
docker run -it --rm protoconf-terraform COMMAND
```

Replace `COMMAND` with the command you want to run, such as `init`, `generate`, or `run`.

Remember that any changes made within the Docker container won't persist after the container is stopped. If you want to persist data across Docker runs, consider using Docker volumes.
