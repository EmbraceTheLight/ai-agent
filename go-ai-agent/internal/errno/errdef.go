package errno

// common error
var (
	RequestInvalid = Errno{4000, "请求非法", "请求非法"}
	RequestFailed  = Errno{4001, "请求失败", "请求失败"}
)

// Tool errors
var (
	ErrToolNotFound        = Errno{5002, "tool 未找到", "tool 未找到"}
	ErrToolExecuteFailed   = Errno{5003, "tool 执行失败", "tool 执行失败"}
	ErrToolCallLimitExceed = Errno{5006, "tool 调用次数超过限制", "tool 调用次数超过限制"}

	// time tool 错误
	ErrSetTimezoneFailed = Errno{5001, "设置时区失败", "设置时区失败"}

	// calculation tool 错误
	ErrDivideZeroError = Errno{5004, "除零错误", "除零错误"}

	// http_get tool 错误
	ErrMethodNotSupport      = Errno{5005, "不支持的 HTTP 方法", "不支持的 HTTP 方法"}
	ErrURLNotAllow           = Errno{Code: 5006, Msg: "url 不在允许访问列表中", ErrMsg: "url 不在允许访问列表中"}
	ErrSendHttpRequestFailed = Errno{Code: 5007, Msg: "http 请求出错", ErrMsg: "http 请求出错"}
)
