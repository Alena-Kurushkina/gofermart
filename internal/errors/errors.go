package gopherror

import "errors"

// ErrRecordAlreadyExists defines error in case of inserting record which constraint already exists in table
var ErrRecordAlreadyExists = errors.New("DB record already exists")

// ErrTokenInvalid defines error in case of invalid JWT
var ErrTokenInvalid = errors.New("token is not valid")