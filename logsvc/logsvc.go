package logsvc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"strconv"
	"time"

	"github.com/IBM/sarama"
	"github.com/decadestory/goutil/auth"
	"github.com/decadestory/goutil/conf"
	"github.com/decadestory/goutil/logger"
	"github.com/decadestory/goutil/misc"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/go-uuid"
)

var serviceId string
var client sarama.SyncProducer

const maxLogSize = 2 * 1024 * 1024      // 只保留前 2MB
const truncatedMark = "... [TRUNCATED]" // 只保留前 2MB

type LogSvc struct{}

var LoggerSvc = LogSvc{}

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
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.NoResponse
	config.Producer.Partitioner = sarama.NewHashPartitioner
	config.Producer.Retry.Max = 0
	config.Producer.Compression = sarama.CompressionSnappy
	config.Producer.Timeout = time.Second * 1
	config.Producer.Return.Successes = true
	newClient, err := sarama.NewSyncProducer([]string{host}, config)
	logger.Logger.Error(err)
	client = newClient
}

func KafkaProducer(msgTxt string) {
	msg := &sarama.ProducerMessage{}
	msg.Topic = "sme-logger"
	msg.Value = sarama.StringEncoder(msgTxt)
	_, _, err := client.SendMessage(msg)
	logger.Logger.Error(err)
}

// Info args[0] LogExtTxt,args[1] Duration
func Info(msg string, args ...interface{}) {
	var ext string = ""
	var dur int64 = 0
	if len(args) > 0 {
		ext = args[0].(string)
	}
	if len(args) > 1 {
		dur = args[1].(int64)
	}

	log := misc.Logger{
		ServiceId:  serviceId,
		RequestId:  "",
		Ip:         misc.GetIp(),
		Path:       getStack(),
		LogType:    "debug",
		LogLevel:   1,
		UserId:     "",
		Account:    "",
		LogTxt:     msg,
		LogExtTxt:  ext,
		Duration:   dur,
		CreateTime: time.Now().Format(misc.FMTMillSEC),
	}
	m, _ := json.Marshal(log)
	KafkaProducer(string(m))
}

// Debug args[0] LogExtTxt,args[1] Duration
func Debug(msg string, args ...interface{}) {
	var ext string = ""
	var dur int64 = 0
	if len(args) > 0 {
		ext = args[0].(string)
	}
	if len(args) > 1 {
		dur = args[1].(int64)
	}
	log := misc.Logger{
		ServiceId:  serviceId,
		RequestId:  "",
		Ip:         misc.GetIp(),
		Path:       getStack(),
		LogType:    "debug",
		LogLevel:   2,
		UserId:     "",
		Account:    "",
		LogTxt:     msg,
		LogExtTxt:  ext,
		Duration:   dur,
		CreateTime: time.Now().Format(misc.FMTMillSEC),
	}
	m, _ := json.Marshal(log)
	KafkaProducer(string(m))
}

// Error args[0] LogExtTxt,args[1] Duration
func Error(msg string, args ...interface{}) {
	var ext string = ""
	var dur int64 = 0
	if len(args) > 0 {
		ext = args[0].(string)
	}
	if len(args) > 1 {
		dur = args[1].(int64)
	}
	log := misc.Logger{
		ServiceId:  serviceId,
		RequestId:  "",
		Ip:         misc.GetIp(),
		Path:       getStack(),
		LogType:    "exception",
		LogLevel:   4,
		UserId:     "",
		Account:    "",
		LogTxt:     msg,
		LogExtTxt:  ext,
		Duration:   dur,
		CreateTime: time.Now().Format(misc.FMTMillSEC),
	}
	m, _ := json.Marshal(log)
	KafkaProducer(string(m))
}

// Error args[0] LogExtTxt,args[1] Duration
func LogApi(c *gin.Context, reqId string, dur int64, reqData, repData string) {
	log := misc.Logger{
		ServiceId:  serviceId,
		RequestId:  reqId,
		Ip:         misc.GetIp(),
		Path:       c.FullPath(),
		LogType:    "api",
		LogLevel:   1,
		UserId:     "",
		Account:    "",
		LogTxt:     reqData,
		LogExtTxt:  repData,
		Duration:   dur,
		CreateTime: time.Now().Format(misc.FMTMillSEC),
	}

	curUser := auth.AuthRds.GetCurUser(c)
	if curUser.Id != 0 {
		log.UserId = strconv.Itoa(curUser.Id)
		log.Account = curUser.Account
	}

	m, _ := json.Marshal(log)
	KafkaProducer(string(m))
}

func LogApiMidware(c *gin.Context) {

	reqId := c.GetHeader("requestId")
	if reqId == "" {
		reqId, _ = uuid.GenerateUUID()
	}

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

	LogApi(c, reqId, time.Since(st).Milliseconds(), string(reqData), writer.body.String())
}

func getStack() (funcs string) {
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
