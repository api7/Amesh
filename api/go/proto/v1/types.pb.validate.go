// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: proto/v1/types.proto

package v1

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"google.golang.org/protobuf/types/known/anypb"
)

// ensure the imports are used
var (
	_ = bytes.MinRead
	_ = errors.New("")
	_ = fmt.Print
	_ = utf8.UTFMax
	_ = (*regexp.Regexp)(nil)
	_ = (*strings.Reader)(nil)
	_ = net.IPv4len
	_ = time.Duration(0)
	_ = (*url.URL)(nil)
	_ = (*mail.Address)(nil)
	_ = anypb.Any{}
)

// Validate checks the field values on Plugin with the rules defined in the
// proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *Plugin) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on Plugin with the rules defined in the
// proto definition for this message. If any rules are violated, the result is
// a list of violation errors wrapped in PluginMultiError, or nil if none found.
func (m *Plugin) ValidateAll() error {
	return m.validate(true)
}

func (m *Plugin) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if _, ok := _Plugin_Type_InLookup[m.GetType()]; !ok {
		err := PluginValidationError{
			field:  "Type",
			reason: "value must be in list [pre-req post-req]",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if utf8.RuneCountInString(m.GetName()) < 1 {
		err := PluginValidationError{
			field:  "Name",
			reason: "value length must be at least 1 runes",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if utf8.RuneCountInString(m.GetConfig()) < 2 {
		err := PluginValidationError{
			field:  "Config",
			reason: "value length must be at least 2 runes",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return PluginMultiError(errors)
	}
	return nil
}

// PluginMultiError is an error wrapping multiple validation errors returned by
// Plugin.ValidateAll() if the designated constraints aren't met.
type PluginMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m PluginMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m PluginMultiError) AllErrors() []error { return m }

// PluginValidationError is the validation error returned by Plugin.Validate if
// the designated constraints aren't met.
type PluginValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e PluginValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e PluginValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e PluginValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e PluginValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e PluginValidationError) ErrorName() string { return "PluginValidationError" }

// Error satisfies the builtin error interface
func (e PluginValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sPlugin.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = PluginValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = PluginValidationError{}

var _Plugin_Type_InLookup = map[string]struct{}{
	"pre-req":  {},
	"post-req": {},
}
