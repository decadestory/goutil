package logsvc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/IBM/sarama"
	"github.com/decadestory/goutil/auth"
	"github.com/decadestory/goutil/br"
	"github.com/decadestory/goutil/conf"
	"github.com/decadestory/goutil/exception"
	"github.com/decadestory/goutil/logger"
	"github.com/decadestory/goutil/misc"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/go-uuid"
)

var serviceId string
var client sarama.SyncProducer
var sip string

const maxLogSize = 2 * 1024 * 1024      // 只保留前 2MB
const truncatedMark = "... [TRUNCATED]" // 只保留前 2MB

type LogSvc struct{}

var LogSvcs = LogSvc{}

// 自定义 ResponseWriter，实现 gin.ResponseWriter 接口
type customResponseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *customResponseWriter) Write(data []byte) (int, error) {
	if w.body.Len() < maxLogSize {
		if len(data)+w.body.Len() > maxLogSize {
			remain := maxLogSize - w.body.Len()
			w.body.Write(data[:remain])
			w.body.WriteString(truncatedMark)
		} else {
			w.body.Write(data)
		}
	}
	return w.ResponseWriter.Write(data)
}

func init() {
	var host string = conf.Configs.GetString("kafka.log.host")
	serviceId = conf.Configs.GetString("service.name")
	sip = misc.GetIp()
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.NoResponse
	config.Producer.Partitioner = sarama.NewHashPartitioner
	config.Producer.Retry.Max = 0
	config.Producer.Compression = sarama.CompressionSnappy
	config.Producer.Timeout = time.Second * 1
	config.Producer.Return.Successes = true
	newClient, err := sarama.NewSyncProducer([]string{host}, config)
	logger.Logs.Error(err)
	client = newClient
}

// Infos 不需要传上下文
// args[0] LogExtTxt
// args[1] Duration
func (log *LogSvc) Infos(msg string, args ...any) {

	var ext string = misc.Ternary(len(args) > 0, args[0].(string), "")
	var dur int64 = misc.Ternary(len(args) > 1, args[1].(int64), 0)

	item := misc.Logger{
		ServiceId:  serviceId,
		Path:       log.getStack(),
		LogType:    "debug",
		LogLevel:   1,
		LogTxt:     msg,
		LogExtTxt:  ext,
		Duration:   dur,
		CreateTime: time.Now().Format(misc.FMTMillSEC),
	}

	m, _ := json.Marshal(item)
	log.kafkaProducer(string(m))
}

// Info
// args[0] LogExtTxt
// args[1] Duration
func (log *LogSvc) Bus(c *gin.Context, msg string, args ...any) {

	var ext string = misc.Ternary(len(args) > 0, args[0].(string), "")
	var dur int64 = misc.Ternary(len(args) > 1, args[1].(int64), 0)

	item := misc.Logger{
		ServiceId:  serviceId,
		Ip:         sip,
		Path:       log.getStack(),
		LogType:    "bus",
		LogLevel:   1,
		LogTxt:     msg,
		LogExtTxt:  ext,
		Duration:   dur,
		CreateTime: time.Now().Format(misc.FMTMillSEC),
	}

	log.setUserInfo(c, &item)
	log.setReqId(c, &item)

	m, _ := json.Marshal(item)
	log.kafkaProducer(string(m))
}

// Info
// args[0] LogExtTxt
// args[1] Duration
func (log *LogSvc) Info(c *gin.Context, msg string, args ...any) {

	var ext string = misc.Ternary(len(args) > 0, args[0].(string), "")
	var dur int64 = misc.Ternary(len(args) > 1, args[1].(int64), 0)

	item := misc.Logger{
		ServiceId:  serviceId,
		Ip:         sip,
		Path:       log.getStack(),
		LogType:    "debug",
		LogLevel:   1,
		LogTxt:     msg,
		LogExtTxt:  ext,
		Duration:   dur,
		CreateTime: time.Now().Format(misc.FMTMillSEC),
	}

	log.setUserInfo(c, &item)
	log.setReqId(c, &item)

	m, _ := json.Marshal(item)
	log.kafkaProducer(string(m))
}

// Debug args[0]
// LogExtTxt,
// args[1] Duration
func (log *LogSvc) Debug(c *gin.Context, msg string, args ...any) {

	var ext string = misc.Ternary(len(args) > 0, args[0].(string), "")
	var dur int64 = misc.Ternary(len(args) > 1, args[1].(int64), 0)

	item := misc.Logger{
		ServiceId:  serviceId,
		Ip:         sip,
		Path:       log.getStack(),
		LogType:    "debug",
		LogLevel:   2,
		LogTxt:     msg,
		LogExtTxt:  ext,
		Duration:   dur,
		CreateTime: time.Now().Format(misc.FMTMillSEC),
	}

	log.setUserInfo(c, &item)
	log.setReqId(c, &item)

	m, _ := json.Marshal(item)
	log.kafkaProducer(string(m))
}

