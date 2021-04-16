package main

import (
	"encoding/json"
	"errors"
	"github.com/afex/hystrix-go/hystrix"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"time"
)

//type DSList struct {
//	lock   *sync.Mutex
//	head   *DSNode
//	tail   *DSNode
//	length int
//}

type DSNode struct {
	Name   string
	Next   *DSNode
	Pre    *DSNode
	stream Stream
}

type Stream interface {
	GetData() (string, error)
}

func init() {
	initLogrus()
	initDataSource()
	initHystrix()
}

func initLogrus() {
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})
	log.SetReportCaller(true)
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stdout)
	//log.SetBufferPool()
}

//var root = &DSNode{Name: "root"}
var root *DSNode

type YinLianStream struct{}

var ylRecover = time.Now().Add(time.Minute * 2)

func (self *YinLianStream) GetData() (string, error) {
	if time.Now().Before(ylRecover) {
		return "", errors.New("the network timeout")
	}
	return "data", nil
}

type BaFangStream struct{}

var bfRecover = time.Now().Add(time.Minute)

func (self *BaFangStream) GetData() (string, error) {
	if time.Now().Before(bfRecover) {
		return "", errors.New("the network timeout")
	}
	return "data", nil
}

type XinYanStream struct {
}

func (self *XinYanStream) GetData() (string, error) {
	return "data", nil
}

var YinLian = "YinLian"
var BaFang = "BaFang"
var XinYan = "XinYan"

func initDataSource() {
	ylNode := &DSNode{
		Name:   YinLian,
		Next:   nil,
		Pre:    nil,
		stream: &YinLianStream{},
	}
	bfNode := &DSNode{
		Name:   BaFang,
		Next:   nil,
		Pre:    nil,
		stream: &BaFangStream{},
	}
	xyNode := &DSNode{
		Name:   XinYan,
		Next:   nil,
		Pre:    nil,
		stream: &XinYanStream{},
	}

	ylNode.Next = bfNode
	bfNode.Next = xyNode

	xyNode.Pre = bfNode
	bfNode.Pre = ylNode

	root = ylNode
}

func initHystrix() {
	defaultCmd := hystrix.CommandConfig{
		Timeout:               int(time.Second * 3),
		MaxConcurrentRequests: 10,
		//统计窗口（10s）， 在统计窗口中，达到这个请求数量后才去判断是否要开启熔断
		RequestVolumeThreshold: 10,
		//当熔断器被打开后，SleepWindow 的时间就是控制过多久后去尝试服务是否可用了。
		SleepWindow: 60 * 1000,
		//错误百分比，请求数量大于等于 RequestVolumeThreshold 并且错误率到达这个百分比后就会启动熔断
		ErrorPercentThreshold: 50,
	}

	cmds := make(map[string]hystrix.CommandConfig)
	cmds[YinLian] = defaultCmd
	cmds[BaFang] = defaultCmd
	cmds[XinYan] = defaultCmd

	hystrix.Configure(cmds)
}

func commonHandler(w http.ResponseWriter, r *http.Request) {
	ret := make(chan string, 1)

	tmp := root
	query(tmp, ret)

	msg := <-ret
	w.Write([]byte(msg))
}

func query(node *DSNode, ret chan string) {
	hystrix.Go(node.Name, func() error {
		//业务逻辑
		log.Debug("get data by ", node.Name)
		data, err := node.stream.GetData()
		if err != nil {
			//log.Errorf("DataSource[%s] has err[%v]", node.Name, err)
			return err
		}

		js := map[string]string{}
		js["DataSource"] = node.Name
		js["Data"] = data

		msg, _ := json.Marshal(js)
		ret <- string(msg)
		return nil
	}, func(err error) error {
		//业务逻辑失败后的处理逻辑
		log.Errorf("Hystrix catch %s err[%v]", node.Name, err)

		if node.Next != nil {
			//log.Debugf("备份接口不为空，调用备份接口[%s]", node.Next.Name)
			query(node.Next, ret)
			return nil
		}

		ret <- err.Error()
		return nil
	})
}

func main() {
	log.Info("SERVER START RUN...")

	http.HandleFunc("/test", commonHandler)

	log.Info(http.ListenAndServe(":8080", nil))
}
