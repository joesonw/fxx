package fxx_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/joesonw/fxx"
)

func TestWithConfigFieldFromGroup(t *testing.T) {
	type S struct {
		Value string `json:"value"`
	}

	var s *S
	app := fxtest.New(t,
		fxx.ProvideConfig("a", func(in interface{}) error {
			return json.Unmarshal([]byte(`{
				"s": { "value": "hello world" }
			}`), in)
		}),
		fxx.ExtractConfigField(`json:"s"`, &S{}),
		fx.Invoke(func(ss *S) {
			s = ss
		}))
	defer app.RequireStart().RequireStop()

	assert.NotNil(t, s)
	assert.Equal(t, s.Value, "hello world")
}

func TestWithConfigFieldFromName(t *testing.T) {
	type S struct {
		Value string `json:"value"`
	}

	var s *S
	app := fxtest.New(t,
		fxx.ProvideConfig("a", func(in interface{}) error {
			return json.Unmarshal([]byte(`{
				"s": { "value": "oops" }
			}`), in)
		}),
		fxx.ProvideConfig("b", func(in interface{}) error {
			return json.Unmarshal([]byte(`{
				"s": { "value": "hello world" }
			}`), in)
		}),
		fxx.ProvideConfig("c", func(in interface{}) error {
			return json.Unmarshal([]byte(`{
				"s": { "value": "not ok" }
			}`), in)
		}),
		fxx.ExtractConfigField(`json:"s"`, &S{}, fxx.ExtractConfigFieldFromFile("b")),
		fx.Invoke(func(ss *S) {
			s = ss
		}))
	defer app.RequireStart().RequireStop()

	assert.NotNil(t, s)
	assert.Equal(t, s.Value, "hello world")
}
