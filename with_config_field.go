package fxx

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"

	yaml "gopkg.in/yaml.v2"

	"github.com/joesonw/fxx/reflectutil"
	"go.uber.org/fx"
)

type Unmarshal func(interface{}) error

const configFileAnnotationPrefix = "github.com/joesonw/fxx.ProvideConfigFile/"
const configFileGroup = "github.com/joesonw/fxx.ProvideConfigFile"

func provideConfigFileFromDisk(file string, unmarshal func([]byte, interface{}) error) fx.Option {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return fx.Error(fmt.Errorf("unable to read file '%s': %w", file, err))
	}

	return ProvideConfigFile(file, func(in interface{}) error {
		return unmarshal(b, in)
	})
}

func ProvideJSONConfigFile(file string) fx.Option {
	return provideConfigFileFromDisk(file, json.Unmarshal)
}

func ProvideYAMLConfigFile(file string) fx.Option {
	return provideConfigFileFromDisk(file, yaml.Unmarshal)
}

func ProvideConfigFile(file string, unmarshal Unmarshal) fx.Option {
	return fx.Options(
		fx.Provide(fx.Annotated{
			Name: configFileAnnotationPrefix + file,
			Target: func() Unmarshal {
				return unmarshal
			},
		}),
		fx.Provide(fx.Annotated{
			Group: configFileGroup,
			Target: func() Unmarshal {
				return unmarshal
			},
		}),
	)
}

type withConfigFieldOptions struct {
	file *string
}

type WithConfigFieldOption func(o *withConfigFieldOptions)

func WithConfigFieldFromFile(file string) WithConfigFieldOption {
	return func(o *withConfigFieldOptions) {
		o.file = &file
	}
}

func WithConfigField(tag string, in interface{}, options ...WithConfigFieldOption) fx.Option {
	o := &withConfigFieldOptions{}
	for i := range options {
		options[i](o)
	}

	returnType := reflect.TypeOf(in)
	var funcTypeIns []reflect.Type
	var annotations []Annotation
	if o.file != nil {
		funcTypeIns = append(funcTypeIns, reflect.TypeOf((*Unmarshal)(nil)).Elem())
		annotations = append(annotations, NameAnnotation(configFileAnnotationPrefix+*o.file))
	} else {
		funcTypeIns = append(funcTypeIns, reflect.TypeOf((*[]Unmarshal)(nil)).Elem())
		annotations = append(annotations, GroupAnnotation(configFileGroup))
	}

	funcType := reflect.FuncOf(funcTypeIns, []reflect.Type{returnType, reflectutil.ErrorType()}, false)
	funcValue := reflect.MakeFunc(funcType, func(args []reflect.Value) (results []reflect.Value) {
		var unmarshal Unmarshal
		if o.file != nil {
			unmarshal = args[0].Interface().(Unmarshal)
		} else {
			a := args[0].Interface().([]Unmarshal)
			if len(a) == 0 {
				return []reflect.Value{
					reflect.Zero(returnType),
					reflect.ValueOf(fmt.Errorf("no config files were provided")),
				}
			}
			unmarshal = a[len(a)-1]
		}

		s := reflect.StructOf([]reflect.StructField{{
			Name: "Config",
			Type: returnType,
			Tag:  reflect.StructTag(tag),
		}})

		val := reflect.New(s)
		if err := unmarshal(val.Interface()); err != nil {
			return []reflect.Value{
				reflect.Zero(returnType),
				reflect.ValueOf(fmt.Errorf("unable to unmarshal config file: %w", err)),
			}
		}
		return []reflect.Value{val.Elem().FieldByName("Config"), reflectutil.NilErrorValue()}
	})

	return fx.Provide(WithAnnotated(annotations...)(funcValue.Interface()))
}
