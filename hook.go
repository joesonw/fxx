package fxx

import (
	"context"

	"go.uber.org/fx"
)

type FxHook interface {
	FxHookOnStart
	FxHookOnStop
}

type FxHookOnStart interface {
	OnStart(context.Context) error
}

type FxHookOnStop interface {
	OnStop(context.Context) error
}

func Hook(hook FxHook) fx.Hook {
	return fx.Hook{
		OnStart: hook.OnStart,
		OnStop:  hook.OnStop,
	}
}

func HookStart(hook FxHookOnStart) fx.Hook {
	return fx.Hook{
		OnStart: hook.OnStart,
		OnStop: func(ctx context.Context) error {
			return nil
		},
	}
}

func HookStop(hook FxHookOnStop) fx.Hook {
	return fx.Hook{
		OnStart: func(ctx context.Context) error {
			return nil
		},
		OnStop: hook.OnStop,
	}
}
