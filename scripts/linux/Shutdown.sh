#!/bin/sh

KillApp(){
	echo "查找 $1 "
    pid=$(ps -ef|grep ./$1|grep Type=|grep -v grep|awk '{print $2}')
    echo $pid
    kill -9 $pid
}

KillApp logger
KillApp center
KillApp config
KillApp gateway
KillApp lobby
KillApp list
KillApp property
KillApp table
KillApp room
KillApp robot
