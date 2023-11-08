package utils

import (
	"errors"
	"reflect"
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
