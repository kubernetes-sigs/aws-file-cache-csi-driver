[![Go Report Card](https://goreportcard.com/badge/github.com/kubernetes-sigs/aws-file-cache-csi-driver)](https://goreportcard.com/report/github.com/kubernetes-sigs/aws-file-cache-csi-driver)

## Amazon File Cache CSI Driver
### Overview

The [Amazon File Cache](https://docs.aws.amazon.com/fsx/latest/FileCacheGuide/) Container Storage Interface (CSI) Driver provides a [CSI](https://github.com/container-storage-interface/spec/blob/master/spec.md) interface used by container orchestrators to manage the lifecycle of Amazon file cache volumes.

### CSI Specification Compability Matrix
| AWS File Cache CSI Driver \ CSI Version | v1.x.x |
|-----------------------------------------|--------|
| v0.1.0                                  | yes    |

### Features

The following CSI interfaces are implemented:
* Controller Service: CreateVolume, DeleteVolume, ControllerGetCapabilities, ValidateVolumeCapabilities
* Node Service: NodePublishVolume, NodeUnpublishVolume, NodeGetCapabilities, NodeGetInfo, NodeGetId
* Identity Service: GetPluginInfo, GetPluginCapabilities, Probe

## Amazon File Cache CSI Driver on Kubernetes

The following sections are Kubernetes-specific. If you are a Kubernetes user, use the following for driver features, installation steps and examples.

### Kubernetes Version Compability Matrix
| AWS File Cache CSI Driver \ Kubernetes Version | v1.22+ |
|------------------------------------------------|--------|
| v0.1.0                                         | yes    |

### Container Images
| File Cache CSI Driver Version | Image                                                          |
|-------------------------------|----------------------------------------------------------------|
| v0.1.0                        | public.ecr.aws/fsx-csi-driver/aws-file-cache-csi-driver:v0.1.0 |

### Features
* Static provisioning - Amazon File Cache needs to be created manually first, then it could be mounted inside container as a volume using the Driver.
* Dynamic provisioning - uses persistent volume claim (PVC) to let Kubernetes create the Amazon File Cache for you and consumes the volume from inside container.
* Mount options - mount options can be specified in storageclass to define how the volume should be mounted.

**Notes**:
* For dynamically provisioned volumes, only one subnet is allowed inside a storageclass's `parameters.subnetId`. This is a [limitation](https://docs.aws.amazon.com/fsx/latest/APIReference/API_FileCacheCreating.html#FSx-Type-FileCacheCreating-SubnetIds) that is enforced by Amazon File Cache.

### Installation

#### Prerequisites
You must have:
* The AWS CLI installed and configured on your device or AWS CloudShell. You can check your current version with `aws --version | cut -d / -f2 | cut -d ' ' -f1`. Package managers such `yum`, `apt-get`, or Homebrew for macOS are often several versions behind the latest version of the AWS CLI. To install the latest version, see [Installing, updating, and uninstalling the AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) and [Quick configuration with aws configure](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-quickstart.html#cli-configure-quickstart-config) in the AWS Command Line Interface User Guide. The AWS CLI version installed in the AWS CloudShell may also be several versions behind the latest version. To update it, see [Installing AWS CLI to your home directory](https://docs.aws.amazon.com/cloudshell/latest/userguide/vm-specs.html#install-cli-software) in the AWS CloudShell User Guide.
* The `eksctl` command line tool installed on your device or AWS CloudShell. To install or update `eksctl`, see [Installing or updating eksctl](https://docs.aws.amazon.com/eks/latest/userguide/eksctl.html).
* The `kubectl` command line tool is installed on your device or AWS CloudShell. The version can be the same as or up to one minor version earlier or later than the Kubernetes version of your cluster. To install or upgrade `kubectl`, see [Installing or updating kubectl](https://docs.aws.amazon.com/eks/latest/userguide/install-kubectl.html).


1. Set a few variables to use in the remaining steps. Replace `my-csi-file-cache` with the name of the test cluster you want to create and `region-code` with the AWS Region that you want to create your test cluster in.
```shell
export cluster_name=my-csi-file-cache
export region_code=region-code
```

2. Create a test cluster.
```shell
eksctl create cluster \
  --name $cluster_name \
  --region $region_code \
  --with-oidc \
  --ssh-access \
  --ssh-public-key my-key
```

3. Set up driver permission

The driver requires IAM permission to talk to Amazon File Cache service to create/delete the file cache on user's behalf. There are several methods to grant driver IAM permission:

* Create a Kubernetes service account for the driver and attach the `AmazonFSxFullAccess` AWS-managed policy to the service account with the following command. If your cluster is in the AWS GovCloud (US-East) or AWS GovCloud (US-West) AWS Regions, then replace `arn:aws:` with `arn:aws-us-gov:`.

```shell
eksctl create iamserviceaccount \
    --name file-cache-csi-controller-sa \
    --namespace kube-system \
    --cluster $cluster_name \
    --attach-policy-arn arn:aws:iam::aws:policy/AmazonFSxFullAccess \
    --approve \
    --role-name AmazonEKSFileCacheCSIDriverFullAccess \
    --region $region_code
```


* Using worker node instance profile - grant all the worker nodes with proper permission by attach policy to the instance profile of the worker.
```shell
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ds:DescribeDirectories",
                "fsx:*"
            ],
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": "iam:CreateServiceLinkedRole",
            "Resource": "*",
            "Condition": {
                "StringEquals": {
                    "iam:AWSServiceName": [
                        "fsx.amazonaws.com"
                    ]
                }
            }
        },
        {
            "Effect": "Allow",
            "Action": "iam:CreateServiceLinkedRole",
            "Resource": "*",
            "Condition": {
                "StringEquals": {
                    "iam:AWSServiceName": [
                        "s3.data-source.lustre.fsx.amazonaws.com"
                    ]
                }
            }
        },
        {
            "Effect": "Allow",
            "Action": [
                "logs:CreateLogGroup",
                "logs:CreateLogStream",
                "logs:PutLogEvents"
            ],
            "Resource": [
                "arn:aws:logs:*:*:log-group:/aws/fsx/*"
            ]
        },
        {
            "Effect": "Allow",
            "Action": [
                "firehose:PutRecord"
            ],
            "Resource": [
                "arn:aws:firehose:*:*:deliverystream/aws-fsx-*"
            ]
        },
        {
            "Effect": "Allow",
            "Action": [
                "ec2:CreateTags"
            ],
            "Resource": [
                "arn:aws:ec2:*:*:route-table/*"
            ],
            "Condition": {
                "StringEquals": {
                    "aws:RequestTag/AmazonFSx": "ManagedByAmazonFSx"
                },
                "ForAnyValue:StringEquals": {
                    "aws:CalledVia": [
                        "fsx.amazonaws.com"
                    ]
                }
            }
        },
        {
            "Effect": "Allow",
            "Action": [
                "ec2:DescribeSecurityGroups",
                "ec2:DescribeSubnets",
                "ec2:DescribeVpcs"
            ],
            "Resource": "*",
            "Condition": {
                "ForAnyValue:StringEquals": {
                    "aws:CalledVia": [
                        "fsx.amazonaws.com"
                    ]
                }
            }
        }
    ]
}
```

4. Deploy driver
```sh
kubectl apply -k "github.com/kubernetes-sigs/aws-file-cache-csi-driver/deploy/kubernetes/overlays/stable/?ref=release-0.1"
```

Alternatively, you could also install the driver using helm:

```sh
helm repo add aws-file-cache-csi-driver https://kubernetes-sigs.github.io/aws-file-cache-csi-driver/
helm repo update
helm upgrade --install aws-file-cache-csi-driver --namespace kube-system aws-file-cache-csi-driver/aws-file-cache-csi-driver
```

#### To deploy a Kubernetes storage class, persistent volume claim, and sample application to verify that the CSI driver is working

1. Note the security group for your cluster. You can see it in the AWS Management Console under the Networking section or by using the following AWS CLI command.

```shell
aws eks describe-cluster --name $cluster_name --query cluster.resourcesVpcConfig.clusterSecurityGroupId
```

2. Create a security group for your Amazon FSx file cache according to the criteria shown in [Amazon VPC Security Groups](https://docs.aws.amazon.com/fsx/latest/FileCacheGuide/limit-access-security-groups.html#fsx-vpc-security-groups) in the Amazon File Cache User Guide. For the **VPC**, select the VPC of your cluster as shown under the **Networking** section. For "the security groups associated with your Lustre clients", use your cluster security group. You can leave the outbound rules alone to allow **All traffic**.

3. Download the storage class manifest with the following command.

```shell
curl -O https://raw.githubusercontent.com/kubernetes-sigs/aws-file-cache-csi-driver/main/examples/kubernetes/dynamic_provisioning/specs/storageclass.yaml
```

4. Edit the parameters section of the `storageclass.yaml` file. Replace every `example value` with your own values.

For information about the other parameters, see [Edit StorageClass](https://github.com/kubernetes-sigs/aws-file-cache-csi-driver/tree/main/examples/kubernetes/dynamic_provisioning#edit-storageclass).

```shell
parameters:
    subnetId: "subnet-0d7b5e117ad7b4961"
    securityGroupIds: "sg-05a37bfe01467059" 
    dataRepositoryAssociations: "fileCachePath=/ns1/,dataRepositoryPath=nfs://10.0.92.69/fsx/"
    fileCacheType(Optional): "Lustre"
    fileCacheTypeVersion(Optional): "2.12"
    weeklyMaintenanceStartTime(Optional): "6:00:00"
    LustreConfiguration(Optional): "{DeploymentType=CACHE_1,MetadataConfiguration=2400,perUnitStorageThroughput=1000}"
```
* **subnetId** – The subnet ID that the Amazon FSx file cache should be created in. Amazon FSx file cache isn't supported in all Availability Zones. Open the Amazon FSx Caches page at https://console.aws.amazon.com/fsx/#fc/file-caches/ to confirm that the subnet that you want to use is in a supported Availability Zone. The subnet can include your nodes, or can be a different subnet or VPC:
  * You can check for the node subnets in the AWS Management Console by selecting the node group under the Compute section.
  * If the subnet that you specify isn't the same subnet that you have nodes in, then your VPCs must be connected, and you must ensure that you have the necessary ports open in your security groups.
* **securityGroupIds** – The ID of the security group you created for the file cache.
* **dataRepositoryAssociations** – A list of up to 8 configurations for data repository associations (DRAs) to be created during the cache creation. The DRAs link the cache to either an Amazon S3 data repository or a Network File cache (NFS) data repository that supports the NFSv3 protocol.
* **other parameters (optional)** – For information about the other parameters, see Edit StorageClass on GitHub.

5. Create the storage class manifest.
 
```shell
kubectl apply -f storageclass.yaml
```
The example output is as follows.
```shell
storageclass.storage.k8s.io/fc-sc created
```

6. Download the persistent volume claim manifest.
```shell
curl -O https://raw.githubusercontent.com/kubernetes-sigs/aws-file-cache-csi-driver/main/examples/kubernetes/dynamic_provisioning/specs/claim.yaml
```

7. (Optional) Edit the `claim.yaml` file. Change `1200Gi` to an incremental value of **1.2 TiB**
```shell
storage: 1200Gi
```

8. Create the persistent volume claim.
```shell
kubectl apply -f claim.yaml
```

The example output is as follows.
```shell
persistentvolumeclaim/fc-claim created
```

9. Confirm that the file cache is provisioned.

```shell
kubectl describe pvc
```

The example output is as follows.
```shell
Name:          fc-claim
Namespace:     default
StorageClass:  fc-sc
Status:        Bound
```

***Note:** The `Status` may show as `Pending` for 5-10 minutes, before changing to `Bound`. Don't continue with the next step until the `Status` is `Bound`. If the `Status` shows `Pending` for more than 10 minutes, use warning messages in the `Events` as reference for addressing any problems.*

10. Deploy the sample application.
```shell
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/aws-file-cache-csi-driver/main/examples/kubernetes/dynamic_provisioning/specs/pod.yaml
```

11. Verify that the sample application is running.
```shell
kubectl get pods
```
The example output is as follows.
```shell
NAME      READY   STATUS              RESTARTS   AGE
fc-app    1/1     Running             0          9m38s
```

12. Verify that the file cache is mounted correctly by the application.
```shell
kubectl exec -ti fc-app -- df -h
```
The example output is as follows.
```shell
Filesystem                     Size  Used Avail Use% Mounted on
overlay                         80G  4.0G   77G   5% /
tmpfs                           64M     0   64M   0% /dev
tmpfs                          3.8G     0  3.8G   0% /sys/fs/cgroup
192.168.100.210@tcp:/d4v2dbev  1.2T   11M  1.2T   1% /data
/dev/nvme0n1p1                  80G  4.0G   77G   5% /etc/hosts
shm                             64M     0   64M   0% /dev/shm
tmpfs                          7.0G   12K  7.0G   1% /run/secrets/kubernetes.io/serviceaccount
tmpfs                          3.8G     0  3.8G   0% /proc/acpi
tmpfs                          3.8G     0  3.8G   0% /sys/firmware
```

13. Verify that data was written to the Amazon File Cache by the sample app.
```shell
kubectl exec -it fc-app -- ls /data
```
The example output is as follows.
```shell
out.txt
```
This example output shows that the sample app successfully wrote the out.txt file to the file cache.

***Note:**
Before deleting the cluster, make sure to delete the Amazon File Cache. For more information, see [Clean up resources](https://docs.aws.amazon.com/fsx/latest/FileCacheGuide/getting-started-step4.html) in the Amazon File Cache User Guide.*

### Examples
Before the example, you need to:
* Get yourself familiar with how to setup Kubernetes on AWS and [create Amazon File Cache](https://docs.aws.amazon.com/fsx/latest/FileCacheGuide/getting-started.html) if you are using static provisioning.
* When creating Amazon File Cache, make sure its VPC is accessible from Kuberenetes cluster's VPC and network traffic is allowed by security group.
    * For Amazon File Cache VPC, you can either create an Amazon File Cache inside the same VPC as Kubernetes cluster or using VPC peering.
    * For security group, make sure port 988 is allowed for the security groups that are attached the file cache ENI.
* Install Amazon File Cache CSI driver following the [Installation](README.md#Installation) steps.

#### Example Links
* [Static provisioning](../examples/kubernetes/static_provisioning/README.md)
* [Dynamic provisioning](../examples/kubernetes/dynamic_provisioning/README.md)
* [Accessing the file cache from multiple pods](../examples/kubernetes/multiple_pods/README.md)

## Development

Please go through [CSI Spec](https://github.com/container-storage-interface/spec/blob/master/spec.md) and [General CSI driver development guideline](https://kubernetes-csi.github.io/docs/Development.html) to get some basic understanding of CSI driver before you start.

### Requirements
* Golang 1.19.0+

### Dependency
Dependencies are managed through go module. To build the project, first turn on go mod using `export GO111MODULE=on`, to build the project run: `make`

### Testing
To execute all unit tests, run: `make test`

## License

This library is licensed under the Apache 2.0 License.
