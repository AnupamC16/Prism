package model

import (
	"errors"
	"fmt"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed on field '%s': %s", e.Field, e.Message)
}

func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}

func NewNotFoundError(resource, id string) *NotFoundError {
	return &NotFoundError{
		Resource: resource,
		ID:       id,
	}
}

type TokenError struct {
	Message string
}

func (e *TokenError) Error() string {
	return fmt.Sprintf("token error: %s", e.Message)
}

func NewTokenError(message string) *TokenError {
	return &TokenError{
		Message: message,
	}
}

type DRMError struct {
	DRMType string
	Message string
}

func (e *DRMError) Error() string {
	return fmt.Sprintf("drm error [%s]: %s", e.DRMType, e.Message)
}

func NewDRMError(drmType, message string) *DRMError {
	return &DRMError{
		DRMType: drmType,
		Message: message,
	}
}

type CacheError struct {
	Operation string
	Key       string
	Err       error
}

func (e *CacheError) Error() string {
	return fmt.Sprintf("cache %s failed for key '%s': %v", e.Operation, e.Key, e.Err)
}

func (e *CacheError) Unwrap() error {
	return e.Err
}

func NewCacheError(operation, key string, err error) *CacheError {
	return &CacheError{
		Operation: operation,
		Key:       key,
		Err:       err,
	}
}

type ConflictError struct {
	Resource string
	ID       string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("%s already exists: %s", e.Resource, e.ID)
}

func NewConflictError(resource, id string) *ConflictError {
	return &ConflictError{
		Resource: resource,
		ID:       id,
	}
}

type UpstreamError struct {
	Service    string
	StatusCode int
	Message    string
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("upstream service '%s' returned %d: %s", e.Service, e.StatusCode, e.Message)
}

func NewUpstreamError(service string, statusCode int, message string) *UpstreamError {
	return &UpstreamError{
		Service:    service,
		StatusCode: statusCode,
		Message:    message,
	}
}

func IsValidationError(err error) bool {
	var target *ValidationError
	return errors.As(err, &target)
}

func IsNotFoundError(err error) bool {
	var target *NotFoundError
	return errors.As(err, &target)
}

func IsTokenError(err error) bool {
	var target *TokenError
	return errors.As(err, &target)
}

func IsDRMError(err error) bool {
	var target *DRMError
	return errors.As(err, &target)
}

func IsUpstreamError(err error) bool {
	var target *UpstreamError
	return errors.As(err, &target)
}
