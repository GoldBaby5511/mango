#!/bin/sh
find ../../cmd -name *.log -type f -print -exec rm -rf {} \;
find ../../cmd -name *.bat -type f -print -exec rm -rf {} \;
find ../../cmd -name log -type d -print -exec rm -rf {} \;
rm -rf ../../cmd/center/center
rm -rf ../../cmd/config/config
rm -rf ../../cmd/config/configs
rm -rf ../../cmd/gateway/gateway
rm -rf ../../cmd/list/list
rm -rf ../../cmd/logger/logger
rm -rf ../../cmd/login/login
rm -rf ../../cmd/property/property
rm -rf ../../cmd/robot/robot
rm -rf ../../cmd/room/room
rm -rf ../../cmd/table/table