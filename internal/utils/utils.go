package utils

import (
	"errors"
	"reflect"
)

func CalculateStructSize(s any) (int, error) {
	v := reflect.ValueOf(s)
	if v.Kind() != reflect.Struct {
		return 0, errors.New("input is not a struct")
	}

	size := 0
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldSize := int(reflect.TypeOf(field.Interface()).Size())
		size += fieldSize
	}

	return size, nil
}
