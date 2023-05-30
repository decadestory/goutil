package goutil

type Enum struct{}

var Enums = &Enum{}

// 获取性别中文
func (em *Enum) GetGenderStr(v int) string {
	switch v {
	case 1:
		return "男"
	case 2:
		return "女"
	default:
		return "未知"
	}
}

// 获取状态中文
func (em *Enum) GetStatusStr(v int) string {
	switch v {
	case 0:
		return "停止"
	case 1:
		return "正常"
	default:
		return "未知"
	}
}

// 获取状态中文
func (em *Enum) GetStatusLogStr(v int) string {
	switch v {
	case 0:
		return "执行中"
	case 1:
		return "成功"
	case 2:
		return "失败"
	default:
		return "未知"
	}
}

// 获取状态中文
func (em *Enum) GetIsValidStr(v bool) string {
	switch v {
	case true:
		return "启用"
	case false:
		return "禁用"
	default:
		return "未知"
	}
}

// 获取状态中文
func (em *Enum) GetTypeStr(v int) string {
	switch v {
	case 1:
		return "HTTP"
	case 2:
		return "SQL"
	default:
		return "未知"
	}
}
