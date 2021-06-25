# Protoconf integration with Terraform for managing Kubernetes resources

### Prerequisites

- The `terraform` binary in your `$PATH`
- The `protoconf` binary in your `$PATH`

### Prepare

Create a `providers.tf` file containing the providers declarations you need.

```hcl
provider "kubernetes" {}
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
$  ls -l src/terraform/
total 992
-rw-r--r--. 1 marius marius  81687 Jun 25 12:39 kubernetes-data.proto
-rw-r--r--. 1 marius marius   1963 Jun 25 12:39 kubernetes-provider.proto
-rw-r--r--. 1 marius marius 911814 Jun 25 12:39 kubernetes-resources.proto
-rw-r--r--. 1 marius marius    461 Jun 25 12:39 meta.proto
-rw-r--r--. 1 marius marius   9100 Jun 25 12:39 terraform.proto
```

### Create directory for the Starlark configuration file (.pconf)

```shell
$ mkdir src/proto-kube/
```

### Create the Starlark configuration file (.pconf)

```python
# vim: filetype=python
# ./src/proto-kube/kube-pod.pconf

load("//terraform/terraform.proto", "Terraform")
load("//terraform/kubernetes-provider.proto", "Kubernetes")
load("//terraform/kubernetes-resources.proto", "KubernetesPod")

tf = Terraform(
    provider=Terraform.Providers(
        kubernetes=[Kubernetes(config_path="/path/to/kubeconfig")]
    ),
    resource=Terraform.Resources(),
    output={},
)


name = KubernetesPod.Metadata(name="example-pod")
spec = KubernetesPod.Spec(
    container=[KubernetesPod.Spec.Container(
        name="test-container",
        image="centos/tools",
        command=["/bin/bash", "-c", "sleep 2000000000000"],
    )]
)

tf.resource.kubernetes_pod["my_pod"] = KubernetesPod(metadata=name, spec=spec)


def main():
    return tf
```

### Compile the config

```shell
$ protoconf compile .
```

### Check the json output

```shell
cat materialized_config/proto-kube/kube-pod.materialized_JSON | jq .
```

```json
{
  "protoFile": "terraform/terraform.proto",
  "value": {
    "@type": "type.googleapis.com/terraform.Terraform",
    "provider": {
      "kubernetes": [
        {
          "config_path": "/path/to/kubeconfig"
        }
      ]
    },
    "resource": {
      "kubernetes_pod": {
        "my_pod": {
          "metadata": {
            "name": "example-pod"
          },
          "spec": {
            "container": {
              "command": [
                "/bin/bash",
                "-c",
                "sleep 2000000000000"
              ],
              "image": "centos/tools",
              "name": "test-container"
            }
          }
        }
      }
    }
  }
}
```

## Prepare to run Terraform

### Create Terraform working directory

```shell
$ mkdir tf
```

### Process json required by Terraform
```shell
$ cat materialized_config/proto-kube/kube-pod.materialized_JSON | \
      jq '.value | del(.["@type"])' > tf/proto-kube.tf.json
```

### Check the json required by Terraform
```shell
$ cat tf/proto-kube.tf.json
{
  "provider": {
    "kubernetes": [
      {
        "config_path": "./kubeconfig"
      }
    ]
  },
  "resource": {
    "kubernetes_pod": {
      "my_pod": {
        "metadata": {
          "name": "example-pod"
        },
        "spec": {
          "container": {
            "command": [
              "/bin/bash",
              "-c",
              "sleep 2000000000000"
            ],
            "image": "centos/tools",
            "name": "test-container"
          }
        }
      }
    }
  }
}

```

### Run Terraform init
```shell
$ cd tf
~/tf $ terraform init

Initializing the backend...

Initializing provider plugins...
- Finding latest version of hashicorp/kubernetes...
- Installing hashicorp/kubernetes v2.3.2...
- Installed hashicorp/kubernetes v2.3.2 (signed by HashiCorp)

Terraform has created a lock file .terraform.lock.hcl to record the provider
selections it made above. Include this file in your version control repository
so that Terraform can guarantee to make the same selections by default when
you run "terraform init" in the future.

Terraform has been successfully initialized!

You may now begin working with Terraform. Try running "terraform plan" to see
any changes that are required for your infrastructure. All Terraform commands
should now work.

If you ever set or change modules or backend configuration for Terraform,
rerun this command to reinitialize your working directory. If you forget, other
commands will detect it and remind you to do so if necessary.

```

### Run Terraform plan
```shell
~/tf $ terraform plan
Terraform used the selected providers to generate the following execution plan. Resource actions are indicated with the following symbols:
  + create
[ ... ]
Plan: 1 to add, 0 to change, 0 to destroy.

──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

```

### Run Terraform apply
```shell
~/tf $ terraform apply -auto-approve

Terraform used the selected providers to generate the following execution plan. Resource actions are indicated with the following symbols:
  + create

Terraform will perform the following actions:

  # kubernetes_pod.my_pod will be created
  + resource "kubernetes_pod" "my_pod" {
      + id = (known after apply)

      + metadata {
          + generation       = (known after apply)
          + name             = "example-pod"
          + namespace        = "default"
          + resource_version = (known after apply)
          + uid              = (known after apply)
        }

      + spec {
          + automount_service_account_token  = true
          + dns_policy                       = "ClusterFirst"
          + enable_service_links             = true
          + host_ipc                         = false
          + host_network                     = false
          + host_pid                         = false
          + hostname                         = (known after apply)
          + node_name                        = (known after apply)
          + restart_policy                   = "Always"
          + service_account_name             = (known after apply)
          + share_process_namespace          = false
          + termination_grace_period_seconds = 30

          + container {
              + command                    = [
                  + "/bin/bash",
                  + "-c",
                  + "sleep 2000000000000",
                ]
              + image                      = "centos/tools"
              + image_pull_policy          = (known after apply)
              + name                       = "test-container"
              + stdin                      = false
              + stdin_once                 = false
              + termination_message_path   = "/dev/termination-log"
              + termination_message_policy = (known after apply)
              + tty                        = false

              + resources {
                  + limits   = (known after apply)
                  + requests = (known after apply)
                }
            }

[ ... ]

Plan: 1 to add, 0 to change, 0 to destroy.
kubernetes_pod.my_pod: Creating...
kubernetes_pod.my_pod: Still creating... [10s elapsed]
kubernetes_pod.my_pod: Still creating... [20s elapsed]
kubernetes_pod.my_pod: Still creating... [30s elapsed]
kubernetes_pod.my_pod: Still creating... [40s elapsed]
kubernetes_pod.my_pod: Still creating... [50s elapsed]
kubernetes_pod.my_pod: Still creating... [1m0s elapsed]
kubernetes_pod.my_pod: Still creating... [1m10s elapsed]
kubernetes_pod.my_pod: Still creating... [1m20s elapsed]
kubernetes_pod.my_pod: Still creating... [1m30s elapsed]
kubernetes_pod.my_pod: Creation complete after 1m34s [id=default/example-pod]

Apply complete! Resources: 1 added, 0 changed, 0 destroyed.
```
