## Dynamic Provisioning Example


This example shows how to create an Amazon File Cache using persistence volume claim (PVC) and consumes it from a pod. Please see the [CreateFileCache API Reference](https://docs.aws.amazon.com/fsx/latest/APIReference/API_CreateFileCache.html#FSx-CreateFileCache-request-DataRepositoryAssociations) for more information.


### Edit [StorageClass](specs/storageclass.yaml)
```sh
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: fc-sc
provisioner: filecache.csi.aws.com
parameters:
  subnetId: [SubnetId]
  securityGroupIds: [SecurityGroupId]
  dataRepositoryAssociations: [DataRepositoryAssociations]
  kmsKeyId: [KmsKeyId]
  copyTagsToDataRepositoryAssociations: "false"
  fileCacheType: "LUSTRE"
  fileCacheTypeVersion: "2.12"
  LustreConfiguration: "DeploymentType=CACHE_1,PerUnitStorageThroughput=1000,MetadataConfiguration={StorageCapacity=2400}"
  weeklyMaintenanceStartTime: "d:HH:MM"
  extraTags: "Tag1=Value1,Tag2=Value2"
```
*Update the parameters not marked as optional below.*

* subnetId - The subnet ID that the Amazon File Cache should be created inside.
* securityGroupIds - A comma separated list of security group IDs that should be attached to the file cache.
* dataRepositoryAssociations - A space separated ist of up to 8 configurations for data repository associations (DRAs) to be created during the cache creation. The DRAs link the cache to either an Amazon S3 data repository or a Network File System (NFS) data repository that supports the NFSv3 protocol. Please see [DataRepositoryAssociations](https://docs.aws.amazon.com/fsx/latest/APIReference/API_CreateFileCache.html#FSx-CreateFileCache-request-DataRepositoryAssociations) for the File Cache DRA configurations requirements.
* copyTagsToDataRepositoryAssociations - A boolean flag indicating whether tags for the cache should be copied to data repository associations. This value defaults to false.
* fileCacheType (Optional) - The type of cache, which must be LUSTRE.
* fileCacheTypeVersion (Optional) - The Lustre version of the cache, which must be 2.12.
* weeklyMaintenanceStartTime (Optional) - The preferred start time to perform weekly maintenance, formatted d:HH:MM in the UTC time zone, where d is the weekday number, from 1 through 7, beginning with Monday and ending with Sunday. The default value is "7:09:00" (Sunday 09:00 UTC)
* kmsKeyId (Optional) - Specifies the ID of the Key Management Service (KMS) key to use for encrypting data on an Amazon File Cache. If a KmsKeyId isn't specified, the Amazon FSx-managed KMS key for your account is used.
* LustreConfiguration (Optional) - The configuration for the Amazon File Cache resource, please view [FileCacheLustreConfiguration](https://docs.aws.amazon.com/fsx/latest/APIReference/API_FileCacheLustreConfiguration.html) for more details on the contents.
  * DeploymentType - Specifies the cache deployment type, which must be CACHE_1
  * PerUnitStorageThroughput - Provisions the amount of read and write throughput for each 1 tebibyte (TiB) of cache storage capacity, in MB/s/TiB. The only supported value is 1000.
  * MetadataConfiguration - The configuration for a Lustre MDT (Metadata Target) storage volume. The storage capacity of the Lustre MDT (Metadata Target) storage volume in gibibytes (GiB). The only supported value is 2400 GiB.
* extraTags (Optional) - Tags that will be set on the FSx resource created in AWS, in the form of a comma separated list with each tag delimited by an equals sign (example - "Tag1=Value1,Tag2=Value2") . Default is a single tag with CSIVolumeName as the key and the generated volume name as it's value.

### Edit [Persistent Volume Claim Spec](./specs/claim.yaml)
```sh
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: fc-claim
spec:
  accessModes:
    - ReadWriteMany
  storageClassName: fc-sc
  resources:
    requests:
      storage: 1200Gi
```
*Update `spec.resource.requests.storage` with the storage capacity to request. The storage capacity value will be rounded up to 1200 GiB, 2400 GiB, or a multiple of 2400 GiB.*

### Deploy the Application
Create PVC, storageclass and the pod that consumes the PV:
```sh
>> kubectl apply -f examples/kubernetes/dynamic_provisioning/specs/storageclass.yaml
>> kubectl apply -f examples/kubernetes/dynamic_provisioning/specs/claim.yaml
>> kubectl apply -f examples/kubernetes/dynamic_provisioning/specs/pod.yaml
```

### Check the Application uses Amazon File Cache
After the objects are created, verify that pod is running:

```sh
>> kubectl get pods
```

Also verify that data is written onto Amazon File Cache:

```sh
>> kubectl exec -ti fc-app -- tail -f /data/out.txt
```
