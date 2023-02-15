## Dynamic Provisioning Example
*~Not yet supported~*

This example shows how to create an Amazon File Cache using persistence volume claim (PVC) and consumes it from a pod.


### Edit [StorageClass](./specs/storageclass.yaml)
```sh
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: fc-sc
provisioner: file.cache.csi.aws.com
parameters:
  subnetId: subnet-0eabfaa81fb22bcaf
  securityGroupIds: sg-068000ccf82dfba88
  dataRepositoryAssociations: "fileCachePath=/ns1/,dataRepositoryPath=nfs://10.0.92.69/fsx/"
  fileCacheType: "LUSTRE"
  fileCacheTypeVersion: "2.12"
  LustreConfiguration: "{DeploymentType=CACHE_1,MetadataConfiguration=2400,perUnitStorageThroughput=1000}"
  weeklyMaintenanceStartTime: "6:00:00"
  extraTags: "Tag1=Value1,Tag2=Value2"
```
* subnetId - The subnet ID that the Amazon File Cache should be created inside.
* securityGroupIds - A comma separated list of security group IDs that should be attached to the filecache
* dataRepositoryAssociations - A list of IDs of data repository associations that are associated with this cache.
* fileCacheType (Optional) - The type of cache, which must be LUSTRE.
* fileCacheTypeVersion (Optional) - The Lustre version of the cache, which must be 2.12.
* LustreConfiguration (Optional) - The configuration for the Amazon File Cache resource, please view [FileCacheLustreConfiguration](https://docs.aws.amazon.com/fsx/latest/APIReference/API_FileCacheLustreConfiguration.html) for more details on the contents.
* weeklyMaintenanceStartTime (Optional) - The preferred start time to perform weekly maintenance, formatted d:HH:MM in the UTC time zone, where d is the weekday number, from 1 through 7, beginning with Monday and ending with Sunday. The default value is "7:09:00" (Sunday 09:00 UTC)
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
Update `spec.resource.requests.storage` with the storage capacity to request. The storage capacity value will be rounded up to 1200 GiB, 2400 GiB, or a multiple of 3600 GiB for SSD. If the storageType is specified as HDD, the storage capacity will be rounded up to 6000 GiB or a multiple of 6000 GiB if the perUnitStorageThroughput is 12, or rounded up to 1800 or a multiple of 1800 if the perUnitStorageThroughput is 40.

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
>> kubectl exec -ti fsx-app -- tail -f /data/out.txt
```