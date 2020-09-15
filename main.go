/*
@Time   : 2020/9/14 12:45
@Author : Haichao Wang
*/
package main

import "lcrtu-update/service"

func main() {
	service.Get().Start()
	service.Get().AddByFunc("MonitoringQTApp", 5, func() { service.MonitoringQTApp() })
	service.StartHttp()
}
