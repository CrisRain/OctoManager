import type {
  Account,
  AccountType,
  ApiKey,
  AppError,
  BatchDeleteAccountResult,
  BatchImportGraphEmailResult,
  BatchEmailAccountResult,
  BatchPatchAccountResult,
  BatchRegisterEmailResult,
  CreateApiKeyResult,
  CreateTriggerResult,
  EmailAccount,
  EmailMessageDetail,
  FireTriggerResult,
  HealthStatus,
  Job,
  JsonObject,
  LatestEmailMessageResult,
  ListEmailMailboxesResult,
  ListEmailMessagesResult,
  ListOctoModuleFilesResult,
  OctoModuleEnsureResult,
  OctoModuleInfo,
  OctoModuleDryRunResult,
  OctoModuleRunHistoryResult,
  OctoModuleSyncResult,
  VenvInfo,
  InstallDepsResult,
  OutlookAuthorizeURLResult,
  OutlookTokenResponse,
  PagedResult,
  SystemMigrateResult,
  SystemStatus,
  TriggerEndpoint
} from "@/types";
import { getAdminKey } from "@/lib/auth";

const API_BASE = (import.meta.env.VITE_API_BASE as string | undefined)?.replace(/\/+$/, "") ?? "";

interface Envelope<T> {
  code: number;
  message: string;
  data?: T;
}

export class ApiRequestError extends Error {
  code: string;
  details?: Record<string, unknown>;

  constructor(payload: AppError) {
    super(payload.message);
    this.name = "ApiRequestError";
    this.code = payload.code;
    this.details = payload.details;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const adminKey = getAdminKey();
  const authHeaders: Record<string, string> = adminKey ? { "X-Api-Key": adminKey } : {};
  const response = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...authHeaders,
      ...init?.headers
    }
  });

  const payload = (await response.json()) as Envelope<T>;
  if (!response.ok || payload.code !== 0) {
    const err: AppError = {
      code: String((payload as any).code ?? "REQUEST_FAILED"),
      message: (payload as any).message ?? "request failed",
      details: undefined
    };
    throw new ApiRequestError(err);
  }
  return payload.data as T;
}

export async function fetchHealth(): Promise<HealthStatus> {
  return request<HealthStatus>("/healthz");
}

