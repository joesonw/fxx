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

type provideConfigFileOptions struct {
	name string
}

type ProvideConfigFileOption func(o *provideConfigFileOptions)

func ProvideConfigFileWithName(name string) ProvideConfigFileOption {
	return func(o *provideConfigFileOptions) {
		o.name = name
	}
}

func provideConfigFileFromDisk(file string, unmarshal func([]byte, interface{}) error, options []ProvideConfigFileOption) fx.Option {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return fx.Error(fmt.Errorf("unable to read file '%s': %w", file, err))
	}
	o := &provideConfigFileOptions{
		name: file,
	}
	for i := range options {
		options[i](o)
	}

	return ProvideConfig(o.name, func(in interface{}) error {
		return unmarshal(b, in)
	})
}

// ProvideJSONConfigFile an helper to ProvideConfigFile for json files, you only need to specify path to file here
func ProvideJSONConfigFile(path string, options ...ProvideConfigFileOption) fx.Option {
	return provideConfigFileFromDisk(path, json.Unmarshal, options)
}

// ProvideYAMLConfigFile an helper to ProvideConfigFile for yaml files, you only need to specify path to file here
func ProvideYAMLConfigFile(path string, options ...ProvideConfigFileOption) fx.Option {
	return provideConfigFileFromDisk(path, yaml.Unmarshal, options)
}

// ProvideConfig inject an unmarshaler with name, which can be specifically used by ExtractConfigFieldFromFile
func ProvideConfig(file string, unmarshal Unmarshal) fx.Option {
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

type extractConfigFieldOptions struct {
	file *string
}

type ExtractConfigFieldOption func(o *extractConfigFieldOptions)

func ExtractConfigFieldFromFile(file string) ExtractConfigFieldOption {
	return func(o *extractConfigFieldOptions) {
		o.file = &file
	}
}

// For example,
//
//  type MySQLConfig struct {
//  	Address  string `json:"address"`
//  	User     string `json:"user"`
//  	Password string `json:"password"`
//  }
//  fx.New(
//    fxx.ProvideJSONConfigFile("/etc/config/myapp.json"),
//    fxx.ExtractConfigField(`json:"mysql"`, &MySQLConfig{}),
//    fx.Provide(func (config *MySQLConfig) (*sql.DB, error) {
//    	return sql.Open("mysql", fmt.Sprintf("mysql://%s:%s@tcp(%s)", config.User, config.Password, config.Address))
//    }),
//  )
//
// ExtractConfigField inject given struct with value from a field of a config file
func ExtractConfigField(tag string, in interface{}, options ...ExtractConfigFieldOption) fx.Option {
	o := &extractConfigFieldOptions{}
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
