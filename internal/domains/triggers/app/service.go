package triggerapp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	jobapp "octomanger/internal/domains/jobs/app"
	triggerdomain "octomanger/internal/domains/triggers/domain"
	triggerpostgres "octomanger/internal/domains/triggers/infra/postgres"
	"octomanger/internal/platform/dbutil"
)

type Service struct {
	repo triggerpostgres.Repository
	jobs jobapp.Service
}

func New(repo triggerpostgres.Repository, jobs jobapp.Service) Service {
	return Service{
		repo: repo,
		jobs: jobs,
	}
}

func (s Service) List(ctx context.Context) ([]triggerdomain.Trigger, error) {
	return s.repo.List(ctx)
}

func (s Service) Get(ctx context.Context, triggerID int64) (*triggerdomain.Trigger, error) {
	return s.repo.GetByID(ctx, triggerID)
}

func (s Service) Create(ctx context.Context, input triggerdomain.CreateInput) (*triggerdomain.CreateResult, error) {
	if strings.TrimSpace(input.Mode) == "" {
		input.Mode = "async"
	}

	token, err := newToken()
	if err != nil {
		return nil, err
	}

	trigger, err := s.repo.Create(ctx, input, token)
	if err != nil {
		return nil, err
	}

	return &triggerdomain.CreateResult{
		Trigger:       *trigger,
		DeliveryToken: token,
	}, nil
}

func (s Service) Patch(ctx context.Context, triggerID int64, input triggerdomain.PatchTriggerInput) (*triggerdomain.Trigger, error) {
	return s.repo.Patch(ctx, triggerID, input)
}

func (s Service) Delete(ctx context.Context, triggerID int64) error {
	return s.repo.Delete(ctx, triggerID)
}

func (s Service) FireByID(ctx context.Context, triggerID int64, input map[string]any) (*triggerdomain.FireResult, error) {
	trigger, err := s.repo.GetByID(ctx, triggerID)
	if err != nil {
		return nil, err
	}
	return s.fire(ctx, *trigger, input)
}

func (s Service) FireByKey(ctx context.Context, key string, token string, input map[string]any) (*triggerdomain.FireResult, error) {
	trigger, tokenHash, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if !trigger.Enabled {
		return nil, fmt.Errorf("trigger is disabled")
	}
	if !triggerpostgres.VerifyToken(token, tokenHash) {
		return nil, fmt.Errorf("invalid trigger token")
	}
	return s.fire(ctx, *trigger, input)
}

func (s Service) fire(ctx context.Context, trigger triggerdomain.Trigger, input map[string]any) (*triggerdomain.FireResult, error) {
	mergedInput := dbutil.MergeMaps(trigger.DefaultInput, input)

	if trigger.Mode == "sync" {
		result, events, err := s.jobs.ExecuteDefinitionDirect(ctx, trigger.JobDefinitionID, mergedInput)
		if err != nil {
			return nil, err
		}

		renderedEvents := make([]any, 0, len(events))
		for _, item := range events {
			renderedEvents = append(renderedEvents, item)
		}

		return &triggerdomain.FireResult{
			Mode:       trigger.Mode,
			TriggerKey: trigger.Key,
			Result:     result,
			Events:     renderedEvents,
		}, nil
	}

	execution, err := s.jobs.EnqueueExecution(ctx, trigger.JobDefinitionID, "trigger:"+trigger.Key, "trigger", mergedInput)
	if err != nil {
		return nil, err
	}

	return &triggerdomain.FireResult{
		Mode:        trigger.Mode,
		TriggerKey:  trigger.Key,
		ExecutionID: &execution.ID,
	}, nil
}

func newToken() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate trigger token: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}
