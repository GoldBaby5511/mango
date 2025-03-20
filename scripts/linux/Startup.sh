#!/bin/sh
cd ../..
cp -r configs cmd/config

build(){
	echo "开始编译 $1 "
	cd cmd/$1/
	go build
	echo "编译完成，启动 $1 "
	nohup ./$1 $2 $3 1>log.log 2>err.log &
	cd ../..
}

build logger -Type=1 -Id=1
build center -Type=2 -Id=50
build config -Type=3 -Id=60
build gateway -Type=4 -Id=100
build lobby -Type=5 -Id=70
build list -Type=6 -Id=80
build property -Type=7 -Id=90
build table -Type=8 -Id=1000
build room -Type=9 -Id=2000
build robot -Type=10 -Id=3000
