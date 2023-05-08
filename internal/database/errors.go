package database

import "errors"

var ErrNoSuchPath = errors.New("no such path")
var ErrDuplicateKey = errors.New("duplicate key")
var ErrNoSuchRecord = errors.New("no such record")
var ErrNoSuchPool = errors.New("no such pool")
var ErrBadRouteType = errors.New("bad route type")
