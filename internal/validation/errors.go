package validation

type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func NewError(code, message string) *Error {
	return &Error{Code: code, Message: message}
}

func newError(code, message string) *Error {
	return NewError(code, message)
}
