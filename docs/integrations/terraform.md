# Protoconf integration with Terraform

### Prerequists

- The `terraform` binray in your `$PATH`
- The `protoconf` binary in your `$PATH`

### Prepare

Create a `providers.tf` file containing the providers declarations you need.

```hcl
provider "random" {}
provider "tls" {}
```

### Initialize Terraform

```shell
$ terraform init
```

### Copy all providers binaries to same directory

```shell
$ find .terraform -type f -exec cp {} .terraform \;
```

### Generate the terraform protos

```shell
$ protoconf import terraform -import_path .terraform
```

Validate the outpus

```shell
$  ll src/terraform
total 48K
drwxr-xr-x 10 smintz  320 May 23 10:46 .
drwxr-xr-x  3 smintz   96 May 23 10:39 ..
-rw-r--r--  1 smintz  461 May 23 10:46 meta.proto
-rw-r--r--  1 smintz   51 May 23 10:46 random-data.proto
-rw-r--r--  1 smintz  371 May 23 10:46 random-provider.proto
-rw-r--r--  1 smintz 8.1K May 23 10:46 random-resources.proto
-rw-r--r--  1 smintz 4.5K May 23 10:46 terraform.proto
-rw-r--r--  1 smintz 1.9K May 23 10:46 tls-data.proto
-rw-r--r--  1 smintz  362 May 23 10:46 tls-provider.proto
-rw-r--r--  1 smintz 6.1K May 23 10:46 tls-resources.proto
```

The `src/terraform/terraform.proto` should looks like this:

```proto
syntax = "proto3";

package terraform;

import "terraform/random-provider.proto";

import "terraform/random-resources.proto";

import "terraform/tls-data.proto";

import "terraform/tls-provider.proto";

import "terraform/tls-resources.proto";

message Terraform {
  Resources resource = 1;

  Datasources data = 2;

  Providers provider = 3;

  map<string, Variable> variable = 4;

  map<string, Output> output = 5;

  map<string, string> locals = 6;

  Module module = 7;

  TerraformSettings terraform = 8;

  message Resources {
    map<string, random.resources.RandomId> random_id = 1 [json_name = "random_id"];

    map<string, random.resources.RandomInteger> random_integer = 2 [json_name = "random_integer"];

    map<string, random.resources.RandomPassword> random_password = 3 [json_name = "random_password"];

    map<string, random.resources.RandomPet> random_pet = 4 [json_name = "random_pet"];

    map<string, random.resources.RandomShuffle> random_shuffle = 5 [json_name = "random_shuffle"];

    map<string, random.resources.RandomString> random_string = 6 [json_name = "random_string"];

    map<string, random.resources.RandomUuid> random_uuid = 7 [json_name = "random_uuid"];

    map<string, tls.resources.TlsCertRequest> tls_cert_request = 8 [json_name = "tls_cert_request"];

    map<string, tls.resources.TlsLocallySignedCert> tls_locally_signed_cert = 9 [json_name = "tls_locally_signed_cert"];

    map<string, tls.resources.TlsPrivateKey> tls_private_key = 10 [json_name = "tls_private_key"];

    map<string, tls.resources.TlsSelfSignedCert> tls_self_signed_cert = 11 [json_name = "tls_self_signed_cert"];
  }

  message Datasources {
    map<string, tls.data.TlsCertificate> tls_certificate = 1 [json_name = "tls_certificate"];

    map<string, tls.data.TlsPublicKey> tls_public_key = 2 [json_name = "tls_public_key"];
  }

  message Providers {
    repeated random.provider.Random random = 1;

    repeated tls.provider.Tls tls = 2;
  }

  message Variable {
    string type = 1;

    string description = 2;

    string default = 3;
  }

  message Output {
    string value = 1;
  }

  message Module {
  }

  message TerraformSettings {
    string required_version = 1 [json_name = "required_version"];

    map<string, Provider> required_providers = 2;

    Backend backend = 3;

    message Provider {
      string source = 1;

      string version = 2;
    }

    message Backend {
      oneof config {
        BackendLocal local = 1;

        BackendS3 s3 = 2;
      }

      message BackendLocal {
        string path = 1;

        string workspace_dir = 2 [json_name = "workspace_dir"];
      }

      message BackendS3 {
        string region = 1;

        string access_key = 2 [json_name = "access_key"];

        string secret_key = 3 [json_name = "secret_key"];

        string iam_endpoint = 4 [json_name = "iam_endpoint"];

        string max_retries = 5 [json_name = "max_retries"];

        string profile = 6;

        string shared_credentials_file = 7 [json_name = "shared_credentials_file"];

        string skip_credentials_validation = 8 [json_name = "skip_credentials_validation"];

        string skip_region_validation = 9 [json_name = "skip_region_validation"];

        string skip_metadata_api_check = 10 [json_name = "skip_metadata_api_check"];

        string sts_endpoint = 11 [json_name = "sts_endpoint"];

        string token = 12;

        string assume_role_duration_seconds = 13 [json_name = "assume_role_duration_seconds"];

        string assume_role_policy = 14 [json_name = "assume_role_policy"];

        string assume_role_policy_arns = 15 [json_name = "assume_role_policy_arns"];

        string assume_role_tags = 16 [json_name = "assume_role_tags"];

        string assume_role_transitive_tag_keys = 17 [json_name = "assume_role_transitive_tag_keys"];

        string external_id = 18 [json_name = "external_id"];

        string role_arn = 19 [json_name = "role_arn"];

        string session_name = 20 [json_name = "session_name"];

        string bucket = 21;

        string key = 22;

        string acl = 23;

        string encrypt = 24;

        string endpoint = 25;

        string force_path_style = 26 [json_name = "force_path_style"];

        string kms_key_id = 27 [json_name = "kms_key_id"];

        string sse_customer_key = 28 [json_name = "sse_customer_key"];

        string workspace_key_prefix = 29 [json_name = "workspace_key_prefix"];

        string dynamodb_endpoint = 30 [json_name = "dynamodb_endpoint"];

        string dynamodb_table = 31 [json_name = "dynamodb_table"];
      }
    }
  }
}
```

