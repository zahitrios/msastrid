package horrors

import "net/http"

type RequestError struct {
	err        string
	StatusCode int
}

func (e *RequestError) Error() string {
	return e.err
}

type BadRequestError struct {
	RequestError
}

type NotFoundError struct {
	RequestError
}

func NewBadRequestError(err string) *BadRequestError {
	return &BadRequestError{
		RequestError: RequestError{
			err:        err,
			StatusCode: http.StatusBadRequest,
		},
	}
}

func NewNotFoundError(err string) *NotFoundError {
	return &NotFoundError{
		RequestError: RequestError{
			err:        err,
			StatusCode: http.StatusNotFound,
		},
	}
}
