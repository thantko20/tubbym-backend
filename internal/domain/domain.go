package domain

import "fmt"

const ErrInternalServer = 9999

type Error struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
	Action  string `json:"action"`
	Err     error  `json:"-"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("Code: %d, Action: %s, Err: %v", e.Code, e.Action, e.Err)
}
