package service

import "errors"

var (
	ErrNotFound = errors.New("tuple not found")
	ErrTimeout  = errors.New("tuple wait timed out")
)
