package service

import (
	"github.com/gin-gonic/gin"
	"os"
)

func StartHttp() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.GET("/update/lcrtu", UpdateBackEnd)
	router.GET("/update/qt", UpdateQtApp)
	err := router.Run(":9876")
	if err != nil {
		log.Info("start http server error")
		os.Exit(1)
	} else {
		log.Info("start http server success,listen at:9876")
	}
}
