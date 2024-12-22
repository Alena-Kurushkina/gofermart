package gopherror

import "errors"

// ErrRecordAlreadyExists defines error in case of inserting record which constraint already exists in table
var ErrRecordAlreadyExists = errors.New("DB record already exists")