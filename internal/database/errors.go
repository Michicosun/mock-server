package database

import "errors"

var ErrNoSuchPath = errors.New("no such path")
var ErrDuplicateKey = errors.New("duplicate key")
