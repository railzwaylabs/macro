package macro

import "go.uber.org/fx"

// Module is a named dependency-wiring unit. Domain and application packages
// remain ordinary Go; only a module's composition entry point needs Fx.
type Module struct {
	name   string
	option fx.Option
}

// NewModule creates a predictable module registration entry point.
func NewModule(name string, options ...fx.Option) Module {
	return Module{name: name, option: fx.Options(options...)}
}

// With returns the module with additional wiring while preserving its name.
func (m Module) With(options ...fx.Option) Module {
	m.option = fx.Options(m.option, fx.Options(options...))
	return m
}

// WithModules registers generated or application-owned modules with the
// service dependency graph.
func WithModules(modules ...Module) Option {
	return func(options *options) {
		options.modules = append(options.modules, modules...)
	}
}
