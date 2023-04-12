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

### Generate the terraform protos

```shell
$ protoconf import terraform
```

Validate the outpus

```shell
$ find src/terraform
src/terraform
src/terraform/v1
src/terraform/v1/terraform.proto
src/terraform/v1/meta.proto
src/terraform/kubernetes
src/terraform/kubernetes/datasources
src/terraform/kubernetes/datasources/v2
src/terraform/kubernetes/datasources/v2/namespace.proto
src/terraform/kubernetes/datasources/v2/all.proto
src/terraform/kubernetes/datasources/v2/persistent.proto
src/terraform/kubernetes/datasources/v2/storage.proto
src/terraform/kubernetes/datasources/v2/service.proto
src/terraform/kubernetes/datasources/v2/ingress.proto
src/terraform/kubernetes/datasources/v2/config.proto
src/terraform/kubernetes/datasources/v2/pod.proto
src/terraform/kubernetes/datasources/v2/secret.proto
src/terraform/kubernetes/resources
src/terraform/kubernetes/resources/v2
src/terraform/kubernetes/resources/v2/mutating.proto
src/terraform/kubernetes/resources/v2/priority.proto
src/terraform/kubernetes/resources/v2/namespace.proto
src/terraform/kubernetes/resources/v2/validating.proto
src/terraform/kubernetes/resources/v2/stateful.proto
src/terraform/kubernetes/resources/v2/manifest.proto
src/terraform/kubernetes/resources/v2/cluster.proto
src/terraform/kubernetes/resources/v2/api.proto
src/terraform/kubernetes/resources/v2/job.proto
src/terraform/kubernetes/resources/v2/persistent.proto
src/terraform/kubernetes/resources/v2/daemonset.proto
src/terraform/kubernetes/resources/v2/cron.proto
src/terraform/kubernetes/resources/v2/role.proto
src/terraform/kubernetes/resources/v2/deployment.proto
src/terraform/kubernetes/resources/v2/storage.proto
src/terraform/kubernetes/resources/v2/csi.proto
src/terraform/kubernetes/resources/v2/endpoints.proto
src/terraform/kubernetes/resources/v2/service.proto
src/terraform/kubernetes/resources/v2/ingress.proto
src/terraform/kubernetes/resources/v2/default.proto
src/terraform/kubernetes/resources/v2/certificate.proto
src/terraform/kubernetes/resources/v2/replication.proto
src/terraform/kubernetes/resources/v2/limit.proto
src/terraform/kubernetes/resources/v2/horizontal.proto
src/terraform/kubernetes/resources/v2/resource.proto
src/terraform/kubernetes/resources/v2/network.proto
src/terraform/kubernetes/resources/v2/config.proto
src/terraform/kubernetes/resources/v2/daemon.proto
src/terraform/kubernetes/resources/v2/pod.proto
src/terraform/kubernetes/resources/v2/secret.proto
src/terraform/kubernetes/provider
src/terraform/kubernetes/provider/v2
src/terraform/kubernetes/provider/v2/kubernetes.proto
```

### Create directory for the Starlark configuration file (.pconf)

```shell
$ mkdir src/proto-kube/
```

### Create the Starlark configuration file (.pconf)

```python
# vim: filetype=python
# ./src/proto-kube/kube-pod.pconf

load("//terraform/v1/terraform.proto", "Terraform")
load("//terraform/kubernetes/provider/v2/kubernetes.proto", "Kubernetes")
load("//terraform/kubernetes/resources/v2/pod.proto", "KubernetesPod")

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
cat materialized_config/proto-kube/kube-pod.materialized_JSON
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
              "command": ["/bin/bash", "-c", "sleep 2000000000000"],
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
