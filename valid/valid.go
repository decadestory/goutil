package valid

import (
	"errors"

	"github.com/gookit/validate"
	"github.com/gookit/validate/locales/zhcn"
)

type Valid struct{}

var Valids = &Valid{}

func init() {
	zhcn.RegisterGlobal()

	// 自定义全局错误消息
	validate.AddGlobalMessages(map[string]string{
		"required": "{field}为必填项",
	})

	// 更改全局选项
	validate.Config(func(opt *validate.GlobalOption) {
		opt.StopOnError = true
	})
}

func (v *Valid) DoValid(param any) {
	errs := validate.Struct(param).ValidateE()
	if errs.One() != "" {
		panic(errors.New(errs.One()))
	}
}
