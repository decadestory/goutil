package valid

import (
	"errors"
	"strings"
	"unicode"

	"github.com/duke-git/lancet/v2/validator"
	"github.com/gookit/validate/v2"
	"github.com/gookit/validate/v2/locales/zhcn"
)

type valid struct{}

var Valids = &valid{}

func init() {
	zhcn.RegisterGlobal()

	// 自定义全局错误消息
	validate.AddGlobalMessages(map[string]string{
		"required": "{field}为必填项",
		"mobile":   "{field}格式不正确",
		"email":    "{field}格式不正确",
		"date":     "{field}格式不正确",
		"url":      "{field}格式不正确",
		"ip":       "{field}格式不正确",
		"json":     "{field}格式不正确",
		"space":    "{field}不能存在空格",
		"pwd":      "{field}密码必须至少6位，包含大小字母和数字",
		"idcard":   "{field}格式不正确",
	})

	// 更改全局选项
	validate.Config(func(opt *validate.GlobalOption) {
		opt.StopOnError = true
	})

	//验证密码 密码必须至少6位，包含大小字母和数字
	validate.AddValidator("pwd", func(val any) bool {
		pwd := val.(string)
		if len(pwd) < 6 {
			return false
		}
		var hasUpper, hasLower, hasDigit bool
		for _, ch := range pwd {
			switch {
			case unicode.IsUpper(ch):
				hasUpper = true
			case unicode.IsLower(ch):
				hasLower = true
			case unicode.IsDigit(ch):
				hasDigit = true
			}
		}
		return hasUpper && hasLower && hasDigit
	})

	validate.AddValidator("idcard", func(val any) bool {
		return validator.IsChineseIdNum(val.(string))
	})

	validate.AddValidator("mobile", func(val any) bool {
		return validator.IsChineseMobile(val.(string))
	})

	validate.AddValidator("space", func(val any) bool {
		return !strings.Contains(val.(string), " ")
	})
}

func (v *valid) DoValid(param any) {
	errs := validate.Struct(param).ValidateE()
	if errs.One() != "" {
		panic(errors.New(errs.One()))
	}
}

func (v *valid) DoValidErr(param any) error {
	errs := validate.Struct(param).ValidateE()
	if errs.One() != "" {
		return errors.New(errs.One())
	}
	return nil
}
