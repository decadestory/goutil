package lang

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/decadestory/goutil/conf"
	"github.com/decadestory/goutil/exception"
	"github.com/redis/go-redis/v9"
)

type lang struct{}

var Langs = &lang{}

var rctx = context.Background()

var Langdb *redis.Client

func init() {
	defer exception.Errors.DeferRecover()
	host := conf.Configs.GetString("lang.host")
	pwd := conf.Configs.GetString("lang.pwd")
	dbId := conf.Configs.GetString("lang.db")

	dbNo, err := strconv.Atoi(dbId)
	exception.Errors.CheckErr(err)
	Langdb = redis.NewClient(&redis.Options{
		Addr:     host,
		Password: pwd,
		DB:       dbNo,
	})
}

func (l *lang) CodeTip(code, lang string) string {
	res, _ := Langdb.HGet(rctx, code, lang).Result()
	return strings.Trim(res, "\"")
}

func (l *lang) CnTip(code string) string {
	res, _ := Langdb.HGet(rctx, code, "cn").Result()
	return strings.Trim(res, "\"")
}

func (l *lang) ConvertTip(msg, lang string) string {
	regex := regexp.MustCompile(`#[a-zA-Z0-9]*#`)
	codeWrap := regex.FindString(msg)

	if lang == "" {
		return strings.Replace(msg, codeWrap, "", -1)
	}

	code := strings.Trim(codeWrap, "#")
	res, _ := Langdb.HGet(rctx, code, lang).Result()
	return strings.Trim(res, "\"")
}

// 添加123#ONB#
