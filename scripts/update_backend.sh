#!/bin/bash

unzipLcrtu(){
  cd /mnt/mmc/tmp
  rm -rf /mnt/mmc/tmp/lcrtu
  unzip lcrtu.zip
  if [ ! -f "/mnt/mmc/tmp/lcrtu" ];then
      echo "lcrtu.zip 不存在/mnt/mmc/tmp/lcrtu 可执行文件"
      return 1
    else
      return 0
    fi
}

#更新后端脚本
#1.解压更新包
unzipLcrtu
if [ $? -eq "1" ]; then
  echo "解压lcrtu.zip失败"
  exit 1
fi

#2.停止后端服务
if systemctl stop lcrtu ;then
  echo "停止lcrtu服务成功"
else
  echo "停止lcrtu服务失败"
  exit 1
fi

#3.重命名后端程序包：lcrtu-bak
cd /mnt/mmc/lcrtu
rm -f lcrtu-bak
mv lcrtu lcrtu-bak
#
##4.复制程序
cp /mnt/mmc/tmp/lcrtu /mnt/mmc/lcrtu
#
##5.赋予可执行权限
chmod 777 lcrtu
#
##6.启动服务
systemctl start lcrtu

#删除缓存
rm -rf /mnt/mmc/tmp/lcrtu.zip
rm -rf /mnt/mmc/tmp/lcrtu