### Create a `.tf.pconf` file

```python
# vim: filetype=python
# ./src/tfdemo/tfdemo.tf.pconf
load("//terraform/terraform.proto", "Terraform")
load("//terraform/random-provider.proto", "Random")
load("//terraform/random-resources.proto", "RandomPet")

tf = Terraform(
    provider=Terraform.Providers(random=[Random()]),
    resource=Terraform.Resources(),
    output={},
)

tf.resource.random_pet["my_dog_name"] = RandomPet()
tf.output["my_dog_name"] = Terraform.Output(value="${random_pet.my_dog_name.id}")


def main():
    return tf
```

### compile the config

```shell
$ protoconf compile .
```

Check the output

```shell
cat materialized_config/tfdemo/tfdemo.tf.materialized_JSON| jq
```

```json
{
  "protoFile": "terraform/terraform.proto",
  "value": {
    "@type": "type.googleapis.com/terraform.Terraform",
    "output": {
      "my_dog_name": {
        "value": "${random_pet.my_dog_name.id}"
      }
    },
    "provider": {
      "random": [
        {}
      ]
    },
    "resource": {
      "random_pet": {
        "my_dog_name": {}
      }
    }
  }
}
```

### prepare to run terraform

```shell
$ mkdir tf
$ cat materialized_config/tfdemo/tfdemo.tf.materialized_JSON | jq '.value | del(.["@type"])' > tf/test.tf.json
$ cd tf
$ terraform init
```
```shell
$ terraform plan

An execution plan has been generated and is shown below.
Resource actions are indicated with the following symbols:
  + create

Terraform will perform the following actions:

  # random_pet.my_dog_name will be created
  + resource "random_pet" "my_dog_name" {
      + id        = (known after apply)
      + length    = 2
      + separator = "-"
    }

Plan: 1 to add, 0 to change, 0 to destroy.

Changes to Outputs:
  + my_dog_name = (known after apply)

------------------------------------------------------------------------

Note: You didn't specify an "-out" parameter to save this plan, so Terraform
can't guarantee that exactly these actions will be performed if
"terraform apply" is subsequently run.
```
```shell
$ terraform apply -auto-approve
random_pet.my_dog_name: Creating...
random_pet.my_dog_name: Creation complete after 0s [id=key-zebra]

Apply complete! Resources: 1 added, 0 changed, 0 destroyed.

Outputs:

my_dog_name = "key-zebra"
```

