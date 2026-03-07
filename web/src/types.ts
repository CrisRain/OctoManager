export type JsonObject = Record<string, unknown>;

export interface PagedResult<T> {
  items: T[];
  total: number;
  limit: number;
  offset: number;
}

export interface AppError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
}

export interface AccountType {
  id: number;
  key: string;
  name: string;
  category: "system" | "email" | "generic";
  schema: JsonObject;
  capabilities: JsonObject;
  script_config?: JsonObject | null;
  version: number;
  created_at: string;
  updated_at: string;
}

export interface Account {
  id: number;

  type_key: string;
  identifier: string;
  status: number;
  tags: string[];
  spec: JsonObject;
  created_at: string;
  updated_at: string;
}

export interface EmailAccount {
  id: number;

  address: string;
  provider?: string;
  status: number;
  graph_summary?: EmailConfigSummary;
  created_at: string;
  updated_at: string;
}

export interface EmailConfigSummary {
  host?: string;
  port?: number;
  ssl: boolean;
  starttls: boolean;
  username?: string;
  token_username?: string;
  auth_method?: string;
  tenant?: string;
  mailbox?: string;
  scope?: string[];
  token_expires_at?: string;
  access_token_present: boolean;
  refresh_token_present: boolean;
  client_id_present: boolean;
  client_secret_present: boolean;
}

export interface EmailMessageSummary {
  id: string;
  subject: string;
  from: string;
  to?: string;
  date: string; // ISO string
  size: number;
  flags?: string[];
}

export interface EmailMessageDetail extends EmailMessageSummary {
  cc?: string;
  headers?: Record<string, string>;
  text_body?: string;
  html_body?: string;
}

export interface ListEmailMessagesResult {
  mailbox: string;
  limit: number;
  offset: number;
  total: number;
  items: EmailMessageSummary[];
}

export interface EmailMailbox {
  name: string;
  delimiter?: string;
  flags?: string[];
}

export interface ListEmailMailboxesResult {
  reference: string;
  pattern: string;
  items: EmailMailbox[];
}

export interface LatestEmailMessageResult {
  mailbox: string;
  found: boolean;
  item?: EmailMessageDetail;
}

export interface BatchRegisterEmailFailure {
  index: number;
  address?: string;
  code: string;
  message: string;
}

export interface BatchRegisterEmailResult {
  requested: number;
  generated: number;
  created: number;
  failed: number;
  provider?: string;
  accounts: EmailAccount[];
  failures: BatchRegisterEmailFailure[];
  queued?: boolean;
  task_id?: string;
  job_id?: number;
}

export interface BatchImportGraphEmailFailure {
  line: number;
  address?: string;
  error: string;
}

export interface BatchImportGraphEmailResult {
  total: number;
  accepted: number;
  skipped: number;
  failures: BatchImportGraphEmailFailure[];
  queued?: boolean;
  task_id?: string;
  job_id?: number;
}

export interface Job {
  id: number;

  type_key: string;
  action_key: string;
  selector: JsonObject;
  params: JsonObject;
  status: number;
  last_run?: JobRun;
  created_at: string;
  updated_at: string;
}

export interface OctoModuleInfo {
  type_key: string;
  category: string;
  script_path: string;
  module_dir: string;
  entry_file: string;
  source: string;
  exists: boolean;
  script_config?: JsonObject | null;
  error?: string;
}

export interface OctoModuleFileInfo {
  name: string;
  size: number;
  is_entry: boolean;
}

export interface ListOctoModuleFilesResult {
  module_dir: string;
  entry_file: string;
  files: OctoModuleFileInfo[];
}

export interface OctoModuleEnsureResult {
  module: OctoModuleInfo;
  created: boolean;
}

export interface OctoModuleOutput {
  status: string;
  result?: JsonObject;
  error_code?: string;
  error_message?: string;
}

export interface OctoModuleDryRunResult {
  module: OctoModuleInfo;
  output: OctoModuleOutput;
}

