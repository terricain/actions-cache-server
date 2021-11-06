package e

import "errors"

var (
	ErrCacheAlreadyExists = errors.New("cache already exists")
	ErrNoCacheFound       = errors.New("no cache found")
	ErrCacheSizeMismatch  = errors.New("cache size mismatch")
	ErrNotFound           = errors.New("not found")
	ErrNotImplemented     = errors.New("not implemented")
)
