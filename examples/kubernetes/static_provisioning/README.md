## Static Provisioning Example

---

This example shows how to make a pre-created Amazon File Cache mounted inside container.

### Edit [Persistent Volume Spec](specs/pv.yaml)
```
apiVersion: v1
kind: PersistentVolume
metadata:
  name: fc-pv
spec:
  capacity:
    storage: 1200Gi
  volumeMode: Filesystem
  accessModes:
    - ReadWriteMany
  mountOptions:
    - flock
  persistentVolumeReclaimPolicy: Recycle
  csi:
    driver: filecache.csi.aws.com
    volumeHandle: [FileCacheId]
    volumeAttributes:
      dnsname: [DNSName] 
      mountname: [MountName]
```
Replace `volumeHandle` with `FileCacheId`, `dnsname` with `DNSName` and `mountname` with `MountName`. You can get both `FileCacheId`, `DNSName` and `MountName` using AWS CLI:

```sh
>> aws fsx describe-file-caches
```

### Deploy the Application
Create PV, persistent volume claim (PVC), and the pod that consumes the PV:
```sh
>> kubectl apply -f examples/kubernetes/static_provisioning/specs/pv.yaml
>> kubectl apply -f examples/kubernetes/static_provisioning/specs/claim.yaml
>> kubectl apply -f examples/kubernetes/static_provisioning/specs/pod.yaml
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
