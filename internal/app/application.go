package app

import (
	"context"
	"errors"
	"time"

	"jimu/internal/contract"
)

type Application struct {
	shutdownTimeout time.Duration
	components      []contract.Component
}

func NewApplication(shutdownTimeout time.Duration, components ...contract.Component) *Application {
	if shutdownTimeout <= 0 {
		shutdownTimeout = 30 * time.Second
	}
	return &Application{shutdownTimeout: shutdownTimeout, components: components}
}

func (a *Application) Run(ctx context.Context) error {
	started := make([]contract.Component, 0, len(a.components))
	for _, component := range a.components {
		if err := component.Start(ctx); err != nil {
			return errors.Join(err, a.stop(started))
		}
		started = append(started, component)
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	errCh := make(chan error, len(started))
	for _, component := range started {
		source, ok := component.(contract.ErrorSource)
		if !ok || source.Errors() == nil {
			continue
		}
		go forwardError(runCtx, errCh, source.Errors())
	}

	var runErr error
	select {
	case <-ctx.Done():
	case runErr = <-errCh:
	}
	cancel()
	return errors.Join(runErr, a.stop(started))
}

func forwardError(ctx context.Context, target chan<- error, source <-chan error) {
	select {
	case err, ok := <-source:
		if ok && err != nil {
			target <- err
		}
	case <-ctx.Done():
	}
}

func (a *Application) stop(started []contract.Component) error {
	ctx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
	defer cancel()
	var result error
	for i := len(started) - 1; i >= 0; i-- {
		result = errors.Join(result, started[i].Stop(ctx))
	}
	return result
}
