#!/bin/bash

unzipQtApp(){
  cd /mnt/mmc/tmp
  rm -rf /mnt/mmc/tmp/qtApp
  unzip qtApp.zip
  if [ ! -d "/mnt/mmc/tmp/qtApp/" ];then
    echo "qtApp.zip 不存在/mnt/mmc/tmp/qtApp路径"
    return 1
  else
    if [ ! -f "/mnt/mmc/tmp/qtApp/qtApp" ];then
      echo "qtApp.zip 不存在/mnt/mmc/tmp/qtApp/qtApp可执行文件"
      return 1
    else
      return 0
    fi
  fi
}


APP_NAME=qtApp
pid=`ps -ef|grep $APP_NAME|grep -v grep|awk '{print $2}' `
unzipQtApp
if [ $? -eq "1" ]; then
  echo "解压qtApp.zip失败"
  exit 1
fi

#杀死qtApp进程
if [ -z "${pid}" ]; then
  echo "qtApp is not running "
else
 kill -9 ${pid}
fi

cd /mnt/mmc/lcrtu
rm -f qtApp-bak
mv qtApp qtApp-bak

cp /mnt/mmc/tmp/qtApp/qtApp /mnt/mmc/lcrtu

chmod 777 qtApp

./qtApp &