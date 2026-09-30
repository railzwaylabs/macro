package macro

import (
	"context"
	"fmt"
	"net/http"

	"go.uber.org/fx"
	"google.golang.org/grpc"
)

type wiringRuntime struct {
	app     *fx.App
	initErr error
}

func newWiringRuntime(modules []Module, grpcServer *grpc.Server, httpServer *http.Server) *wiringRuntime {
	seen := make(map[string]struct{}, len(modules))
	fxOptions := []fx.Option{fx.NopLogger}
	if grpcServer != nil {
		fxOptions = append(fxOptions, fx.Supply(grpcServer))
	}
	if httpServer != nil {
		fxOptions = append(fxOptions, fx.Supply(httpServer))
	}
	for _, module := range modules {
		if module.name == "" {
			return &wiringRuntime{initErr: fmt.Errorf("macro: module name is required")}
		}
		if _, exists := seen[module.name]; exists {
			return &wiringRuntime{initErr: fmt.Errorf("macro: module %q is registered more than once", module.name)}
		}
		seen[module.name] = struct{}{}
		fxOptions = append(fxOptions, module.option)
	}
	app := fx.New(fxOptions...)
	if err := app.Err(); err != nil {
		return &wiringRuntime{app: app, initErr: fmt.Errorf("macro: validate dependency wiring: %w", err)}
	}
	return &wiringRuntime{app: app}
}

func (runtime *wiringRuntime) Start(ctx context.Context) error {
	if runtime.initErr != nil {
		return runtime.initErr
	}
	if err := runtime.app.Start(ctx); err != nil {
		return fmt.Errorf("start dependency wiring: %w", err)
	}
	return nil
}

func (runtime *wiringRuntime) Stop(ctx context.Context) error {
	if err := runtime.app.Stop(ctx); err != nil {
		return fmt.Errorf("stop dependency wiring: %w", err)
	}
	return nil
}
