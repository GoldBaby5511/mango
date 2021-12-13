# xlddz

## 概述

站在巨人肩膀，核心部分基于[leaf](https://github.com/name5566/leaf)，一个经典开源游戏服务。但leaf为单进程多模块模型，无法进行分布式部署。本项目即在此基础上做了扩展，实现了多进程分布式部署，业务模型为本人之前做过的一个项目，兴隆斗地主。定义为分布式框架，适用所有分区、分房间类游戏。

----

## 架构

采用网状拓扑结构，中心center服务负责服务治理，除了日志服之外，其他所有服务两两互联且向center服务注册管理。

* gateway：网关服务
* center：中心服务
* logger：日志服
* config：配置中心，支持协程apollo配置中心 或本地json文本配置文件，配置更新实时通知相应服务
* login：登录服务
* 其他业务服务：比如 list、fund、room、table... ...(暂未实现)

---

## 用法

* windows下执行statup.bat，依次编译并启动logger.exe、center.exe、config.exe、gateway.exe、login.exe

* 解压client，双击执行client.exe，单击“微信登录”，选择Test001

---

## Docker部署

* 本机部署好Docker环境，命令行下依次执行一下命令，生成镜像

```bash
docker build --target center --file ./servers/center/Dockerfile --tag xlddz/center .
docker build --target config --file ./servers/config/Dockerfile --tag xlddz/config .
docker build --target gateway --file ./servers/gateway/Dockerfile --tag xlddz/gateway .
docker build --target logger --file ./servers/logger/Dockerfile --tag xlddz/logger .
docker build --target login --file ./servers/login/Dockerfile --tag xlddz/login .
```

* 创建网桥

```bash
docker network create xlddz
```

* 执行以下命令，运行本地容器

```bash
docker run -d --name="logger" --network xlddz xlddz/logger
docker run -d --name="center" --network xlddz xlddz/center
docker run -d --name="config" --network xlddz xlddz/config
docker run -d --name="login" --network xlddz xlddz/login
docker run -d -p 10102:10102 --name="gateway" --network xlddz xlddz/gateway
```

---

## 服务热切换

以login服务为例，服务在router内的注册是以 appType+appId 为唯一标识，所以若要升级login只需要新开一个实例，使用 appType+新appId，然后修改配置文件(common-server-router.json)或 apollo内“服务维护”字段，写入原login的appType+appId，配置中心会实时将配置通知router，之后的登录消息将会被路由到新login。

注意：

* 切换完成后一定要删除“服务维护”内配置，否则配置内的服务重新启动后将无法正常工作
* 目前该方法较为简单粗糙，适用于一些无状态的服务，若服务内有状态将会带来一定状态损失(待优化)

---

## 将来

1. 完善剩余服务 list、fund、room、table使之成为一套完整架构
2. 日志服对分片文本文件自动压缩；具备kafka上报，方便接入ELK、信息统计、消息预警等
3. 服务治理，对除网关之外的服务实现热插拔式切换更新
4. 管理工具，服务启动、监控守护、更新、切换等

最终目的不仅是一套完整的服务框架，同时可以将是某些特定业务直接的解决方案。

---

## 参考引用

* leaf：https://github.com/name5566/leaf.git
* agollo：https://github.com/apolloconfig/agollo.git
* fsnotify：https://github.com/fsnotify/fsnotify.git
* proto：https://github.com/protocolbuffers/protobuf.git

---

## 交流群

* QQ群：781335145



