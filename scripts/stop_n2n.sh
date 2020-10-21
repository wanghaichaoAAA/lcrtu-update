#!/bin/bash

ps -ef|grep edge|grep -v grep|awk '{print $2}'|xargs kill -9
