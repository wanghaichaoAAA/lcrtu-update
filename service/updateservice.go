package service

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"
)

var serviceLog = log.WithFields(log.Fields{"method": "lcrtu update service"})

const (
	FILE_PATH         = "/mnt/mmc/tmp"
	UPDATE_TYPE_LCRTU = "lcrtu"
	UPDATE_TYPE_QT    = "qtApp"
)

func UpdateBackEnd(c *gin.Context) {
	gatewayAddr := c.GetHeader("gateway_addr")
	buildAtStr := c.GetHeader("build_at")

	if gatewayAddr == "" || buildAtStr == "" {
		serviceLog.Error("空参数")
		c.String(http.StatusForbidden, "空参数")
		return
	}
	buildAt, err := time.ParseInLocation("2006-01-02 15:04:05", buildAtStr, time.Local)
	if err != nil {
		serviceLog.Error("编译时间出错")
		c.String(http.StatusForbidden, "参数错误")
		return
	}
	success := updateAndInstallLatestVersion(gatewayAddr, buildAt, UPDATE_TYPE_LCRTU)
	if success {
		c.String(http.StatusOK, "更新成功")
	} else {
		c.String(http.StatusInternalServerError, "更新失败")
	}
}

func UpdateQtApp(c *gin.Context) {
	gatewayAddr := c.GetHeader("gateway_addr")
	buildAtStr := c.GetHeader("build_at")
	if gatewayAddr == "" || buildAtStr == "" {
		serviceLog.Error("空参数")
		c.String(http.StatusForbidden, "空参数")
		return
	}
	buildAt, err := time.ParseInLocation("2006-01-02 15:04:05", buildAtStr, time.Local)
	if err != nil {
		serviceLog.Error("编译时间出错")
		c.String(http.StatusForbidden, "空参数")
		return
	}
	success := updateAndInstallLatestVersion(gatewayAddr, buildAt, UPDATE_TYPE_QT)
	if success {
		c.String(http.StatusOK, "更新成功")
	} else {
		c.String(http.StatusInternalServerError, "更新失败")
	}
}

func updateAndInstallLatestVersion(gatewayAddr string, buildAt time.Time, updateType string) bool {
	for i := 0; i < 5; i++ {
		serviceLog.Error("第", fmt.Sprint(i), "次自动升级", updateType)
		success := updateAndInstall(gatewayAddr, buildAt, updateType)
		if success {
			return true
		}
	}
	return false
}

func updateAndInstall(gatewayAddr string, buildAt time.Time, updateType string) bool {
	//版本检查
	if !checkBackEndVersion(gatewayAddr, buildAt, updateType) {
		serviceLog.Error("已经升级到最新版程序,不执行更新")
		return false
	}
	//下载版本
	if !downLatestVersion(gatewayAddr, updateType) {
		return false
	}
	command := "./scripts/update_backend.sh"
	if updateType == "qtApp" {
		command = "./scripts/update_qt.sh"
	}
	//执行升级脚本
	err := exec.Command("/bin/bash", "-c", command).Run()
	if err != nil {
		serviceLog.Error("自动升级出错:执行更新脚本出错,", err)
		return false
	}
	serviceLog.Error("自动升级成功")
	return true
}

