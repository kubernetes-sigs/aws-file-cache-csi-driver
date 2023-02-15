[![Build Status](https://travis-ci.org/aws/aws-fsx-csi-driver.svg?branch=master)](https://travis-ci.org/aws/aws-fsx-csi-driver)

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

### Kubernetes Version Compability Matrix
| AWS File Cache CSI Driver \ Kubernetes Version    | v1.17+ |
|---------------------------------------------------|--------|
| master branch                                     | yes    |

## Features
Currently only static provisioning is supported. With static provisioning, a file cache should be created manually, then it could be mounted inside container as a persistence volume (PV) using File Cache CSI Driver.

The following CSI interfaces are implemented:
* Controller Service: 
* Node Service: NodePublishVolume, NodeUnpublishVolume, NodeGetCapabilities, NodeGetInfo, NodeGetId
* Identity Service: GetPluginInfo, GetPluginCapabilities, Probe

## Examples
This example shows how to make an Amazon File Cache availble inside container for the application to consume. Before this, get yourself familiar with how to setup kubernetes on AWS and [create an Amazon file cache](https://docs.aws.amazon.com/fsx/latest/FileCacheGuide/getting-started.html). And when creating an Amazon File Cache, make sure it is created inside the same VPC as kuberentes cluster or it is accessible through VPC peering.

Once kubernetes cluster and an Amazon File Cache is created, create secret manifest file using [secret.yaml](../deploy/kubernetes/secret.yaml).

Then create the secret object:
```sh
kubectl apply -f deploy/kubernetes/secret.yaml 
```

Deploy the Amazon file cache CSI driver:

```sh
kubectl apply -k deploy/kubernetes/base/
```

Edit the [persistence volume manifest file](../examples/kubernetes/static_provisioning/specs/pv.yaml):
```sh
apiVersion: v1
kind: PersistentVolume
metadata:
  name: fc-pv
spec:
  capacity:
    storage: 1200Gi
  volumeMode: FileCache
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Recycle
  storageClassName: fc-sc
  csi:
    driver: file.cache.csi.aws.com
    volumeHandle: [FileCacheId]
    volumeAttributes:
      dnsname: [DNSName] 
```
Replace `volumeHandle` with `FileCacheId` and `dnsname` with `DNSName`. You can get both `FileCacheId` and `DNSName` using AWS CLI:

```sh
aws fsx describe-file-caches
```

Then create PV, persistence volume claim (PVC) and storage class:
```sh
kubectl apply -f examples/kubernetes/dynamic_provisioning/specs/storageclass.yaml
kubectl apply -f examples/kubernetes/dynamic_provisioning/specs/pv.yaml
kubectl apply -f examples/kubernetes/dynamic_provisioning/specs/claim.yaml
kubectl apply -f examples/kubernetes/dynamic_provisioning/specs/pod.yaml
```

After objects are created, verify that pod is running:

```sh
kubectl get pods
```

Make sure data is written onto Amazon File Cache:

```sh
kubectl exec -ti fc-app -- df -h
kubectl exec -it fc-app -- ls /data
```

## Development
Please go through [CSI Spec](https://github.com/container-storage-interface/spec/blob/master/spec.md) and [General CSI driver development guideline](https://kubernetes-csi.github.io/docs/Development.html) to get some basic understanding of CSI driver before you start.

### Requirements
* Golang 1.9+

### Testing
To execute all unit tests, run: `make test`

## License
This library is licensed under the Apache 2.0 License. 


