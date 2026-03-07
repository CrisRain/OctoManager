package dto

type SystemStatusResponse struct {
	Initialized bool `json:"initialized"`
	NeedsSetup  bool `json:"needs_setup"`
}

type SystemMigrateResponse struct {
	DroppedTables  []string `json:"dropped_tables,omitempty"`
	DroppedColumns []string `json:"dropped_columns,omitempty"`
}

type SetupRequest struct {
	AdminKeyName string `json:"admin_key_name"`
}