export const api = {
  // --- Account Types ---
  listAccountTypes: () => request<AccountType[]>("/api/v1/account-types/"),
  getAccountType: (key: string) => request<AccountType>(`/api/v1/account-types/${encodeURIComponent(key)}`),
  createAccountType: (payload: unknown) =>
    request<AccountType>("/api/v1/account-types/", {
      method: "POST",
      body: JSON.stringify(payload)
    }),
  patchAccountType: (key: string, payload: unknown) =>
    request<AccountType>(`/api/v1/account-types/${encodeURIComponent(key)}`, {
      method: "PATCH",
      body: JSON.stringify(payload)
    }),
  deleteAccountType: (key: string) =>
    request<{ deleted: boolean }>(`/api/v1/account-types/${encodeURIComponent(key)}`, {
      method: "DELETE"
    }),

  // --- Accounts ---
  listAccounts: (params?: { limit?: number; offset?: number; type_key?: string }) => {
    const filtered = Object.fromEntries(
      Object.entries(params ?? {}).filter(([, v]) => v !== undefined && v !== "")
    ) as Record<string, string>;
    const search = new URLSearchParams(filtered).toString();
    return request<PagedResult<Account>>(`/api/v1/accounts/${search ? "?" + search : ""}`);
  },
  getAccount: (id: number) => request<Account>(`/api/v1/accounts/${id}`),
  createAccount: (payload: unknown) =>
    request<Account>("/api/v1/accounts/", {
      method: "POST",
      body: JSON.stringify(payload)
    }),
  patchAccount: (id: number, payload: unknown) =>
    request<Account>(`/api/v1/accounts/${id}`, {
      method: "PATCH",
      body: JSON.stringify(payload)
    }),
  deleteAccount: (id: number) =>
    request<{ deleted: boolean }>(`/api/v1/accounts/${id}`, {
      method: "DELETE"
    }),
  batchPatchAccounts: (payload: { ids: number[]; status?: number; tags?: string[] }) =>
    request<BatchPatchAccountResult>("/api/v1/accounts/batch-patch", {
      method: "POST",
      body: JSON.stringify(payload)
    }),
  batchDeleteAccounts: (ids: number[]) =>
    request<BatchDeleteAccountResult>("/api/v1/accounts/batch-delete", {
      method: "POST",
      body: JSON.stringify({ ids })
    }),

  // --- Email Accounts ---
  listEmailAccounts: (params?: { limit?: number; offset?: number }) => {
    const search = params ? new URLSearchParams(params as Record<string, string>).toString() : "";
    return request<PagedResult<EmailAccount>>(`/api/v1/email/accounts/${search ? "?" + search : ""}`);
  },
  getEmailAccount: (id: number) => request<EmailAccount>(`/api/v1/email/accounts/${id}`),
  createEmailAccount: (payload: unknown) =>
    request<EmailAccount>("/api/v1/email/accounts/", {
      method: "POST",
      body: JSON.stringify(payload)
    }),
  batchImportGraphEmailAccounts: (payload: unknown) =>
    request<BatchImportGraphEmailResult>("/api/v1/email/accounts/batch-import-graph", {
      method: "POST",
      body: JSON.stringify(payload)
    }),
  batchRegisterEmailAccounts: (payload: unknown) =>
    request<BatchRegisterEmailResult>("/api/v1/email/accounts/batch-register", {
      method: "POST",
      body: JSON.stringify(payload)
    }),
  patchEmailAccount: (id: number, payload: unknown) =>
    request<EmailAccount>(`/api/v1/email/accounts/${id}`, {
      method: "PATCH",
      body: JSON.stringify(payload)
    }),
  deleteEmailAccount: (id: number) =>
    request<{ deleted: boolean }>(`/api/v1/email/accounts/${id}`, {
      method: "DELETE"
    }),
  verifyEmailAccount: (id: number) =>
    request<EmailAccount>(`/api/v1/email/accounts/${id}:verify`, {
      method: "POST",
      body: JSON.stringify({})
    }),

  // Email Account - Details
  listEmailMailboxes: (id: number, params?: { reference?: string; pattern?: string }) => {
    const search = new URLSearchParams(params as Record<string, string>).toString();
    return request<ListEmailMailboxesResult>(`/api/v1/email/accounts/${id}/mailboxes?${search}`);
  },
  listEmailMessages: (id: number, params?: { mailbox?: string; limit?: number; offset?: number }) => {
    const search = new URLSearchParams(params as any).toString();
    return request<ListEmailMessagesResult>(`/api/v1/email/accounts/${id}/messages?${search}`);
  },
  getLatestEmailMessage: (id: number, mailbox?: string) => {
    const search = mailbox ? `?mailbox=${encodeURIComponent(mailbox)}` : "";
    return request<LatestEmailMessageResult>(`/api/v1/email/accounts/${id}/messages/latest${search}`);
  },
  getEmailMessage: (id: number, messageId: string, mailbox?: string) => {
    const search = mailbox ? `?mailbox=${encodeURIComponent(mailbox)}` : "";
    return request<EmailMessageDetail>(`/api/v1/email/accounts/${id}/messages/${encodeURIComponent(messageId)}${search}`);
  },

  // Email Account - Preview & OAuth
  previewLatestEmailMessage: (payload: unknown) =>
    request<LatestEmailMessageResult>("/api/v1/email/accounts/preview/messages/latest", {
      method: "POST",
      body: JSON.stringify(payload)
    }),
  previewEmailMailboxes: (payload: unknown) =>
    request<ListEmailMailboxesResult>("/api/v1/email/accounts/preview/mailboxes", {
      method: "POST",
      body: JSON.stringify(payload)
    }),
  batchDeleteEmailAccounts: (ids: number[]) =>
    request<BatchEmailAccountResult>("/api/v1/email/accounts/batch-delete", {
      method: "POST",
      body: JSON.stringify({ ids })
    }),
  batchVerifyEmailAccounts: (ids: number[]) =>
    request<BatchEmailAccountResult>("/api/v1/email/accounts/batch-verify", {
      method: "POST",
      body: JSON.stringify({ ids })
    }),
  buildOutlookAuthorizeURL: (payload: unknown) =>
    request<OutlookAuthorizeURLResult>("/api/v1/email/accounts/outlook/oauth/authorize-url", {
      method: "POST",
      body: JSON.stringify(payload)
    }),
  exchangeOutlookCode: (payload: unknown) =>
    request<OutlookTokenResponse>("/api/v1/email/accounts/outlook/oauth/token", {
      method: "POST",
      body: JSON.stringify(payload)
    }),
  refreshOutlookToken: (payload: unknown) =>
    request<OutlookTokenResponse>("/api/v1/email/accounts/outlook/oauth/refresh", {
      method: "POST",
      body: JSON.stringify(payload)
    }),

  // --- Octo Modules ---
  listOctoModules: () => request<OctoModuleInfo[]>("/api/v1/octo-modules/"),
  getOctoModule: (typeKey: string) => request<OctoModuleInfo>(`/api/v1/octo-modules/${encodeURIComponent(typeKey)}`),
  syncOctoModules: () =>
    request<OctoModuleSyncResult>("/api/v1/octo-modules/sync", {
      method: "POST",
      body: JSON.stringify({})
    }),
  ensureOctoModule: (typeKey: string) =>
    request<OctoModuleEnsureResult>(`/api/v1/octo-modules/${encodeURIComponent(typeKey)}:ensure`, {
      method: "POST",
      body: JSON.stringify({})
    }),
  dryRunOctoModule: (typeKey: string, payload: { action: string; account: { identifier: string; spec?: JsonObject }; params?: JsonObject }) =>
    request<OctoModuleDryRunResult>(`/api/v1/octo-modules/${encodeURIComponent(typeKey)}:dry-run`, {
      method: "POST",
      body: JSON.stringify(payload)
    }),
  getOctoModuleScript: (typeKey: string) =>
    request<{ content: string }>(`/api/v1/octo-modules/${encodeURIComponent(typeKey)}/script`),
  updateOctoModuleScript: (typeKey: string, payload: { content: string }) =>
    request<{ updated: boolean }>(`/api/v1/octo-modules/${encodeURIComponent(typeKey)}/script`, {
      method: "PUT",
      body: JSON.stringify(payload)
    }),
  listOctoModuleRuns: (typeKey: string, params?: { limit?: number; offset?: number }) => {
    const search = params ? new URLSearchParams(params as Record<string, string>).toString() : "";
    return request<OctoModuleRunHistoryResult>(`/api/v1/octo-modules/${encodeURIComponent(typeKey)}/runs${search ? "?" + search : ""}`);
  },
  listOctoModuleFiles: (typeKey: string) =>
    request<ListOctoModuleFilesResult>(`/api/v1/octo-modules/${encodeURIComponent(typeKey)}/files`),
  getOctoModuleFile: (typeKey: string, filename: string) =>
    request<{ content: string }>(`/api/v1/octo-modules/${encodeURIComponent(typeKey)}/files/${encodeURIComponent(filename)}`),
  updateOctoModuleFile: (typeKey: string, filename: string, payload: { content: string }) =>
    request<{ updated: boolean }>(`/api/v1/octo-modules/${encodeURIComponent(typeKey)}/files/${encodeURIComponent(filename)}`, {
      method: "PUT",
      body: JSON.stringify(payload)
    }),
  getModuleVenv: (typeKey: string) =>
    request<VenvInfo>(`/api/v1/octo-modules/${encodeURIComponent(typeKey)}/venv`),
  installModuleDeps: (typeKey: string, payload: { packages?: string[]; from_requirements?: boolean; requirements_content?: string }) =>
    request<InstallDepsResult>(`/api/v1/octo-modules/${encodeURIComponent(typeKey)}/venv/install`, {
      method: "POST",
      body: JSON.stringify(payload),
    }),

  // --- System ---
  getSystemStatus: () => request<SystemStatus>("/api/v1/system/status"),
  runMigration: () =>
    request<SystemMigrateResult>("/api/v1/system/migrate", { method: "POST", body: JSON.stringify({}) }),
  setup: (payload: { admin_key_name?: string }) =>
    request<CreateApiKeyResult>("/api/v1/system/setup", { method: "POST", body: JSON.stringify(payload) }),

  // --- API Keys ---
  listApiKeys: () => request<ApiKey[]>("/api/v1/api-keys/"),
  createApiKey: (payload: { name: string; role?: string; webhook_scope?: string }) =>
    request<CreateApiKeyResult>("/api/v1/api-keys/", {
      method: "POST",
      body: JSON.stringify(payload)
    }),
  setApiKeyEnabled: (id: number, enabled: boolean) =>
    request<ApiKey>(`/api/v1/api-keys/${id}`, {
      method: "PATCH",
      body: JSON.stringify({ enabled })
    }),
  deleteApiKey: (id: number) =>
    request<{ deleted: boolean }>(`/api/v1/api-keys/${id}`, {
      method: "DELETE"
    }),

  // --- Triggers ---
  listTriggers: () => request<TriggerEndpoint[]>("/api/v1/triggers/"),
  getTrigger: (id: number) => request<TriggerEndpoint>(`/api/v1/triggers/${id}`),
  createTrigger: (payload: unknown) =>
    request<CreateTriggerResult>("/api/v1/triggers/", {
      method: "POST",
      body: JSON.stringify(payload)
    }),
  patchTrigger: (id: number, payload: unknown) =>
    request<TriggerEndpoint>(`/api/v1/triggers/${id}`, {
      method: "PATCH",
      body: JSON.stringify(payload)
    }),
  deleteTrigger: (id: number) =>
    request<{ deleted: boolean }>(`/api/v1/triggers/${id}`, {
      method: "DELETE"
    }),
  fireTrigger: (slug: string, token: string, payload?: unknown) =>
    request<FireTriggerResult>(`/webhooks/${encodeURIComponent(slug)}`, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`
      },
      body: JSON.stringify(payload ?? {})
    }),

  // --- System Config ---
  getConfig: (key: string) =>
    request<{ key: string; value: unknown }>(`/api/v1/config/${encodeURIComponent(key)}`),
  setConfig: (key: string, value: unknown) =>
    request<{ key: string }>(`/api/v1/config/${encodeURIComponent(key)}`, {
      method: "PUT",
      body: JSON.stringify({ value }),
    }),

  // --- Jobs ---
  listJobs: (params?: { limit?: number; offset?: number }) => {
    const search = params ? new URLSearchParams(params as Record<string, string>).toString() : "";
    return request<PagedResult<Job>>(`/api/v1/jobs/${search ? "?" + search : ""}`);
  },
  getJob: (id: number) => request<Job>(`/api/v1/jobs/${id}`),
  createJob: (payload: unknown) =>
    request<Job>("/api/v1/jobs/", {
      method: "POST",
      body: JSON.stringify(payload)
    }),
  patchJob: (id: number, payload: unknown) =>
    request<Job>(`/api/v1/jobs/${id}`, {
      method: "PATCH",
      body: JSON.stringify(payload)
    }),
  cancelJob: (id: number) =>
    request<Job>(`/api/v1/jobs/${id}:cancel`, {
      method: "POST",
      body: JSON.stringify({})
    }),
  deleteJob: (id: number) =>
    request<{ deleted: boolean }>(`/api/v1/jobs/${id}`, {
      method: "DELETE"
    })
};

export function extractErrorMessage(error: unknown): string {
  if (error instanceof ApiRequestError) {
    return `${error.code}: ${error.message}`;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return "unknown error";
}
