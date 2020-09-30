package service

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"
)

var log = logging.MustGetLogger("service")

const (
	FILE_PATH = "/mnt/mmc/tmp"
)

func UpdateBackEnd(c *gin.Context) {
	gatewayAddr := c.GetHeader("gateway_addr")
	buildAtStr := c.GetHeader("build_at")

	if gatewayAddr == "" || buildAtStr == "" {
		log.Error("空参数")
		c.String(http.StatusForbidden, "空参数")
		return
	}
	buildAt, err := time.ParseInLocation("2006-01-02 15:04:05", buildAtStr, time.Local)
	if err != nil {
		log.Error("编译时间出错")
		c.String(http.StatusForbidden, "参数错误")
		return
	}

	//1.版本检查，传入当前版本的时间，然后获取网关最新版本，如果落后就返回true
	if !checkBackEndVersion(gatewayAddr, buildAt, "lcrtu") {
		log.Error("已经升级到最新版程序")
		c.String(http.StatusForbidden, "版本检查失败")
		return
	}
	//2.下载版本,
	if !downLatestVersion(gatewayAddr, "lcrtu") {
		log.Error("下载最新版程序出错")
		c.String(http.StatusForbidden, "下载最新程序失败")
		return
	}
	//3.执行升级程序命令
	command := "./scripts/update_backend.sh"
	err = exec.Command("/bin/bash", "-c", command).Run()
	if err != nil {
		log.Error("执行更新脚本出错：", err)
		c.String(http.StatusForbidden, "执行更新脚本出错")
		return
	}
	c.String(http.StatusOK, "更新成功")
}

func UpdateQtApp(c *gin.Context) {
	gatewayAddr := c.GetHeader("gateway_addr")
	buildAtStr := c.GetHeader("build_at")
	if gatewayAddr == "" || buildAtStr == "" {
		log.Error("空参数")
		return
	}
	buildAt, err := time.ParseInLocation("2006-01-02 15:04:05", buildAtStr, time.Local)
	if err != nil {
		log.Error("编译时间出错")
		return
	}
	//1.版本检查
	if !checkBackEndVersion(gatewayAddr, buildAt, "qtApp") {
		log.Error("已经升级到最新版程序")
		return
	}
	//2.下载版本
	if !downLatestVersion(gatewayAddr, "qtApp") {
		log.Error("下载最新版程序出错")
	}
	Get().DelByID("MonitoringQTApp")
	//3.升级程序
	command := "./scripts/update_qt.sh"
	err = exec.Command("/bin/bash", "-c", command).Run()
	if err != nil {
		log.Error("执行更新脚本出错：", err)
	}
	Get().AddByFunc("MonitoringQTApp", 5, func() { MonitoringQTApp() })
}

func downLatestVersion(gatewayAddr string, updateType string) bool {
	var (
		fsize   int64
		buf     = make([]byte, 1024*1024*1024)
		written int64
	)

	//1.向网关发起请求，下载最新的压缩包
	remoteAddr := "http://" + gatewayAddr + "/api/update/program?mode=download&type=" + updateType
	resp, err := http.Get(remoteAddr)
	if err != nil {
		log.Error("下载最新版本出错：", err)
		return false
	}
	if resp.Body == nil {
		log.Error("返回为空")
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Error("服务器拒绝下载：", resp.StatusCode)
		return false
	}
	fsize, err = strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 32)
	if err != nil {
		log.Error("获取Content-Length失败：", err)
		return false
	}
	//2.获取返回值中压缩包的md5值
	fileMd5Str := resp.Header.Get("file_md5")
	if fileMd5Str == "" {
		log.Error("file_md5字段为空")
		return false
	}
	//3.将返回体中的文件下载到本地
	err = os.MkdirAll(FILE_PATH, os.ModePerm)
	if err != nil {
		log.Error("创建tmp目录失败:", err)
		return false
	}
	filePath := FILE_PATH + "/" + updateType + ".zip"
	os.Remove(filePath)
	f, err := os.Create(filePath)
	if err != nil {
		log.Error("create ", filePath, " error ", err)
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
			log.Error("写入本地文件出错：", err)
			break
		}
		//读取是数据长度不等于写入的数据长度
		if nr != nw {
			log.Error("写入本地文件的数据长度出错")
			break
		}
		if nw > 0 {
			written += int64(nw)
		}
		fmt.Print(fmt.Sprintf("%.0f", float32(written)/float32(fsize)*100), "% ")
	}
	fmt.Println()
	fmt.Println("-----------------------下载结束:", time.Now().Format("2006-01-02 15:04:05"), "-----------------------")
	//_, err = io.Copy(f, resp.Body)
	//if err != nil {
	//	log.Error("copy resp.Body to ", filePath, " error ", err)
	//	return false
	//}

	//fNew, _ := os.Open(filePath)
	md5 := md5.New()
	_, err = io.Copy(md5, f)
	if err != nil {
		log.Error("generate md5 error ", err)
		return false
	}
	//4.计算下载后的md5值，比较，相等返回true
	md5Str := hex.EncodeToString(md5.Sum(nil))
	if md5Str != fileMd5Str {
		log.Error("verify md5 error ", err)
		return false
	}
	return true
}

