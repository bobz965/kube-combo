# kube-ovn-operator

该项目用于在 kube-ovn-cni 的一个**补充**，用于实现一些 kube-ovn-cni 中属于间接关联的一些网络应用。直接关联的网络功能应直接在 kube-ovn 中实现。
为了保持该项目的定位的清晰和轻量：

- 该项目最多会对 kube-ovn crd 的存在与否做一些 get 校验，目前不会 CRUD kube-ovn crd 资源。
- 该项目只会实现 kube-ovn 中所不直接具备的 crd，不会提供关于某个用户场景的需求的多个 CRD 的再次封装为一个新的业务网络功能的CRD。
- 在使用上，业务需求方负责对基础 CRD API 接口进行编排，需直接对接 kube-ovn的crd，或者，该项目提供的 crd。

## 1. Code init

``` bash

operator-sdk init --domain kube-ovn-operator.com --repo github.com/bobz965/kube-ovn-operator --plugins=go/v4-alpha

# we'll use a domain of kube-ovn-operator.com
# so all API groups will be <group>.kube-ovn-operator.com

# --plugins=go/v4-alpha  mac arm 芯片需要指定

# 该步骤后可创建 api
# operator-sdk create api
operator-sdk create api --group vpn-gw --version v1 --kind VpnGw --resource --controller

#  make generate   生成controller 相关的 informer clientset 等代码
 
## 下一步就是编写crd
## 重新生成代码
## 编写 reconcile 逻辑

### 最后就是生成部署文件
make manifests

```

## 2. 设计

公网访问方式

- fip
- router lb （后续的 ha 方案）

### 2.1 ssl vpn

该功能基于 openvpn 实现，可以通过公网 ip，在个人 电脑，手机客户端直接访问 kube-ovn 自定义 vpc subnet 内部的 pod 以及 switch lb 对应是的 svc endpoint。

### 2.2 ipsec vpn 设计

## 3. 维护

基于 olm 来维护， olm 也叫 operator 生命周期管理器， 可以对接到应用商店 kubeapp 。

### 3.1 项目打包

Docker

``` bash
make docker-build 

make docker-push
```

OLM

``` bash
# make bundle bundle-build bundle-push
make bundle
make bundle-build
make bundle-push


## 目前不支持直接测试，必须要先把bundle 传到 registry，有issue记录: https://github.com/operator-framework/operator-sdk/issues/6432


```


### 3.2 基于 olm 部署

[operator-sdk 二进制安装方式](https://sdk.operatorframework.io/docs/installation/)

```bash
# 在 k8s集群安装该项目
operator-sdk olm install

## ref https://github.com/operator-framework/operator-lifecycle-manager/releases/tag/v0.24.0

curl -L https://github.com/operator-framework/operator-lifecycle-manager/releases/download/v0.24.0/install.sh -o install.sh
chmod +x install.sh
./install.sh v0.24.0


# 运行 operator

operator-sdk run bundle registry.cn-hangzhou.aliyuncs.com/bobz/kube-ovn-operator-bundle:v0.0.1

# 检查 operator 已安装

kubectl get csv



## 基于 kubectl apply 运行一个该 operator 维护的 crd

# 清理该 operator
k get operator

operator-sdk cleanup vpn-gw

```





