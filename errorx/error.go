package errorx

// 自定义错误码
const (
	CodeInternal Code = 1000 + iota
	CodeConfig
	CodeLogic
	CodeMysql
	CodeRedis
)

type Code int

// ErrorX 自定义错误
type ErrorX struct {
	Code Code   `json:"error_code"`
	Msg  string `json:"error_message"`
}

func New(code Code, msg string) *ErrorX {
	return &ErrorX{
		Code: code,
		Msg:  msg,
	}
}

func (e *ErrorX) Error() string {
	return e.Msg
}
