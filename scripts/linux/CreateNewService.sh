#!/bin/sh

echo -n "请输入新服务名,appName="
read appName

cd ../..
cp -r scripts/template cmd
mv cmd/template cmd/$appName
cd cmd/$appName
mkdir business
mv business.txt business/business.go
mv main.txt main.go
sed -i "s/template/$appName/g" `grep "template" -rl ./`
echo "新服务 $appName 创建完成"