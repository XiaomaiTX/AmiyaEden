package service

// UserVisibleError 表示可以安全返回给前端的业务错误信息。
type UserVisibleError struct {
	message string
}

func (e *UserVisibleError) Error() string {
	return e.message
}

func NewUserVisibleError(message string) error {
	return &UserVisibleError{message: message}
}

func IsUserVisibleError(err error) bool {
	_, ok := err.(*UserVisibleError)
	return ok
}
