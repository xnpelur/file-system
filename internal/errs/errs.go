package errs

import "fmt"

var ErrMissingArguments = fmt.Errorf("missing arguments")
var ErrUnknownArguments = fmt.Errorf("unknown arguments")
var ErrIncorrectFileName = fmt.Errorf("incorrect file name")
var ErrUnknownCommand = fmt.Errorf("unknown command")
var ErrRecordAlreadyExists = fmt.Errorf("record with this name already exists")
var ErrIllegalArgument = fmt.Errorf("illegal argument")
var ErrRecordIsNotFile = fmt.Errorf("record is not a file")
var ErrRecordIsNotDirectory = fmt.Errorf("record is not a directory")
var ErrNullNotFound = fmt.Errorf("null terminator not found in file")
var ErrRecordNotFound = fmt.Errorf("record not found")
var ErrIncorrectPassword = fmt.Errorf("incorrect password")
var ErrPermissionDenied = fmt.Errorf("permission denied")
