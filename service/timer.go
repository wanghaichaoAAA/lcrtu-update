package service

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	"sync"
)

//cron 结构体
type Timer struct {
	inner *cron.Cron
	ids   map[string]cron.EntryID
	mutex sync.Mutex
}

var cronTimer *Timer

//定时器初始化
func init() {
	cronTimer = &Timer{
		inner: cron.New(cron.WithSeconds()),
		ids:   make(map[string]cron.EntryID),
	}
}

func Get() *Timer {
	return cronTimer
}

//创建新的定时任务
func (c *Timer) AddByFunc(id string, second int, f func()) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.ids[id]; ok {
		return errors.Errorf("scheduled task is already exist,id=", id)
	}
	cronSpec := SecondConvertCron(second)
	eid, err := c.inner.AddFunc(cronSpec, f)
	if err != nil {
		return err
	}
	c.ids[id] = eid
	log.Info("create scheduled task success,id=", id, " interval:", cronSpec)
	return nil
}

//根据定时任务id删除单个任务
func (c *Timer) DelByID(id string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	eid, ok := c.ids[id]
	if !ok {
		return
	}
	c.inner.Remove(eid)
	delete(c.ids, id)
	log.Info("delete scheduled task success,id:", id)
}

//删除整个定时器
func (c *Timer) Stop() {
	c.inner.Stop()
}

//判断定时任务是否存在
func (c *Timer) IsExists(jid string) bool {
	_, exist := c.ids[jid]
	return exist
}

func (c *Timer) Start() {
	c.inner.Start()
}

//秒数转cron字符串
func SecondConvertCron(secondInt int) string {
	if secondInt <= 59 {
		return fmt.Sprintf("0/%d * * * * *", secondInt)
	} else if secondInt <= 3540 {
		return fmt.Sprintf("* */%d * * * *", secondInt/60)
	} else if secondInt <= 82800 {
		return fmt.Sprintf("* * */%d * * *", secondInt/60/60)
	} else if secondInt <= 31449600 {
		return fmt.Sprintf("* * * */%d * * *", secondInt/24/60/60)
	}
	return ""
}
