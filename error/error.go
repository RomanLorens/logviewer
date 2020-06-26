package error

import "fmt"

//Error custom error
type Error struct {
	Message    string `json:"message"`
	StatusCode int    `json:"code"`
}

//Errorf error
func Errorf(status int, format string, args ...interface{}) *Error {
	return &Error{StatusCode: status, Message: fmt.Sprintf(format, args...)}
}

func (e Error) String() string {
	return fmt.Sprintf("%v - [%v]", e.Message, e.StatusCode)
}