// Debug args[0]
// LogExtTxt,
// args[1] Duration
func (log *LogSvc) Warn(c *gin.Context, msg string, args ...any) {

	var ext string = misc.Ternary(len(args) > 0, args[0].(string), "")
	var dur int64 = misc.Ternary(len(args) > 1, args[1].(int64), 0)

	item := misc.Logger{
		ServiceId:  serviceId,
		Ip:         sip,
		Path:       log.getStack(),
		LogType:    "debug",
		LogLevel:   3,
		LogTxt:     msg,
		LogExtTxt:  ext,
		Duration:   dur,
		CreateTime: time.Now().Format(misc.FMTMillSEC),
	}

	log.setUserInfo(c, &item)
	log.setReqId(c, &item)

	m, _ := json.Marshal(item)
	log.kafkaProducer(string(m))
}

func (log *LogSvc) Error(c *gin.Context, err error) {
	buf := make([]byte, 1024)
	n := runtime.Stack(buf, false)

	item := misc.Logger{
		ServiceId:  serviceId,
		Ip:         sip,
		Path:       log.getStack(),
		LogType:    "exception",
		LogLevel:   4,
		LogTxt:     exception.Errors.ErrorToString(err),
		LogExtTxt:  string(buf[:n]),
		CreateTime: time.Now().Format(misc.FMTMillSEC),
	}

	log.setUserInfo(c, &item)
	log.setReqId(c, &item)

	m, _ := json.Marshal(item)
	log.kafkaProducer(string(m))
}

func (log *LogSvc) Fatal(c *gin.Context, r any) {

	buf := make([]byte, 1024)
	n := runtime.Stack(buf, false)

	item := misc.Logger{
		ServiceId:  serviceId,
		Ip:         sip,
		Path:       log.getStack(),
		LogType:    "exception",
		LogLevel:   5,
		LogTxt:     exception.Errors.ErrorToString(r),
		LogExtTxt:  string(buf[:n]),
		CreateTime: time.Now().Format(misc.FMTMillSEC),
	}

	log.setUserInfo(c, &item)
	log.setReqId(c, &item)

	m, _ := json.Marshal(item)
	log.kafkaProducer(string(m))
}

func (log *LogSvc) LogApi(c *gin.Context, dur int64, reqData, repData string) {
	item := misc.Logger{
		ServiceId:  serviceId,
		Ip:         sip,
		Path:       c.FullPath(),
		LogType:    "api",
		LogLevel:   1,
		LogTxt:     reqData,
		LogExtTxt:  repData,
		Duration:   dur,
		CreateTime: time.Now().Format(misc.FMTMillSEC),
	}

	log.setUserInfo(c, &item)
	log.setReqId(c, &item)

	m, _ := json.Marshal(item)
	log.kafkaProducer(string(m))
}

func (log *LogSvc) LogApiMidware(c *gin.Context) {

	reqId := c.GetHeader("requestId")
	if reqId == "" {
		reqId, _ = uuid.GenerateUUID()
	}
	c.Set("requestId", reqId)

	st := time.Now()
	var reqData []byte

	body, _ := io.ReadAll(c.Request.Body)

	if len(body) > maxLogSize {
		keep := misc.Ternary(maxLogSize-len(truncatedMark) > 0, maxLogSize-len(truncatedMark), 0)
		reqData = append(body[:keep], []byte(truncatedMark)...)
	} else {
		reqData = body
	}

	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	writer := &customResponseWriter{
		ResponseWriter: c.Writer,
		body:           bytes.NewBufferString(""),
	}
	c.Writer = writer
	c.Next()

	log.LogApi(c, time.Since(st).Milliseconds(), string(reqData), writer.body.String())
}

// Gin中间件，全局捕获异常
func (log *LogSvc) Recover(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			debug.PrintStack()
			//封装通用json返回
			res := br.Br{Status: -2, ExtData: 0, Data: nil, Msg: exception.Errors.ErrorToString(r)}
			log.Fatal(c, r)
			c.JSON(http.StatusOK, res)
			//终止后续接口调用，不加的话recover到异常后，还会继续执行接口里后续代码
			c.Abort()
		}
	}()
	//加载完 defer recover，继续后续接口调用
	c.Next()
}

func (log *LogSvc) kafkaProducer(msgTxt string) {
	msg := &sarama.ProducerMessage{}
	msg.Topic = "sme-logger"
	msg.Value = sarama.StringEncoder(msgTxt)
	_, _, err := client.SendMessage(msg)
	logger.Logs.Error(err)
}

func (log *LogSvc) getStack() (funcs string) {
	// 获取调用链路径信息
	pc := make([]uintptr, 10)
	n := runtime.Callers(0, pc)
	frames := runtime.CallersFrames(pc[:n])

	// 遍历调用链路径并打印
	for {
		frame, more := frames.Next()
		if !more {
			funcs += fmt.Sprintf("%s:%d", frame.Function, frame.Line)
			break
		}
	}
	return
}

func (log *LogSvc) setReqId(c *gin.Context, item *misc.Logger) {

	reqId, ok := c.Get("requestId")
	if !ok {
		item.RequestId, _ = uuid.GenerateUUID()
		c.Set("requestId", reqId)
	}

	item.RequestId = reqId.(string)
}

func (log *LogSvc) setUserInfo(c *gin.Context, item *misc.Logger) {
	curUser := auth.AuthRds.GetCurUser(c)
	if curUser.Id != 0 {
		item.UserId = strconv.Itoa(curUser.Id)
		item.Account = curUser.Account
	}
}
