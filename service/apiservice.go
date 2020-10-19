package service

import (
	"github.com/gin-gonic/gin"
	"os"
)

func StartHttp() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	//自定下载更新
	router.GET("/update/lcrtu", UpdateBackEnd)
	router.GET("/update/qt", UpdateQtApp)
	//网关手动安装
	router.GET("/update/given/lcrtu", UpdateGivenBackEnd)
	router.GET("/update/given/qt", UpdateGivenQtApp)
	//本地手动安装
	router.GET("/update/local", UpdateLocalRtuApp)
	//代理管理
	router.GET("/update/agent", AgentManage)
	router.GET("/update/agent/status", agentStatus) //代理状态
	serviceLog.Info("lcrtu更新服务启动成功,端口:9876")
	err := router.Run(":9876")
	if err != nil {
		serviceLog.Warn("lcrtu更新服务启动失败:", err)
		os.Exit(1)
	}
}
