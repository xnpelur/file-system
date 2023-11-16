package utils

import (
	"errors"
	"reflect"
	"strings"
)

func CalculateStructSize(s any) (uint32, error) {
	v := reflect.ValueOf(s)
	if v.Kind() != reflect.Struct {
		return 0, errors.New("input is not a struct")
	}

	size := uint32(0)
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldSize := uint32(reflect.TypeOf(field.Interface()).Size())
		size += fieldSize
	}

	return size, nil
}

func StringToByteBlock(str string, blockSize uint32) []byte {
	data := make([]byte, blockSize)
	stringBytes := []byte(str)
	copy(data, stringBytes)
	return data
}

func ChangeDirectoryPath(currentPath, arg string) string {
	if strings.HasPrefix(arg, "/") {
		return arg
	}

	currentPath = strings.Trim(currentPath, "/")
	currentDirs := strings.Split(currentPath, "/")

	if currentDirs[0] == "" {
		currentDirs = currentDirs[1:]
	}

	switch arg {
	case ".":
	case "..":
		if len(currentDirs) > 0 {
			currentDirs = currentDirs[:len(currentDirs)-1]
		}
	default:
		currentDirs = append(currentDirs, arg)
	}

	if len(currentDirs) == 0 {
		return "/"
	}

	return "/" + strings.Join(currentDirs, "/")
}

func SplitPath(input string) (string, string) {
	index := strings.LastIndex(input, "/")

	if index == -1 {
		return "", input
	}

	firstPart := input[:index]
	secondPart := input[index+1:]

	return firstPart, secondPart
}
