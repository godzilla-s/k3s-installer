# K3S-Installer 

k3s-installer是一个简单部署k3s集群的工具，用于搭建或者部署一个云基础平台。

## 使用

k3s-installer使用起来非常简单

### 安装
安装（部署）:
```shell
./k3s-install install -f example/config.yaml
```

卸载:
```shell
./k3s-install uninstall -f example/config.yaml
```


## 配置

安装配置比较简单明了，其分为： 

+ 全局配置
+ 预安装的包
+ 预加载的镜像
+ chart包
+ 节点定义 
