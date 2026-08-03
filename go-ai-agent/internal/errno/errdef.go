package errno

// common error
var (
	RequestInvalid = Errno{4000, "request invalid", "invalid request"}
)

// Tool errors
var (
	ErrSetTimezoneFailed = Errno{5001, "set time zone failed", "invalid timezone"}
	ErrToolNotFound      = Errno{5002, "tool not found", "tool not found"}
	ErrToolExecuteFailed = Errno{5003, "tool execute failed", "tool execute failed"}
	ErrDivideZeroError   = Errno{5004, "divide zero error", "divide zero error"}
)
