# sample-device-plugin

## 编译二进制

```
make go.build
```

## 编译镜像

```
make image
```

## 部署

```
# kubectl label nodes xiabingyao-lc0 sample-device=enable
# kubectl apply -f build/deploy/manifest/sample-device-plugin.yaml
```

## 查看

```
# kubectl describe nodes master-01
Allocated resources:
  (Total limits may be over 100 percent, i.e., overcommitted.)
  Resource           Requests    Limits
  --------           --------    ------
  cpu                1090m (2%)  16 (33%)
  memory             560Mi (0%)  8532Mi (8%)
  ephemeral-storage  0 (0%)      0 (0%)
  hugepages-1Gi      0 (0%)      0 (0%)
  hugepages-2Mi      0 (0%)      0 (0%)
  sample.com/sample  0           0
```
出现sample.com/sample资源

## 测试
```
# cat  build/deploy/sample/pod-use-sample-resource.yaml
apiVersion: v1
kind: Pod
metadata:
  name: sample-pod
  labels:
    name: sample-pod
spec:
  containers:
    - name: sample-pod
      image: nginx:1.14.2
      resources:
        requests:
          memory: "50Mi"
          cpu: "50m"
          sample.com/sample: 1
        limits:
          memory: "150Mi"
          cpu: "500m"
          sample.com/sample: 1
      ports:
        - containerPort: 80

# kubectl apply -f build/deploy/sample/pod-use-sample-resource.yaml
```