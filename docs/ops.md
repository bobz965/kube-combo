# 3. 维护

基于 operator 生命周期管理器（olm）来维护， 可以对接到应用商店 kubeapp。

## 3.1 项目打包

Docker

``` bash
make docker-build docker-push

# make docker-build 
# make docker-push


# build openvpn image

make docker-build-ssl-vpn docker-push-ssl-vpn

make docker-build-ipsec-vpn docker-push-ipsec-vpn

```

OLM

``` bash
make bundle bundle-build bundle-push

# make bundle
# make bundle-build
# make bundle-push


## 目前不支持直接测试，必须要先把bundle 传到 registry，有issue记录: https://github.com/operator-framework/operator-sdk/issues/6432


```

### 3.2  部署

目前认为 olm 本身不够成熟，基于 `make deploy` 来部署

``` bash

cd config/manager && /root/kube-combo/bin/kustomize edit set image controller=registry.cn-hangzhou.aliyuncs.com/bobz/kube-combo:latest
/root/kube-combo/bin/kustomize build config/default | kubectl apply -f -


```

[operator-sdk 二进制安装方式](https://sdk.operatorframework.io/docs/installation/)

```bash
# 在 k8s集群安装该项目
operator-sdk olm install

## ref https://github.com/operator-framework/operator-lifecycle-manager/releases/tag/v0.24.0

curl -L https://github.com/operator-framework/operator-lifecycle-manager/releases/download/v0.24.0/install.sh -o install.sh
chmod +x install.sh
./install.sh v0.24.0


# 运行 operator

operator-sdk run bundle registry.cn-hangzhou.aliyuncs.com/bobz/kube-combo-bundle:v0.0.1

# 检查 operator 已安装

kubectl get csv



## 基于 kubectl apply 运行一个该 operator 维护的 crd

# 清理该 operator
k get operator

operator-sdk cleanup vpn-gw

```

### 4. certmanager

``` bash
operator-sdk olm install

# 功能上 operator-sdk == kubectl operator 

kubectl krew install operator
kubectl create ns cert-manager
kubectl operator install cert-manager -n cert-manager --channel candidate --approval Automatic --create-operator-group 

# kubectl operator install cert-manager -n operators --channel stable --approval Automatic

kubectl get events -w -n operators

kubectl operator list
kubectl operator uninstall cert-manager -n cert-manager

# 目前 基于operator 安装的版本普遍较旧，差了一个大版本，可能要跟下 operator 的维护策略
# 目前认为最好是基于 kubectl apply 安装最新的

kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.12.1/cert-manager.yaml
kubectl get pods -n cert-manager
kubectl get crd | grep cert-manager.io

# 清理: https://cert-manager.io/docs/installation/kubectl/


```

### 5. 通用性

目前 vpn gw pod 只需要一个 IP，所以只需要保证固定内网 IP 是符合 k8s 通用规范即可保证能够适用于其他 CNI。
该 IP 对应的 nat， 以及如何公网互联的路由和该功能是完全解耦的。

各大公有云都是 sdn 网络，支持在 k8s 托管， 基于该 vpn gw operator 互相打通，比起申请虚拟机资源部署的方式应该更节省成本。

### 6. 参考

- [一个简单的 ipsec vpn 在公有云部署的项目就可以有 23k 的 star](https://github.com/hwdsl2/setup-ipsec-vpn/blob/master/README-zh.md#%E4%B8%8B%E4%B8%80%E6%AD%A5)
