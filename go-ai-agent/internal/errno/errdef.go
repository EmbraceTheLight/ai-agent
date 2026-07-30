package errno

// common error
var (
	RequestInvalid = Errno{4000, "request invalid", "invalid request"}
)

// Tool errors
var (
	ErrSetTimezoneFailed = Errno{5001, "set time zone failed", "invalid timezone"}
)