func checkBackEndVersion(gatewayAddr string, buildAt time.Time, updateType string) bool {

	//获取最新版本
	remoteAddr := "http://" + gatewayAddr + "/api/update/program?mode=version&type=" + updateType
	resp, err := http.Get(remoteAddr)
	if err != nil {
		log.Error("请求网关获取版本接口出错：", err)
		return false
	}
	defer resp.Body.Close()
	latestBuildAtStr := resp.Header.Get("build_at")
	if latestBuildAtStr == "" {
		log.Error("请求网关获取版本接口出错:build_at字段为空")
		return false
	}
	latestBuildAt, err := time.ParseInLocation("2006-01-02 15:04:05", latestBuildAtStr, time.Local)
	if err != nil {
		log.Error("build_at字段类型错误:", latestBuildAtStr)
		return false
	}

	//当前版本的时间早于最新的版本时间，返回true
	if buildAt.Before(latestBuildAt) {
		return true
	}
	return false
}
func MonitoringQTApp() {
	command := "./scripts/monitor_qt.sh"
	cmd := exec.Command("/bin/bash", "-c", command)
	_ = cmd.Run()
}

type resStruct struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    []byte `json:"data"`
}

func UpdateGivenBackEnd(c *gin.Context) {

	fileId := c.Query("file_id")
	if fileId == "" {
		log.Error("执行更新脚本出错：id不能为空")
		c.String(http.StatusForbidden, "获取新版文件出错,id不能为空")
		return
	}

	//1.向网关下载特定版本的压缩包
	getRes, getErr := http.Get("http://172.20.0.70:9002/api/rtu_update/special_version?rtu_id=" + fileId)
	if getErr != nil {
		log.Error("执行更新脚本出错：", getErr.Error())
		c.String(http.StatusForbidden, "获取新版文件出错")
		return
	}
	defer getRes.Body.Close()
	body, _ := ioutil.ReadAll(getRes.Body)
	var res resStruct
	jsonErr := json.Unmarshal(body, &res)
	if jsonErr != nil {
		log.Error("解析数据出错：", jsonErr.Error())
		c.String(http.StatusForbidden, "解析数据出错")
		return
	}

	//将文件下载到本地,重命名压缩包
	out, err := os.Create(FILE_PATH + "/lcrtu.zip")
	if err != nil {
		log.Error("执行更新脚本出错：", err)
		c.String(http.StatusForbidden, "获取新版文件出错")
		return
	}
	defer out.Close()

	writeCount, writeErr := out.Write(res.Data)
	println("写入数量：", writeCount)
	if writeErr != nil {
		log.Error("执行写入出错：", writeErr)
		c.String(http.StatusForbidden, "获取新版文件出错")
		return
	}

	//3.执行更新脚本
	command := "./scripts/update_backend.sh"
	execErr := exec.Command("/bin/bash", "-c", command).Run()
	if execErr != nil {
		log.Error("执行更新脚本出错：", execErr)
		c.String(http.StatusForbidden, "执行更新脚本出错")
		return
	}
	c.String(http.StatusOK, "更新成功")
}

func UpdateGivenQtApp(c *gin.Context) {
	fileId := c.Query("file_id")
	if fileId == "" {
		log.Error("执行更新脚本出错：id不能为空")
		c.String(http.StatusForbidden, "获取新版文件出错,id不能为空")
		return
	}

	//1.向网关下载特定版本的压缩包
	getRes, getErr := http.Get("http://172.20.0.70:9002/api/qt_update/special_version?qt_id=" + fileId)
	if getErr != nil {
		log.Error("执行更新脚本出错：", getErr.Error())
		c.String(http.StatusForbidden, "获取新版文件出错")
		return
	}
	defer getRes.Body.Close()
	body, _ := ioutil.ReadAll(getRes.Body)
	var res resStruct
	jsonErr := json.Unmarshal(body, &res)
	if jsonErr != nil {
		log.Error("解析数据出错：", jsonErr.Error())
		c.String(http.StatusForbidden, "解析数据出错")
		return
	}

	//将文件下载到本地,重命名压缩包
	out, err := os.Create(FILE_PATH + "/qtApp.zip")
	if err != nil {
		log.Error("执行更新脚本出错：", err)
		c.String(http.StatusForbidden, "获取新版文件出错")
		return
	}
	defer out.Close()

	writeCount, writeErr := out.Write(res.Data)
	println("写入数量：", writeCount)
	if writeErr != nil {
		log.Error("执行写入出错：", writeErr)
		c.String(http.StatusForbidden, "获取新版文件出错")
		return
	}

	//3.执行更新脚本
	command := "./scripts/update_qt.sh"
	execErr := exec.Command("/bin/bash", "-c", command).Run()
	if execErr != nil {
		log.Error("执行更新脚本出错：", execErr)
		c.String(http.StatusForbidden, "执行更新脚本出错")
		return
	}
	c.String(http.StatusOK, "更新成功")
}

func UpdateLocalRtuApp(c *gin.Context) {
	//3.升级程序
	command := "./scripts/update_backend.sh"
	err := exec.Command("/bin/bash", "-c", command).Run()
	if err != nil {
		log.Error("执行更新脚本出错：", err)
		c.String(http.StatusForbidden, "执行更新脚本出错")
	}
	Get().AddByFunc("MonitoringQTApp", 5, func() { MonitoringQTApp() })
	c.String(http.StatusOK, "更新成功")
}

func UpdateLocalQtApp(c *gin.Context) {
	Get().DelByID("MonitoringQTApp")
	//3.升级程序
	command := "./scripts/update_qt.sh"
	err := exec.Command("/bin/bash", "-c", command).Run()
	if err != nil {
		log.Error("执行更新脚本出错：", err)
		c.String(http.StatusForbidden, "执行更新脚本出错")
	}
	Get().AddByFunc("MonitoringQTApp", 5, func() { MonitoringQTApp() })
	c.String(http.StatusOK, "更新成功")
}
