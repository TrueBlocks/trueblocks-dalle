package dalle

import (
	"errors"
	"fmt"
)

type ErrorCode string

const (
	ErrInvalidInput               ErrorCode = "invalid_input"
	ErrSeriesNotFound             ErrorCode = "series_not_found"
	ErrSeriesInvalid              ErrorCode = "series_invalid"
	ErrDatabaseManifestInvalid    ErrorCode = "database_manifest_invalid"
	ErrDatabaseVersionUnavailable ErrorCode = "database_version_unavailable"
	ErrDatabaseHashMismatch       ErrorCode = "database_hash_mismatch"
	ErrRegenerationRefused        ErrorCode = "regeneration_refused"
	ErrMetadataInvalid            ErrorCode = "metadata_invalid"
	ErrArtifactMissing            ErrorCode = "artifact_missing"
	ErrProviderUnavailable        ErrorCode = "provider_unavailable"
	ErrProviderFailed             ErrorCode = "provider_failed"
)

type Error struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func NewError(code ErrorCode, message string) *Error {
	return &Error{Code: code, Message: message}
}

func WrapError(code ErrorCode, message string, err error) *Error {
	return &Error{Code: code, Message: message, Err: err}
}

func ErrorCodeOf(err error) ErrorCode {
	if err == nil {
		return ""
	}
	var coded codedError
	if errors.As(err, &coded) {
		return coded.GetCode()
	}
	return ""
}

type codedError interface {
	GetCode() ErrorCode
}

func (e *Error) GetCode() ErrorCode {
	if e == nil {
		return ""
	}
	return e.Code
}
