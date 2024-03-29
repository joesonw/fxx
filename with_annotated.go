package fxx

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"go.uber.org/fx"
)

// Annotation this will be passed to WithAnnotated to identify which to be injected
type Annotation interface {
	isAnnotation()
	IsOptional() Annotation
}

type groupAnnotation struct {
	group    string
	optional bool
}

func (groupAnnotation) isAnnotation() {}

func (a groupAnnotation) IsOptional() Annotation {
	return groupAnnotation{
		group:    a.group,
		optional: true,
	}
}

// GroupAnnotation use group Annotated inject
func GroupAnnotation(group string) Annotation {
	return groupAnnotation{
		group: group,
	}
}

type nameAnnotation struct {
	name     string
	optional bool
}

func (nameAnnotation) isAnnotation() {}

func (a nameAnnotation) IsOptional() Annotation {
	return nameAnnotation{
		name:     a.name,
		optional: true,
	}
}

// NameAnnotation use name Annotated inject
func NameAnnotation(name string) Annotation {
	return nameAnnotation{
		name: name,
	}
}

// WithAnnotated allows to inject annotated options without declare your own struct
//
// For example,
//
//   func NewReadOnlyConnection(...) (*Connection, error)
//   fx.Provide(fx.Annotated{
//     Name: "ro",
//     Target: NewReadOnlyConnection,
//   })
//   fx.Supply(&Server{})
//
//   fx.Invoke(fx.WithAnnotated(fx.NameAnnotation("ro)(func(roConn *Connection, s *Server) error {
//   })
//
// Is equivalent to,
//
//   type Params struct {
//     fx.In
//
//     Connection *Connection `name:"ro"`
//     Server *Server
//   }
//
//   fx.Invoke(func(params Params) error {
//      roConn := params.Connection
//      s := params.Server
//      return nil
//   })
//
// WithAnnotated takes an array of names, and returns function to be called with user function. names are in order.
func WithAnnotated(annos ...Annotation) func(interface{}) interface{} {
	numNames := len(annos)
	return func(f interface{}) interface{} {
		userFunc := reflect.ValueOf(f)
		userFuncType := userFunc.Type()
		if userFuncType.Kind() != reflect.Func {
			return func() error {
				return errors.New("WithAnnotated returned function must be called with a function")
			}
		}
		numArgs := userFuncType.NumIn()
		digInStructFields := []reflect.StructField{{
			Name:      "In",
			Anonymous: true,
			Type:      reflect.TypeOf(fx.In{}),
		}}
		for i := 0; i < numArgs; i++ {
			name := fmt.Sprintf("Field%d", i)
			field := reflect.StructField{
				Name: name,
				Type: userFuncType.In(i),
			}
			
			if i < numNames { // namedArguments
				var tags []string
				optional := false
				annos[i].isAnnotation()
				switch anno := annos[i].(type) {
				case groupAnnotation:
					tags = append(tags, fmt.Sprintf(`group:"%s"`, anno.group))
					optional = anno.optional
				case nameAnnotation:
					tags = append(tags, fmt.Sprintf(`name:"%s"`, anno.name))
					optional = anno.optional
				}

				if optional {
					tags = append(tags, `optional:"true"`)
				}

				field.Tag = reflect.StructTag(strings.Join(tags, " "))
			}
			digInStructFields = append(digInStructFields, field)
		}

		outs := make([]reflect.Type, userFuncType.NumOut())
		for i := 0; i < userFuncType.NumOut(); i++ {
			outs[i] = userFuncType.Out(i)
		}

		paramType := reflect.StructOf(digInStructFields)
		fxOptionFuncType := reflect.FuncOf([]reflect.Type{paramType}, outs, false)
		fxOptionFunc := reflect.MakeFunc(fxOptionFuncType, func(args []reflect.Value) []reflect.Value {
			callUserFuncINs := make([]reflect.Value, numArgs)
			params := args[0]
			for i := 0; i < numArgs; i++ {
				callUserFuncINs[i] = params.Field(i + 1)
			}
			return userFunc.Call(callUserFuncINs)
		})

		return fxOptionFunc.Interface()
	}
}
