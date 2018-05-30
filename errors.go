package vcon

import (
	"context"
	"fmt"
	"time"
)

// ErrorCoder gives errors a code to return with os.Exit
type ErrorCoder interface {
	// Code provides the integer error code to return
	Code() int
}

// ConnectionError occurs when vcon fails to establish a connection to vSphere
// or the user has not provided a valid datacenter or datastore name
type ConnectionError struct{}

func (ce ConnectionError) Error() string {
	return "Failed to connect to vSphere"
}

func (ce ConnectionError) Code() int {
	return 1
}

type NotFoundError struct {
	Path string
}

func (nfe NotFoundError) Error() string {
	return fmt.Sprintf("Failed to find VM identified by '%s'", nfe.Path)
}

func (nfe NotFoundError) Code() int {
	return 2
}

// TimeoutExceededError occurs when a collection of vSphere operations does
// not complete in the determined timeout
type TimeoutExceededError struct {
	timeout time.Duration
}

func (tee TimeoutExceededError) Error() string {
	return fmt.Sprintf("Timed out after %d seconds", tee.timeout)
}

// Cause returns the root cause, which will always be context.DeadlineExceeded
func (tee TimeoutExceededError) Cause() error {
	return context.DeadlineExceeded
}

func (tee TimeoutExceededError) Code() int {
	return 3
}
