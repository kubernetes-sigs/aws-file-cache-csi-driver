## Multiple Pods Read Write Many
This example shows how to create a static provisioned Amazon File Cache PV and access it from multiple pods with RWX access mode.

### Edit Persistent Volume
Edit persistent volume using sample [spec](./spec/pv.yaml):
```sh
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
  persistentVolumeReclaimPolicy: Recycle
  storageClassName: fc-sc
  csi:
    driver: file.cache.csi.aws.com
    volumeHandle: [FileCacheId]
    volumeAttributes:
      dnsname: [DNSName] 
```
Replace `volumeHandle` with `FileCacheId` and `dnsname` with `DNSName`. Note that the access mode is `RWX` which means the PV can be read and write from multiple pods.

You can get both `FileCacheId` and `DNSName` using AWS CLI:

```sh
aws fsx describe-file-caches
```

### Deploy the Application
Create PV, persistence volume claim (PVC), storageclass and the pods that consume the PV:
```sh
kubectl apply -f examples/kubernetes/multiple_pods/specs/storageclass.yaml
kubectl apply -f examples/kubernetes/multiple_pods/specs/pv.yaml
kubectl apply -f examples/kubernetes/multiple_pods/specs/claim.yaml
kubectl apply -f examples/kubernetes/multiple_pods/specs/pod1.yaml
kubectl apply -f examples/kubernetes/multiple_pods/specs/pod2.yaml
```

Both pod1 and pod2 are writing to the same Amazon File Cache at the same time.

### Check the Application uses Amazon File Cache
After the objects are created, verify that pod is running:

```
kubectl get pods
```

Also verify that data is written onto Amazon File Cache:

```
kubectl exec -ti app1 -- tail -f /data/out1.txt
kubectl exec -ti app2 -- tail -f /data/out2.txt
```
