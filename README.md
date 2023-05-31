# kube-ovn-operator

## 1.init

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