func downLatestVersion(gatewayAddr string, updateType string) bool {
	var (
		fsize   int64
		buf     = make([]byte, 1024*1024)
		written int64
	)
	//1.向网关发起请求，下载最新的压缩包
	remoteAddr := "http://" + gatewayAddr + "/api/update/program?mode=download&type=" + updateType
	resp, err := http.Get(remoteAddr)
	if err != nil {
		serviceLog.Error("自动升级出错:网关获取最新更新包接口出错,", err)
		return false
	}
	if resp.Body == nil {
		serviceLog.Error("自动升级出错:网关获取最新更新包接口响应体为空")
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		serviceLog.Error("自动升级出错:服务器拒绝下载,", resp.StatusCode)
		return false
	}
	fsize, err = strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 32)
	if err != nil {
		serviceLog.Error("自动升级出错:获取Content-Length失败：", err)
		return false
	}
	//2.获取返回值中压缩包的md5值
	fileMd5Str := resp.Header.Get("file_md5")
	if fileMd5Str == "" {
		serviceLog.Error("自动升级出错:网关获取最新更新包接口file_md5字段为空")
		return false
	}
	//3.将返回体中的文件下载到本地
	err = os.MkdirAll(FILE_PATH, os.ModePerm)
	if err != nil {
		serviceLog.Error("自动升级出错:创建tmp目录失败,", err)
		return false
	}
	filePath := FILE_PATH + "/" + updateType + ".zip"
	os.Remove(filePath)
	f, err := os.Create(filePath)
	if err != nil {
		serviceLog.Error("自动升级出错:创建", filePath, "文件出错,", err)
		return false
	}
	defer f.Close()
	fmt.Println("-----------------------开始下载:", time.Now().Format("2006-01-02 15:04:05"), "-----------------------")
	for {
		nr, err := resp.Body.Read(buf)
		if (err != nil && err != io.EOF) || nr <= 0 {
			break
		}
		nw, ew := f.Write(buf[0:nr])
		//写入出错
		if ew != nil {
			serviceLog.Error("自动升级出错:写入本地文件出错：", err)
			break
		}
		//读取是数据长度不等于写入的数据长度
		if nr != nw {
			serviceLog.Error("自动升级出错:写入本地文件的数据长度出错")
			break
		}
		if nw > 0 {
			written += int64(nw)
		}
		fmt.Print(fmt.Sprintf("%.0f", float32(written)/float32(fsize)*100), "% ")
	}
	fmt.Println()
	fmt.Println("-----------------------下载结束:", time.Now().Format("2006-01-02 15:04:05"), "-----------------------")

	md5 := md5.New()
	f.Seek(0, 0)
	_, err = io.Copy(md5, f)
	if err != nil {
		serviceLog.Error("自动升级出错:生成MD5值出错,", err)
		return false
	}
	//4.计算下载后的md5值，比较，相等返回true
	md5Str := hex.EncodeToString(md5.Sum(nil))
	println("md5:", md5Str)
	if md5Str != fileMd5Str {
		serviceLog.Error("自动升级出错:MD5值校验失败")
		return false
	}
	return true
}

func checkBackEndVersion(gatewayAddr string, buildAt time.Time, updateType string) bool {
	//获取最新版本
	remoteAddr := "http://" + gatewayAddr + "/api/update/program?mode=version&type=" + updateType
	resp, err := http.Get(remoteAddr)
	if err != nil {
		serviceLog.Error("自动升级出错:请求网关最新版本接口出错,", err)
		return false
	}
	defer resp.Body.Close()
	latestBuildAtStr := resp.Header.Get("build_at")
	if latestBuildAtStr == "" {
		serviceLog.Error("自动升级出错:网关最新版本接口build_at字段为空")
		return false
	}
	latestBuildAt, err := time.ParseInLocation("2006-01-02 15:04:05", latestBuildAtStr, time.Local)
	if err != nil {
		serviceLog.Error("自动升级出错:网关最新版本接口build_at字段类型错误,", err)
		return false
	}
	//当前版本的时间早于最新的版本时间，返回true
	if buildAt.Before(latestBuildAt) {
		return true
	}
	return false
}

func existQtAppPid() bool {
	var outInfo bytes.Buffer
	command := "./scripts/pid_atApp.sh"
	cmd := exec.Command("/bin/bash", "-c", command)
	cmd.Stdout = &outInfo
	cmd.Run()
	if outInfo.Len() > 5 && outInfo.String() != "<nil>" {
		return true
	}
	return false
}

func UpdateGivenBackEnd(c *gin.Context) {
	fileId := c.Query("file_id")
	gateWayAddr := c.Query("gateway_addr")
	if fileId == "" || gateWayAddr == "" {
		serviceLog.Error("下载指定安装包出错：参数不能为空")
		c.String(http.StatusForbidden, "下载指定安装包出错:参数不能为空")
		return
	}
	success := downloadAndInstallGivenVersionProgram(gateWayAddr, UPDATE_TYPE_LCRTU, fileId)
	if success {
		c.String(http.StatusOK, "更新成功")
	}
	c.String(http.StatusInternalServerError, "下载指定安装包出错")
}

func UpdateGivenQtApp(c *gin.Context) {
	fileId := c.Query("file_id")
	gateWayAddr := c.Query("gateway_addr")
	if fileId == "" || gateWayAddr == "" {
		serviceLog.Error("下载指定安装包出错：参数不能为空")
		c.String(http.StatusForbidden, "下载指定安装包出错:参数不能为空")
		return
	}
	success := downloadAndInstallGivenVersionProgram(gateWayAddr, UPDATE_TYPE_QT, fileId)
	if success {
		c.String(http.StatusOK, "更新成功")
	}
	c.String(http.StatusInternalServerError, "下载指定安装包出错")
}

