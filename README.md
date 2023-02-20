[![Build Status](https://travis-ci.org/kubernetes-sigs/aws-file-cache-csi-driver.svg?branch=master)](https://travis-ci.org/kubernetes-sigs/aws-fsx-csi-driver)
[![Coverage Status](https://coveralls.io/repos/github/kubernetes-sigs/aws-file-cache-csi-driver/badge.svg?branch=master)](https://coveralls.io/github/kubernetes-sigs/aws-file-cache-csi-driver?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubernetes-sigs/aws-file-cache-csi-driver)](https://goreportcard.com/report/github.com/kubernetes-sigs/aws-file-cache-csi-driver)

**WARNING**: This driver is in pre ALPHA currently. This means that there may potentially be backwards compatible breaking changes moving forward. Do NOT use this driver in a production environment in its current state.

**DISCLAIMER**: This is not an officially supported Amazon product

## Amazon File Cache CSI Driver
### Overview

The [Amazon File Cache]() Container Storage Interface (CSI) Driver provides a [CSI]() interface used by container orchestrators to manage the lifecycle of Amazon file cache volumes.

This driver is in alpha stage. Basic volume operations that are functional include NodePublishVolume/NodeUnpublishVolume.

### CSI Specification Compability Matrix
| AWS File Cache CSI Driver \ CSI Version           | v1.0.0|
|---------------------------------------------------|-------|
| master branch                                     | yes   |

### Features
Currently only static provisioning is supported. With static provisioning, a file cache should be created manually, then it could be mounted inside container as a persistence volume (PV) using File Cache CSI Driver.

The following CSI interfaces are implemented:
* Controller Service: 
* Node Service: NodePublishVolume, NodeUnpublishVolume, NodeGetCapabilities, NodeGetInfo, NodeGetId
* Identity Service: GetPluginInfo, GetPluginCapabilities, Probe

## Amazon File Cache CSI Driver on Kubernetes

---------
The following sections are Kubernetes-specific. If you are a Kubernetes user, use the following for driver features, installation steps and examples.

### Kubernetes Version Compability Matrix
| AWS File Cache CSI Driver \ Kubernetes Version    | v1.24+ |
|---------------------------------------------------|--------|
| master branch                                     | yes    |

### Container Images
| File Cache CSI Driver Version | Image                                                         |
|-------------------------------|---------------------------------------------------------------|
| master branch                 | public.ecr.aws/fsx-csi-driver/aws-filecache-csi-driver:latest |

### Features
* Static provisioning - Amazon File Cache needs to be created manually first, then it could be mounted inside container as a volume using the Driver.
* Dynamic provisioning (currently not supported) - uses persistent volume claim (PVC) to let Kubernetes create the Amazon File Cache for you and consumes the volume from inside container.
* Mount options - mount options can be specified in storageclass to define how the volume should be mounted.

**Notes**:
* For dynamically provisioned volumes, only one subnet is allowed inside a storageclass's `parameters.subnetId`. This is a [limitation](https://docs.aws.amazon.com/fsx/latest/APIReference/API_FileCacheCreating.html#FSx-Type-FileCacheCreating-SubnetIds) that is enforced by Amazon File Cache.

### Installation
#### Set up driver permission
The driver requires IAM permission to talk to Amazon File Cache service to create/delete the filecache on user's behalf. There are several methods to grant driver IAM permission:
* Using secret object - create an IAM user with proper permission, put that user's credentials in [secret manifest](../deploy/kubernetes/secret.yaml) then deploy the secret.

```sh
curl https://raw.githubusercontent.com/kubernetes-sigs/aws-file-cache-csi-driver/master/deploy/kubernetes/secret.yaml > secret.yaml
# Edit the secret with user credentials
kubectl apply -f secret.yaml
```

* Using worker node instance profile - grant all the worker nodes with proper permission by attach policy to the instance profile of the worker.
```sh
`kubectl annotate serviceaccount -n kube-system file-cache-csi-controller-sa \
 eks.amazonaws.com/role-arn=arn:aws:iam::111111111111:role/AmazonEKSFileCacheCSIDriverFullAccess --overwrite=true
```


#### Deploy driver
```sh
kubectl apply -k deploy/kubernetes/base/
```

TODO: Add helm installation option
```sh

```





------------------


### Examples
Before the example, you need to:
* Get yourself familiar with how to setup Kubernetes on AWS and [create Anmazon File Cache](https://docs.aws.amazon.com/fsx/latest/FileCacheGuide/getting-started.html) if you are using static provisioning.
* When creating Amazon File Cache, make sure its VPC is accessible from Kuberenetes cluster's VPC and network traffic is allowed by security group.
    * For FSx for Lustre VPC, you can either create an Amazon File Cache inside the same VPC as Kubernetes cluster or using VPC peering.
    * For security group, make sure port 988 is allowed for the security groups that are attached the lustre filesystem ENI.
* Install Amazon File Cache CSI driver following the [Installation](README.md#Installation) steps.

#### Example Links
* [Static provisioning](examples/kubernetes/static_provisioning/README.md)
* [Dynamic provisioning](examples/kubernetes/dynamic_provisioning/README.md)
* [Accessing the filesystem from multiple pods](examples/kubernetes/multiple_pods/README.md)

## Development

----
Please go through [CSI Spec](https://github.com/container-storage-interface/spec/blob/master/spec.md) and [General CSI driver development guideline](https://kubernetes-csi.github.io/docs/Development.html) to get some basic understanding of CSI driver before you start.

### Requirements
* Golang 1.19.0+

### Dependency
Dependencies are managed through go module. To build the project, first turn on go mod using `export GO111MODULE=on`, to build the project run: `make`

### Testing
To execute all unit tests, run: `make test`

## License

----
This library is licensed under the Apache 2.0 License. 

