package handler

import "amiya-eden/internal/apicontract"

// contractOperationFailure carries the existing API business error until the
// typed handler selects the operation's response variant.
type contractOperationFailure struct {
	code    int
	message string
}

func contractBusinessError(code int, message string) apicontract.BusinessError {
	return apicontract.BusinessError{Code: apicontract.BusinessErrorCode(code), Msg: message}
}
