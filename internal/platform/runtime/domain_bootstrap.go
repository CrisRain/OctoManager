package runtime

import (
	accounttypeapp "octomanger/internal/domains/account-types/app"
	accounttypepostgres "octomanger/internal/domains/account-types/infra/postgres"
	accountapp "octomanger/internal/domains/accounts/app"
	accountpostgres "octomanger/internal/domains/accounts/infra/postgres"
	agentapp "octomanger/internal/domains/agents/app"
	agentpostgres "octomanger/internal/domains/agents/infra/postgres"
	emailapp "octomanger/internal/domains/email/app"
	emailpostgres "octomanger/internal/domains/email/infra/postgres"
	jobapp "octomanger/internal/domains/jobs/app"
	jobpostgres "octomanger/internal/domains/jobs/infra/postgres"
	plugins "octomanger/internal/domains/plugins"
	systemapp "octomanger/internal/domains/system/app"
	triggerapp "octomanger/internal/domains/triggers/app"
	triggerpostgres "octomanger/internal/domains/triggers/infra/postgres"
)

type domainServices struct {
	accountTypes accounttypeapp.Service
	accounts     accountapp.Service
	email        emailapp.Service
	triggers     triggerapp.Service
	plugins      plugins.PluginService
	jobs         jobapp.Service
	agents       *agentapp.Service
	system       systemapp.Service
}

func bootstrapDomainServices(resources *platformResources, pluginSvc plugins.PluginService) *domainServices {
	accountTypes := accounttypeapp.New(accounttypepostgres.New(resources.db))
	accounts := accountapp.New(accountpostgres.New(resources.db))
	email := emailapp.New(emailpostgres.New(resources.db))
	jobs := jobapp.New(resources.logger, jobpostgres.New(resources.db, resources.rdb), pluginSvc, resources.cfg.Worker.ID)
	triggers := triggerapp.New(triggerpostgres.New(resources.db), jobs)
	agents := agentapp.New(
		resources.logger,
		agentpostgres.New(resources.db, resources.rdb),
		pluginSvc,
		resources.rdb,
		resources.cfg.Worker.ID,
		resources.cfg.Worker.AgentLoopInterval,
		resources.cfg.Worker.AgentErrorBackoff,
	)
	system := systemapp.New(resources.db, pluginSvc, resources.rdb)

	return &domainServices{
		accountTypes: accountTypes,
		accounts:     accounts,
		email:        email,
		triggers:     triggers,
		plugins:      pluginSvc,
		jobs:         jobs,
		agents:       agents,
		system:       system,
	}
}
