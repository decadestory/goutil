package misc

import "time"

type Option struct {
	Id    int32  `json:"id"`
	Key   string `json:"key"`
	Code  string `json:"code"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type BaseDbModel struct {
	AddTime    time.Time `gorm:"column:add_time;comment:添加时间" json:"addTime"`
	AddUserId  int       `gorm:"column:add_user_id;comment:添加人;size:100" json:"addUserId"`
	EditTime   time.Time `gorm:"column:edit_time;comment:编辑时间" json:"editTime"`
	EditUserId int       `gorm:"column:edit_user_id;comment:编辑人;size:100" json:"editUserId"`
	IsValid    bool      `gorm:"column:is_valid;comment:是否删除" json:"isValid"`
}

type BaseDtoModel struct {
	KeyWord     string `gorm:"-" json:"keyWord"`
	PageIndex   int    `gorm:"-" json:"pageIndex"`
	PageSize    int    `gorm:"-" json:"pageSize"`
	AddTimeStr  string `gorm:"-" json:"addTimeStr"`
	EditTimeStr string `gorm:"-" json:"editTimeStr"`
	IsValidStr  string `gorm:"-" json:"isValidStr"`
}

type Logger struct {
	RequestId  string `json:"requestId"`
	ServiceId  string `json:"serviceId"`
	Ip         string `json:"ip"`
	Path       string `json:"path"`
	LogType    string `json:"logType"`
	LogLevel   int    `json:"logLevel"`
	UserId     string `json:"userId"`
	Account    string `json:"account"`
	LogTxt     string `json:"logTxt"`
	LogExtTxt  string `json:"logExtTxt"`
	Duration   int64  `json:"duration"`
	CreateTime string `json:"createTime"`
}
