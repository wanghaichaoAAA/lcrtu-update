#!/bin/bash

ps -ef|grep qtApp|grep -v grep|awk '{print $2}'