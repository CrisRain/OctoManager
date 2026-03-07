package service

// Container wires all services used by HTTP handlers.
type Container struct {
	AccountType  AccountTypeService
	Account      AccountService
	EmailAccount EmailAccountService
	OctoModule   OctoModuleService
	Job          JobService
	ApiKey       ApiKeyService
	Trigger      TriggerService
	SystemConfig SystemConfigService
	System       SystemService
}