export interface OctoModuleSyncItem {
  type_key: string;
  category: string;
  script_path: string;
  source: string;
  exists: boolean;
  created: boolean;
  error?: string;
}

export interface OctoModuleSyncResult {
  total: number;
  created: number;
  existing: number;
  failed: number;
  items: OctoModuleSyncItem[];
}

export interface BatchAccountFailure {
  id: number;
  code: string;
  message: string;
}

export interface BatchEmailAccountResult {
  total: number;
  success: number;
  failed: number;
  failures: BatchAccountFailure[];
  queued?: boolean;
  task_id?: string;
  job_id?: number;
}

export interface BatchPatchAccountResult {
  total: number;
  success: number;
  failed: number;
  failures: BatchAccountFailure[];
  queued?: boolean;
  task_id?: string;
  job_id?: number;
}

export interface BatchDeleteAccountResult {
  total: number;
  success: number;
  failed: number;
  failures: BatchAccountFailure[];
  queued?: boolean;
  task_id?: string;
  job_id?: number;
}

export interface HealthStatus {
  status: string;
  time: string;
}

export interface JobRun {
  id: number;
  job_id: number;
  job_type_key: string;
  job_action_key: string;
  account_id?: number;
  worker_id: string;
  attempt: number;
  result?: JsonObject;
  error_code?: string;
  error_message?: string;
  started_at: string;
  ended_at?: string;
}

export interface OctoModuleRunHistoryResult {
  items: JobRun[];
  total: number;
  limit: number;
  offset: number;
}

export type ApiKeyRole = "admin" | "webhook";

export interface ApiKey {
  id: number;
  name: string;
  key_prefix: string;
  role: ApiKeyRole;
  webhook_scope?: string;
  enabled: boolean;
  last_used_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateApiKeyResult {
  api_key: ApiKey;
  raw_key: string;
}

export interface SystemStatus {
  initialized: boolean;
  needs_setup: boolean;
}

export interface SystemMigrateResult {
  dropped_tables?: string[];
  dropped_columns?: string[];
}

export type TriggerMode = "async" | "sync";

export interface TriggerEndpoint {
  id: number;
  name: string;
  slug: string;
  type_key: string;
  action_key: string;
  mode: TriggerMode;
  default_selector: JsonObject;
  default_params: JsonObject;
  token_prefix: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateTriggerResult {
  endpoint: TriggerEndpoint;
  raw_token?: string;
}

export interface TriggerExecutionInput {
  type_key: string;
  action_key: string;
  selector: JsonObject;
  params: JsonObject;
}

export interface TriggerExecutionSession {
  type: string;
  payload: JsonObject;
  expires_at?: string;
}

export interface TriggerExecutionAccountResult {
  run_id?: number;
  account_id: number;
  identifier: string;
  status: string;
  result?: JsonObject;
  error_code?: string;
  error_message?: string;
  session?: TriggerExecutionSession;
  started_at: string;
  ended_at?: string;
}

export interface TriggerExecutionOutput {
  job_status: number;
  matched_accounts: number;
  processed_accounts: number;
  error_code?: string;
  error_message?: string;
  results: TriggerExecutionAccountResult[];
}

export interface FireTriggerResult {
  endpoint: TriggerEndpoint;
  mode: TriggerMode;
  queued: boolean;
  input: TriggerExecutionInput;
  job?: Job;
  output?: TriggerExecutionOutput;
}

export interface OutlookAuthorizeURLResult {
  authorize_url: string;
  tenant: string;
  scope: string[];
  state?: string;
  code_challenge?: string;
  code_challenge_method?: string;
}

export interface VenvInfo {
  exists: boolean;
  dir: string;
  python_path: string;
  has_requirements: boolean;
  requirements_content?: string;
}

export interface InstallDepsResult {
  success: boolean;
  output: string;
}

export interface OutlookTokenResponse {
  token_type: string;
  scope: string;
  expires_in: number;
  expires_at: string;
  token_url?: string;
  access_token?: string;
  refresh_token?: string;
}
