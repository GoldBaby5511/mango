# mango

---

## 概述

站在巨人肩膀，核心部分基于[leaf](https://github.com/name5566/leaf)，一个经典开源游戏服务。但leaf为单进程多模块模型，无法进行分布式部署。本项目即在此基础上做了扩展，实现了多进程分布式部署，业务模型为本人之前做过的一个项目。定义为分布式框架，适用所有分区、分房间类游戏。

---

## 相关组成

* mango:https://github.com/GoldBaby5511/mango.git
* mango-admin:https://github.com/GoldBaby5511/mango-admin.git
* mango-admin-ui:https://github.com/GoldBaby5511/mango-admin-ui.git 

----

## 架构

采用网状拓扑结构，中心center服务负责服务治理，除了日志服之外，其他所有服务两两互联且向center服务注册管理。

* gateway：网关服务(可水平扩展)
* center：中心服务，服务注册、治理
* logger：日志服，日志上报、预警
* config：配置中心，支持携程apollo配置中心或本地json、csv、excel文本配置文件，配置更新实时通知相应服务
* lobby：大厅服务，登录、在线管理等
* room：房间服务，用户匹配等(可水平扩展)
* table：桌子服务，游戏具体逻辑(可水平扩展)
* list：房间列表服务，房间负载均衡、列表查询等
* property：财富服务，用户道具增删改查
* robot：机器人服务，模拟客户端行为、陪玩、压测(可水平扩展)
* daemon：守护服务，执行管理端命令，开启/守护相关服务

---

## 使用

### 脚本

* windows
1. 启动：右键scripts\windows目录下Startup.bat已管理员身份运行，若无权限问题会依次编译并运行各个服务

2. 创建：右键scripts\windows目录下CreateNewService.bat已管理员身份运行，输入新服务名，将会在cmd目录下创建对应新服务目录及模板源文件

3. 清理：右键scripts\windows目录下Cleanup.bat已管理员身份运行，将删除cmd内各服务内生成的中间及log文件
* linux
1. 执行权限检查，转到scripts\linux目录，查看三个脚本是否有执行权限若没有则执行以下命令赋权

```bash
chmod +x Cleanup.sh
chmod +x CreateNewService.sh
chmod +x Shutdown.sh
chmod +x Startup.sh
```

2. 启动：转到scripts\linux目录下执行./Startup.sh ，依次编译并后台运行各个服务

```bash
#执行命令，验证服务是否启动成功
ps -aux

#成功的话会有十个进程，类似如下，若有未启动成功进程，可转入cmd对应服务目录下查看日志
sanfeng   12248 15.3  0.7 1015440 29876 pts/1   Sl   11:41   0:02 ./logger -Type=1 -Id=1
sanfeng   12333  0.1  0.3 906492 11916 pts/1    Sl   11:41   0:00 ./center -Type=2 -Id=50
sanfeng   12417  1.1  0.6 1467784 23432 pts/1   Sl   11:41   0:00 ./config -Type=3 -Id=60
sanfeng   12507 18.4  7.4 1417264 286952 pts/1  Sl   11:41   0:02 ./gateway -Type=4 -Id=100
sanfeng   12593  4.9  0.5 1080816 20128 pts/1   Sl   11:41   0:00 ./lobby -Type=5 -Id=70
sanfeng   12673  2.7  0.4 990388 17028 pts/1    Sl   11:41   0:00 ./list -Type=6 -Id=80
sanfeng   12764  2.5  0.4 1055672 15468 pts/1   Sl   11:41   0:00 ./property -Type=7 -Id=90
sanfeng   12851  1.9  0.5 941548 21116 pts/1    Sl   11:41   0:00 ./table -Type=8 -Id=1000
sanfeng   12942  5.0  0.6 1146420 25668 pts/1   Sl   11:41   0:00 ./room -Type=9 -Id=2000
sanfeng   13025 20.3 10.9 1627632 421024 pts/1  Sl   11:41   0:02 ./robot -Type=10 -Id=3000
```

3. 关闭：转到scripts\linux目录下执行./Shutdown.sh
4. 创建：转到scripts\linux目录下执行./CreateNewService.sh，输入名称，会在cmd目录下生成对应服务
5. 清理：转到scripts\linux目录下执行./Cleanup.sh

### 手动编译

windows下可能存在权限问题，导致脚本运行失败，若出现该类情况则可手动编译运行。

1. 启动命令行或shell分别进入cmd下各个服务内，执行 go build
2. 拷贝configs目录至cmd\config目录下，因为config(配置中心服)需要加载各个服务配置
3. 命令行或shell进入cmd下各个服务内，执行以下命令启动服务

```bash
.\logger -Type=1 -Id=1
.\center -Type=2 -Id=50
.\config -Type=3 -Id=55
.\gateway -Type=4 -Id=100
.\lobby -Type=5 -Id=60
.\property -Type=6 -Id=65
.\list -Type=10 -Id=70
.\table -Type=11 -Id=1000
.\room -Type=12 -Id=2000
.\robot -Type=9 -Id=3000
.\daemon -Type=100 -Id=300
```

服务启动完成后，robot会默认创建1000用户模拟客户端行为，连接网关-->登录-->报名-->举手-->游戏。起始用户数量可配，robot-3000.json 文件 "机器人数量"字段

---

## Docker部署

* 本机部署好Docker环境，命令行执行以下命令，生成镜像

```bash
docker build --file ./build/package/Dockerfile.center --tag mango/center .
docker build --file ./build/package/Dockerfile.config --tag mango/config .
docker build --file ./build/package/Dockerfile.gateway --tag mango/gateway .
docker build --file ./build/package/Dockerfile.logger --tag mango/logger .
docker build --file ./build/package/Dockerfile.lobby --tag mango/lobby .
docker build --file ./build/package/Dockerfile.list --tag mango/list .
docker build --file ./build/package/Dockerfile.property --tag mango/property .
docker build --file ./build/package/Dockerfile.table --tag mango/table .
docker build --file ./build/package/Dockerfile.room --tag mango/room .
docker build --file ./build/package/Dockerfile.robot --tag mango/robot .
```

* 创建网桥

```bash
docker network create mango
```

* 执行以下命令，运行本地容器

```bash
docker run -d --name="logger" --network mango mango/logger
docker run -d --name="center" --network mango mango/center
docker run -d --name="config" --network mango mango/config
docker run -d --name="lobby" --network mango mango/lobby
docker run -d --name="list" --network mango mango/list
docker run -d --name="property" --network mango mango/property
docker run -d --name="table" --network mango mango/table
docker run -d --name="room" --network mango mango/room
docker run -d --name="robot" --network mango mango/robot
docker run -d -p 10100:10100 --name="gateway" --network mango mango/gateway
```

---

## 将来

1. 日志服对分片文本文件自动压缩；具备kafka上报，方便接入ELK、信息统计、消息预警等
2. 服务治理，对除网关之外的服务实现热插拔式切换更新
3. 管理工具，服务启动、监控守护、更新、切换等

最终目的不仅是一套完整的服务框架，同时可以将是某些特定业务直接的解决方案。

---

## 相关博客

mango(一)：杂谈项目由来：https://blog.csdn.net/weixin_42780662/article/details/122006434

mango(二)：架构：https://blog.csdn.net/weixin_42780662/article/details/122172058

---

## 参考引用

* leaf：https://github.com/name5566/leaf.git
* agollo：https://github.com/apolloconfig/agollo.git
* fsnotify：https://github.com/fsnotify/fsnotify.git
* proto：https://github.com/protocolbuffers/protobuf.git
* project-layout：https://github.com/golang-standards/project-layout.git

---

## 交流群

* QQ交流群：781335145