func downloadAndInstallGivenVersionProgram(gateWayAddr, downloadType, fileId string) bool {
	for i := 0; i < 5; i++ {
		serviceLog.Error("第 "+fmt.Sprint(i)+" 下载指定版本升级包,网关地址:", gateWayAddr, ",下载类型:", downloadType, ",文件id:", fileId)
		success := downloadAndInstall(gateWayAddr, downloadType, fileId)
		if success {
			return true
		}
	}
	return false
}

func downloadAndInstall(gateWayAddr, downloadType, fileId string) bool {
	var buf = make([]byte, 1024*1024)
	var written int64
	var fsize int64
	url := "/api/qt_update/special_version?qt_id="
	fileName := "/qtApp.zip"
	if downloadType == UPDATE_TYPE_LCRTU {
		url = "/api/rtu_update/special_version?rtu_id="
		fileName = "/lcrtu.zip"
	}
	//1.向网关下载特定版本的压缩包
	resp, err := http.Get("http://" + gateWayAddr + url + fileId)
	if err != nil {
		serviceLog.Error("下载指定版本升级包出错:请求网关下载地址错误,", err.Error())
		return false
	}
	defer resp.Body.Close()
	fsize, contentLengthErr := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 32)
	if contentLengthErr != nil {
		serviceLog.Error("下载指定版本升级包出错:获取Content-Length失败：", contentLengthErr)
		return false
	}
	os.Remove(FILE_PATH + fileName)
	//将文件下载到本地,重命名压缩包
	createFile, err := os.Create(FILE_PATH + fileName)
	if err != nil {
		serviceLog.Error("下载指定版本升级包出错:创建下载文件出错,", err)
		return false
	}
	defer createFile.Close()

	fmt.Println("-----------------------开始下载:", time.Now().Format("2006-01-02 15:04:05"), "-----------------------")
	for {
		nr, err := resp.Body.Read(buf)
		if (err != nil && err != io.EOF) || nr <= 0 {
			break
		}
		nw, ew := createFile.Write(buf[0:nr])
		//写入出错
		if ew != nil {
			serviceLog.Error("下载指定版本升级包出错:写入本地文件出错,", err)
			break
		}
		//读取是数据长度不等于写入的数据长度
		if nr != nw {
			serviceLog.Error("下载指定版本升级包出错:写入本地文件的数据长度出错")
			break
		}
		if nw > 0 {
			written += int64(nw)
		}
		fmt.Print(fmt.Sprintf("%.0f", float32(written)/float32(fsize)*100), "% ")
	}
	fmt.Println()
	fmt.Println("-----------------------下载结束:", time.Now().Format("2006-01-02 15:04:05"), "-----------------------")
	command := "./scripts/update_backend.sh"
	if downloadType == UPDATE_TYPE_QT {
		command = "./scripts/update_qt.sh"
	}
	execErr := exec.Command("/bin/bash", "-c", command).Run()
	if execErr != nil {
		serviceLog.Error("更新指定版本升级包出错:执行更新脚本出错：", execErr)
		return false
	}
	return true
}

//数采仪本地安装
func UpdateLocalRtuApp(c *gin.Context) {
	fileType := c.Query("file_type")
	if fileType == "" || (fileType != "qt" && fileType != "lcrtu") {
		c.String(http.StatusOK, "file_type error")
		return
	}
	//qt
	if fileType == "qt" {
		command := "./scripts/update_qt.sh"
		err := exec.Command("/bin/bash", "-c", command).Run()
		if err != nil {
			serviceLog.Error("执行更新脚本出错：", err)
			c.String(http.StatusForbidden, "执行更新脚本出错")
			return
		}
		c.String(http.StatusOK, "更新成功")
		return
	}
	//数采仪
	command := "./scripts/update_backend.sh"
	err := exec.Command("/bin/bash", "-c", command).Run()
	if err != nil {
		serviceLog.Error("执行更新脚本出错：", err)
		c.String(http.StatusForbidden, "执行更新脚本出错")
		return
	}
	c.String(http.StatusOK, "更新成功")
}
