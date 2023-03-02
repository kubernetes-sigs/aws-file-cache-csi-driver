[![Go Report Card](https://goreportcard.com/badge/github.com/kubernetes-sigs/aws-file-cache-csi-driver)](https://goreportcard.com/report/github.com/kubernetes-sigs/aws-file-cache-csi-driver)

**WARNING**: This driver is in pre ALPHA currently. This means that there may potentially be backwards compatible breaking changes moving forward. Do NOT use this driver in a production environment in its current state.

**DISCLAIMER**: This is not an officially supported Amazon product

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
#### 1. Set a few variables to use in the remaining steps. Replace `my-csi-file-cache` with the name of the test cluster you want to create and `region-code` with the AWS Region that you want to create your test cluster in.
```shell
export cluster_name=my-csi-file-cache
export region_code=region-code
```

#### 2. Create a test cluster.
```shell
eksctl create cluster \
  --name $cluster_name \
  --region $region_code \
  --with-oidc \
  --ssh-access \
  --ssh-public-key my-key
```

#### 3. Set up driver permission
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

#### 4. Deploy driver
```sh
kubectl apply -k "github.com/kubernetes-sigs/aws-file-cache-csi-driver/deploy/kubernetes/overlays/stable/?ref=release-0.1"
```

Alternatively, you could also install the driver using helm:


TODO: Add helm installation option
```sh

```

### Examples
Before the example, you need to:
* Get yourself familiar with how to setup Kubernetes on AWS and [create Anmazon File Cache](https://docs.aws.amazon.com/fsx/latest/FileCacheGuide/getting-started.html) if you are using static provisioning.
* When creating Amazon File Cache, make sure its VPC is accessible from Kuberenetes cluster's VPC and network traffic is allowed by security group.
    * For Amazon File Cache VPC, you can either create an Amazon File Cache inside the same VPC as Kubernetes cluster or using VPC peering.
    * For security group, make sure port 988 is allowed for the security groups that are attached the file cache ENI.
* Install Amazon File Cache CSI driver following the [Installation](README.md#Installation) steps.

#### Example Links
* [Static provisioning](examples/kubernetes/static_provisioning/README.md)
* [Dynamic provisioning](examples/kubernetes/dynamic_provisioning/README.md)
* [Accessing the file cache from multiple pods](examples/kubernetes/multiple_pods/README.md)

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

