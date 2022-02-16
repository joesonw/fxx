package fxx_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/joesonw/fxx"
)

func TestWithAnnotated(t *testing.T) {
	type a struct {
		name string
	}

	type b struct {
		name string
	}

	type c struct {
		name string
	}

	newA := func() *a {
		return &a{name: "foo"}
	}

	newB := func() *b {
		return &b{name: "bar"}
	}

	newC := func() *c {
		return &c{name: "foobar"}
	}

	t.Run("Provided", func(t *testing.T) {
		var inA *a
		var inB *b
		var inC *c
		app := fxtest.New(t,
			fx.Provide(
				fx.Annotated{
					Name:   "foo",
					Target: newA,
				},
				fx.Annotated{
					Group:  "bar",
					Target: newB,
				},
				newC,
			),
			fx.Invoke(fxx.WithAnnotated(fxx.NameAnnotation("foo"), fxx.GroupAnnotation("bar"))(func(aa *a, bb []*b, cc *c) error {
				inA = aa
				inB = bb[0]
				inC = cc
				return nil
			})),
		)
		defer app.RequireStart().RequireStop()
		assert.NotNil(t, inA, "expected a to be injected")
		assert.NotNil(t, inB, "expected b to be injected")
		assert.NotNil(t, inC, "expected c to be injected")
		assert.Equal(t, "foo", inA.name, "expected to get a type 'a' of name 'foo'")
		assert.Equal(t, "bar", inB.name, "expected to get a type 'b' of name 'bar'")
		assert.Equal(t, "foobar", inC.name, "expected to get a type 'c' of name 'foobar'")
	})
}

func TestWithAnnotatedError(t *testing.T) {
	type a struct {
		name string
	}

	newA := func() *a {
		return &a{name: "foo"}
	}

	t.Run("Provided", func(t *testing.T) {
		app := fx.New(
			fx.Logger(fxtest.NewTestPrinter(t)),
			fx.Provide(
				fx.Annotated{
					Name:   "foo",
					Target: newA,
				},
			),
			fx.Invoke(fxx.WithAnnotated(fxx.NameAnnotation("foo"))("")),
		)
		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "WithAnnotated returned function must be called with a function")
	})
}

func TestWithAnnotatedOptional(t *testing.T) {
	type a struct {
	}

	t.Run("Not Provided", func(t *testing.T) {
		app := fx.New(
			fx.Logger(fxtest.NewTestPrinter(t)),
			fx.Invoke(fxx.WithAnnotated(fxx.NameAnnotation("foo"))(func(aa *a) error {
				return nil
			})),
		)
		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing dependencies for function")
	})

	t.Run("Optional", func(t *testing.T) {
		app := fx.New(
			fx.Logger(fxtest.NewTestPrinter(t)),
			fx.Invoke(fxx.WithAnnotated(fxx.NameAnnotation("foo").IsOptional())(func(aa *a) error {
				if aa != nil {
					return fmt.Errorf("not nil")
				}
				return nil
			})),
		)
		err := app.Err()
		assert.Nil(t, err)
	})
}
