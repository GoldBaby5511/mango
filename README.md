# xlddz

## 概述

站在巨人肩膀，核心部分基于[leaf](https://github.com/name5566/leaf)，一个经典开源游戏服务。但原作为单进程多模块模型，无法进行分布式部署。本项目即在此基础上做了扩展，实现了多进程分布式部署，业务模型为本人之前做过的一个项目，兴隆斗地主。定义为分布式棋牌框架，但适用所有分区、分房间类游戏。

----

## 架构

采用星状拓扑结构，存在一个中心router服务，除了日志服之外其他所有服务都会连接router，所以整个架构内存在四类服务。

* gateway：网关服务，特点，既对外同时连接router
* router：中心路由服务，特点，只对外
* log：日志服
* config：配置中心，支持协程apollo配置中心 或本地json文本配置文件，配置更新实时通知相应服务
* login：登录服务
* 其他业务服务：比如 list、fund、room、table... ...(暂未实现)

---

## 用法

* xlddz目录下打开命令行切换到logger目录，执行 go build 生成 logger.exe 启动

```bash
cd .\servers\logger\
go build
.\logger.exe
```

* xlddz目录下打开命令行切换到config目录，执行 go build 生成 config.exe 启动

```bash
cd .\servers\config\
go build
.\config.exe
```

* xlddz目录下打开命令行切换到router目录，执行 go build 生成 router.exe 启动

```bash
cd .\servers\router\
go build
.\router.exe
```

* xlddz目录下打开命令行切换到login目录，执行 go build 生成 login.exe 启动

```bash
cd .\servers\login\
go build
.\login.exe
```

* xlddz目录下打开命令行切换到gateway目录，执行 go build 生成 gateway.exe 启动

```bash
cd .\servers\gateway\
go build
.\gateway.exe
```

* 解压client，双击执行client.exe，单击“微信登录”，选择Test001

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



