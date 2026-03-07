package generic

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"octomanger/backend/internal/worker/adapter"
	"octomanger/backend/internal/worker/bridge"
)

type Adapter struct {
	typeKey string
	bridge  bridge.PythonBridge
}

func New(typeKey string, bridge bridge.PythonBridge) *Adapter {
	return &Adapter{
		typeKey: typeKey,
		bridge:  bridge,
	}
}

func (a *Adapter) TypeKey() string {
	return a.typeKey
}

func (a *Adapter) ValidateSpec(spec map[string]any) error {
	if spec == nil {
		return errors.New("spec is required")
	}
	return nil
}

func (a *Adapter) ExecuteAction(ctx context.Context, request adapter.ActionRequest) (adapter.Result, error) {
	scriptPath := strings.TrimSpace(request.ModuleScript)
	var (
		output bridge.Output
		err    error
	)

	input := bridge.Input{
		Action: request.Action,
		Account: bridge.InputAccount{
			Identifier: request.Account.Identifier,
			Spec:       request.Account.Spec,
		},
		Params: request.Params,
		Context: bridge.InputContext{
			TenantID:  request.TenantID,
			RequestID: request.RequestID,
		},
	}

	if scriptPath != "" {
		output, err = a.bridge.ExecuteWithScript(ctx, scriptPath, input)
	} else {
		output, err = a.bridge.Execute(ctx, input)
	}
	if err != nil {
		return adapter.Result{}, err
	}

	if output.Status != "success" {
		return adapter.Result{}, fmt.Errorf("%s: %s", output.ErrorCode, output.ErrorMessage)
	}

	var session *adapter.Session
	if output.Session != nil {
		session = &adapter.Session{
			Type:      output.Session.Type,
			Payload:   output.Session.Payload,
			ExpiresAt: output.Session.ExpiresAt,
		}
	}

	return adapter.Result{
		Status:  output.Status,
		Result:  output.Result,
		Session: session,
	}, nil
}
