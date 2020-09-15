#!/bin/bash

APP_NAME=qtApp

pid=`ps -ef|grep $APP_NAME|grep -v grep|awk '{print $2}' `

if [ -z "${pid}" ]; then
  echo "qtApp is killed ,starting process..."
  /mnt/mmc/lcrtu/qtApp &
else
  echo "qtApp is ok"
fi
