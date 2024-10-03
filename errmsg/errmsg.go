package errmsg

import (
	"errors"
	"fmt"
	"strings"
)

// endUserError is an error that holds an end-user message.
type endUserError struct {
	message string
	cause   error
}

// Error implements the error interface by returning the internal error message.
func (e *endUserError) Error() string { return e.cause.Error() }

// Unwrap implements the Unwrap interface.
func (e *endUserError) Unwrap() error { return e.cause }

// As implements the required function to ensure errors.As can properly match
// endUserError targets.
func (e *endUserError) As(target any) bool {
	if tgt, ok := target.(**endUserError); ok {
		*tgt = e
		return true
	}

	return false
}

// AddMessage adds an end-user message to an error, prepending it to any
// potential existing message.
func AddMessage(err error, msg string) error {
	if msgInCause := Message(err); msgInCause != "" {
		msg = fmt.Sprintf("%s %s", msg, msgInCause)
	}

	return &endUserError{
		cause:   err,
		message: msg,
	}
}

// Message extracts an end-user message from the error.
func Message(err error) string {
	for err != nil {
		if endUserErr, ok := err.(*endUserError); ok {
			return endUserErr.message
		}

		// If the error was generated through errors.Join, Unwrap returns an
		// array of errors, several of which might contain a message.
		if joinedErr, ok := err.(interface{ Unwrap() []error }); ok {
			unwrappedErrs := joinedErr.Unwrap()
			msgs := make([]string, 0, len(unwrappedErrs))
			for _, uwe := range unwrappedErrs {
				msg := Message(uwe)
				if msg != "" {
					msgs = append(msgs, Message(uwe))
				}
			}

			return strings.Join(msgs, " ")
		}

		err = errors.Unwrap(err)
	}

	return ""
}

// MessageOrErr extracts an end-user message from the error. If no message is
// found, err.Error() is returned.
func MessageOrErr(err error) string {
	msg := Message(err)
	if msg == "" {
		return err.Error()
	}

	return msg
}
