package errno

import "fmt"

type Errno struct {
	Code   int
	Msg    string
	ErrMsg string
}

func (err Errno) Error() string {
	return err.Msg
}

func (err Errno) WithMsg(msg string) Errno {
	err.Msg = err.Msg + "," + msg
	return err
}

func (err Errno) WithMsgf(format string, msg ...any) Errno {
	err.Msg = err.Msg + "," + fmt.Sprintf(format, msg...)
	return err
}

func (err Errno) WithError(rawError error) Errno {
	var msg string
	if rawError != nil {
		msg = rawError.Error()
	}
	err.ErrMsg = err.Msg + "," + msg
	return err
}

func (err Errno) IsOk() bool {
	return err.Code == 200
}
func (err Errno) NotOk() bool {
	return err.IsOk() == false
}
