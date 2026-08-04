package errno

// common error
var (
	RequestInvalid = Errno{4000, "request invalid", "invalid request"}
)

// Tool errors
var (
	ErrToolNotFound      = Errno{5002, "tool not found", "tool not found"}
	ErrToolExecuteFailed = Errno{5003, "tool execute failed", "tool execute failed"}

	// time tool 错误
	ErrSetTimezoneFailed = Errno{5001, "set time zone failed", "invalid timezone"}

	// calculation tool 错误
	ErrDivideZeroError = Errno{5004, "divide zero error", "divide zero error"}

	// http_get tool 错误
	ErrMethodNotSupport      = Errno{5005, "method not support", "method not support"}
	ErrURLNotAllow           = Errno{Code: 5006, Msg: "this url is not allowed to access", ErrMsg: "this url is not allowed to access"}
	ErrSendHttpRequestFailed = Errno{Code: 5007, Msg: "http request error", ErrMsg: "http request error"}
)
