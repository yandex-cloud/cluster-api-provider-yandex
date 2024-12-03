package ycerrors

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IsNotFound reports whether err is a YandexCloud API error
// with GRPC NOT_FOUND code.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	s, ok := status.FromError(err)
	return ok && s.Code() == codes.NotFound
}
