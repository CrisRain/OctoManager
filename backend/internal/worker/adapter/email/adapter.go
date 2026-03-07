package email

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"octomanger/backend/internal/worker/adapter"
)

type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) TypeKey() string {
	return "email"
}

func (a *Adapter) ValidateSpec(spec map[string]any) error {
	if spec == nil {
		return errors.New("spec is required")
	}
	return nil
}

func (a *Adapter) ExecuteAction(_ context.Context, request adapter.ActionRequest) (adapter.Result, error) {
	action := strings.ToUpper(request.Action)
	switch action {
	case "VERIFY":
		return adapter.Result{
			Status: "success",
			Result: map[string]any{
				"verified": true,
			},
		}, nil
	case "SEND":
		return adapter.Result{
			Status: "success",
			Result: map[string]any{
				"message": "queued",
			},
		}, nil
	case "FETCH":
		return adapter.Result{
			Status: "success",
			Result: map[string]any{
				"messages": []any{},
			},
		}, nil
	case "HEALTH_CHECK":
		return adapter.Result{
			Status: "success",
			Result: map[string]any{
				"healthy": true,
			},
		}, nil
	default:
		return adapter.Result{}, fmt.Errorf("unsupported email action: %s", request.Action)
	}
}

