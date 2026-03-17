package models

import "fmt"

type ErrNotFound struct {
	Resource string
	ID       string
}

func (e *ErrNotFound) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}

type ErrConflict struct {
	Message string
}

func (e *ErrConflict) Error() string {
	return e.Message
}

type ErrValidation struct {
	Message string
}

func (e *ErrValidation) Error() string {
	return e.Message
}
