package app

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type fakeComponent struct {
	name     string
	calls    *[]string
	startErr error
	stopErr  error
	errors   chan error
}

func (f *fakeComponent) Start(context.Context) error {
	*f.calls = append(*f.calls, "start:"+f.name)
	return f.startErr
}

func (f *fakeComponent) Stop(context.Context) error {
	*f.calls = append(*f.calls, "stop:"+f.name)
	return f.stopErr
}

func (f *fakeComponent) Errors() <-chan error { return f.errors }

func TestApplicationStopsComponentsInReverseOrder(t *testing.T) {
	var calls []string
	a := NewApplication(time.Second,
		&fakeComponent{name: "resources", calls: &calls},
		&fakeComponent{name: "worker", calls: &calls},
		&fakeComponent{name: "management", calls: &calls},
		&fakeComponent{name: "public", calls: &calls},
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := a.Run(ctx); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"start:resources", "start:worker", "start:management", "start:public",
		"stop:public", "stop:management", "stop:worker", "stop:resources",
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}

func TestApplicationRollsBackStartedComponents(t *testing.T) {
	var calls []string
	wantErr := errors.New("start failed")
	a := NewApplication(time.Second,
		&fakeComponent{name: "resources", calls: &calls},
		&fakeComponent{name: "worker", calls: &calls, startErr: wantErr},
		&fakeComponent{name: "public", calls: &calls},
	)
	err := a.Run(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want %v", err, wantErr)
	}
	want := []string{"start:resources", "start:worker", "stop:resources"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}

func TestApplicationReturnsComponentRuntimeError(t *testing.T) {
	var calls []string
	errCh := make(chan error, 1)
	wantErr := errors.New("serve failed")
	a := NewApplication(time.Second,
		&fakeComponent{name: "resources", calls: &calls},
		&fakeComponent{name: "public", calls: &calls, errors: errCh},
	)
	errCh <- wantErr
	if err := a.Run(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want %v", err, wantErr)
	}
}
