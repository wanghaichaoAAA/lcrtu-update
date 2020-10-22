package service

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"os/exec"
	"sync"
	"time"
)

var agentUsed = false        //正在使用是true
var agentStartTime time.Time //代理开始时间
var agentIP string           //代理IP
var mute sync.Mutex          //互斥锁

//数采仪代理
func AgentManage(c *gin.Context) {
	operation := c.Query("operation")    //使用状态
	agentAddr := c.Query("agent_addr")   //代理地址
	serverAddr := c.Query("server_addr") //服务器地址
	if operation != "start" && operation != "stop" {
		serviceLog.Error("操作错误")
		c.String(http.StatusForbidden, "err:操作错误")
		return
	}

	if operation == "start" {
		if agentUsed {
			serviceLog.Error("代理已开启")
			c.String(http.StatusForbidden, "err:代理已开启")
			return
		}
		mute.Lock()
		defer mute.Unlock()
		err := startAgent(agentAddr, serverAddr)
		if err != nil {
			serviceLog.Error("代理开启失败", err.Error())
			c.String(http.StatusForbidden, "err:代理开启失败", err.Error())
			return
		}
		agentUsed = true

	} else {
		if !agentUsed {
			serviceLog.Error("代理已关闭")
			c.String(http.StatusForbidden, "err:代理已关闭")
			return
		}
		stopAgent()
	}
	c.String(http.StatusOK, operation+"成功")
}

//开启代理
func startAgent(agentAddr string, serverAddr string) error {
	exec.Command("/bin/bash", "-c", "/mnt/mmc/lcrtu/scripts/stop_n2n.sh").Run()
	exec.Command("/bin/bash", "-c", "/mnt/mmc/lcrtu/scripts/add_ko.sh").Run()
	err := exec.Command("/bin/bash", "-c", "/mnt/mmc/lcrtu/scripts/start_edge.sh "+agentAddr+" "+serverAddr).Run()
	if err != nil {
		serviceLog.Error("开启代理失败", err.Error(), "agentIP:", agentAddr, "serverIP:", serverAddr)
		return err
	}
	agentStartTime = time.Now()
	agentIP = agentAddr
	serviceLog.Error("开启代理成功", "agentIP:", agentIP, "serverIP:", serverAddr)
	time.AfterFunc(time.Duration(2)*time.Hour, stopAgent)
	return nil
}

//关闭代理
func stopAgent() {
	command := exec.Command("/bin/bash", "-c", "/mnt/mmc/lcrtu/scripts/stop_n2n.sh")
	command.Run()
	agentUsed = false
	agentIP = ""
	agentStartTime = time.Time{}
	serviceLog.Error("关闭代理成功", "agentIP:", agentIP)
}

//状态
func agentStatus(c *gin.Context) {
	agentStatus := "开启"
	if !agentUsed {
		agentStatus = "关闭"
	}
	resStr := agentStatus + "," + agentIP + "," + agentStartTime.Format("2006-01-02 15:04:05")
	serviceLog.Error("获取代理信息成功", "agentIP:", agentIP)
	c.String(http.StatusOK, resStr)
}
