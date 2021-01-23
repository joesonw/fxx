package reflectutil

import "reflect"

func NilErrorValue() reflect.Value {
	return reflect.Zero(ErrorType())
}

func ErrorType() reflect.Type {
	return reflect.TypeOf((*error)(nil)).Elem()
}
