import { render, screen, waitFor } from "@testing-library/react";
import React from "react";
import { BrowserRouter, MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

type MockFn = ReturnType<typeof vi.fn>;

const mockState = vi.hoisted(() => {
  const mockFns = new Map<string, MockFn>();
  const mockFetchHealth = vi.fn();

  function accountTypeFixture() {
    return {
      id: 1,
      key: "generic_demo",
      name: "Generic Demo",
      category: "generic" as const,
      schema: { type: "object" },
      capabilities: { actions: [{ key: "REGISTER" }] },
      version: 1,
      created_at: "2026-03-07T00:00:00Z",
      updated_at: "2026-03-07T00:00:00Z",
    };
  }

  function accountFixture() {
    return {
      id: 7,
      type_key: "generic_demo",
      identifier: "alpha@example.com",
      status: 1,
      tags: ["demo"],
      spec: {},
      created_at: "2026-03-07T00:00:00Z",
      updated_at: "2026-03-07T00:00:00Z",
    };
  }

  function emailAccountFixture() {
    return {
      id: 11,
      address: "mail@example.com",
      provider: "outlook",
      status: 1,
      graph_summary: {
        ssl: true,
        starttls: false,
        access_token_present: false,
        refresh_token_present: false,
        client_id_present: false,
        client_secret_present: false,
      },
      created_at: "2026-03-07T00:00:00Z",
      updated_at: "2026-03-07T00:00:00Z",
    };
  }

  function jobFixture() {
    return {
      id: 21,
      type_key: "generic_demo",
      action_key: "REGISTER",
      selector: {},
      params: {},
      status: 2,
      created_at: "2026-03-07T00:00:00Z",
      updated_at: "2026-03-07T00:00:00Z",
    };
  }

  function triggerFixture() {
    return {
      id: 31,
      name: "Demo Trigger",
      slug: "demo-trigger",
      type_key: "generic_demo",
      action_key: "REGISTER",
      mode: "async" as const,
      default_selector: {},
      default_params: {},
      token_prefix: "tok_demo",
      enabled: true,
      created_at: "2026-03-07T00:00:00Z",
      updated_at: "2026-03-07T00:00:00Z",
    };
  }

  function moduleFixture() {
    return {
      type_key: "generic_demo",
      category: "generic",
      script_path: "/modules/generic_demo/main.py",
      module_dir: "/modules/generic_demo",
      entry_file: "main.py",
      source: "generated",
      exists: true,
      script_config: {},
    };
  }

  function defaultResponse(name: string): unknown {
    switch (name) {
      case "listAccountTypes":
        return [accountTypeFixture()];
      case "getAccountType":
        return accountTypeFixture();
      case "listAccounts":
        return { items: [accountFixture()], total: 1, limit: 10, offset: 0 };
      case "getAccount":
      case "createAccount":
      case "patchAccount":
        return accountFixture();
      case "deleteAccount":
        return { deleted: true };
      case "batchPatchAccounts":
      case "batchDeleteAccounts":
      case "batchDeleteEmailAccounts":
      case "batchVerifyEmailAccounts":
        return { total: 1, success: 1, failed: 0, failures: [] };
      case "listEmailAccounts":
        return { items: [emailAccountFixture()], total: 1, limit: 10, offset: 0 };
      case "getEmailAccount":
      case "createEmailAccount":
      case "patchEmailAccount":
      case "verifyEmailAccount":
        return emailAccountFixture();
      case "deleteEmailAccount":
        return { deleted: true };
      case "listEmailMailboxes":
      case "previewEmailMailboxes":
        return { reference: "", pattern: "", items: [] };
      case "listEmailMessages":
        return { mailbox: "INBOX", limit: 10, offset: 0, total: 0, items: [] };
      case "getLatestEmailMessage":
      case "previewLatestEmailMessage":
        return { mailbox: "INBOX", found: false };
      case "getEmailMessage":
        return {
          id: "message-1",
          subject: "Hello",
          from: "sender@example.com",
          date: "2026-03-07T00:00:00Z",
          size: 1,
        };
      case "batchImportGraphEmailAccounts":
        return { total: 1, accepted: 1, skipped: 0, failures: [] };
      case "batchRegisterEmailAccounts":
        return {
          requested: 1,
          generated: 1,
          created: 1,
          failed: 0,
          provider: "outlook",
          accounts: [emailAccountFixture()],
          failures: [],
        };
      case "buildOutlookAuthorizeURL":
        return {
          authorize_url: "https://login.example/authorize",
          tenant: "consumers",
          scope: [],
        };
      case "exchangeOutlookCode":
      case "refreshOutlookToken":
        return {
          token_type: "Bearer",
          scope: "",
          expires_in: 3600,
          expires_at: "2026-03-07T12:00:00Z",
        };
      case "listJobs":
        return { items: [jobFixture()], total: 1, limit: 10, offset: 0 };
      case "getJob":
      case "createJob":
      case "patchJob":
      case "cancelJob":
        return jobFixture();
      case "deleteJob":
        return { deleted: true };
      case "listOctoModules":
        return [moduleFixture()];
      case "getOctoModule":
        return moduleFixture();
      case "syncOctoModules":
        return {
          total: 1,
          created: 0,
          existing: 1,
          failed: 0,
          items: [{ ...moduleFixture(), created: false }],
        };
      case "ensureOctoModule":
        return { module: moduleFixture(), created: false };
      case "dryRunOctoModule":
        return { module: moduleFixture(), output: { status: "success", result: { ok: true } } };
      case "getOctoModuleScript":
      case "getOctoModuleFile":
        return { content: "print('hello')" };
      case "updateOctoModuleScript":
      case "updateOctoModuleFile":
        return { updated: true };
      case "listOctoModuleRuns":
        return {
          items: [
            {
              id: 1,
              job_id: 21,
              job_type_key: "generic_demo",
              job_action_key: "REGISTER",
              worker_id: "worker",
              attempt: 1,
              started_at: "2026-03-07T00:00:00Z",
            },
          ],
          total: 1,
          limit: 20,
          offset: 0,
        };
      case "listOctoModuleFiles":
        return {
          module_dir: "/modules/generic_demo",
          entry_file: "main.py",
          files: [{ name: "main.py", size: 12, is_entry: true }],
        };
      case "getModuleVenv":
        return {
          exists: false,
          dir: "/modules/generic_demo/.venv",
          python_path: "python",
          has_requirements: false,
          requirements_content: "",
        };
      case "installModuleDeps":
        return { success: true, output: "installed" };
      case "listApiKeys":
        return [
          {
            id: 1,
            name: "integration",
            key_prefix: "octo_",
            enabled: true,
            created_at: "2026-03-07T00:00:00Z",
            updated_at: "2026-03-07T00:00:00Z",
          },
        ];
      case "createApiKey":
        return {
          api_key: {
            id: 1,
            name: "integration",
            key_prefix: "octo_",
            enabled: true,
            created_at: "2026-03-07T00:00:00Z",
            updated_at: "2026-03-07T00:00:00Z",
          },
          raw_key: "raw-secret",
        };
      case "setApiKeyEnabled":
        return {
          id: 1,
          name: "integration",
          key_prefix: "octo_",
          enabled: true,
          created_at: "2026-03-07T00:00:00Z",
          updated_at: "2026-03-07T00:00:00Z",
        };
      case "deleteApiKey":
        return { deleted: true };
      case "listTriggers":
        return [triggerFixture()];
      case "getTrigger":
      case "patchTrigger":
        return triggerFixture();
      case "createTrigger":
        return { endpoint: triggerFixture(), raw_token: "raw-token" };
      case "deleteTrigger":
        return { deleted: true };
      case "fireTrigger":
        return { endpoint: triggerFixture(), mode: "async", queued: true, input: { type_key: "generic_demo", action_key: "REGISTER", selector: {}, params: {} } };
      case "getConfig":
        return { key: "outlook_oauth_config", value: {} };
      case "setConfig":
        return { key: "outlook_oauth_config" };
      default:
        return {};
    }
  }

  function apiFn(name: string): MockFn {
    const existing = mockFns.get(name);
    if (existing) {
      return existing;
    }
    const fn = vi.fn().mockResolvedValue(defaultResponse(name));
    mockFns.set(name, fn);
    return fn;
  }

  function reset() {
    for (const [name, fn] of mockFns) {
      fn.mockReset();
      fn.mockResolvedValue(defaultResponse(name));
    }
    mockFetchHealth.mockReset();
    mockFetchHealth.mockResolvedValue({
      status: "ok",
      time: "2026-03-07T00:00:00Z",
    });
  }

  const apiProxy = new Proxy<Record<string, unknown>>(
    {},
    {
      get(_target, prop) {
        if (typeof prop !== "string") {
          return undefined;
        }
        return apiFn(prop);
      },
    },
  );

  return {
    apiFn,
    apiProxy,
    mockFetchHealth,
    reset,
  };
});

vi.mock("@/lib/api", () => ({
  api: mockState.apiProxy,
  fetchHealth: mockState.mockFetchHealth,
  extractErrorMessage: (error: unknown) => (error instanceof Error ? error.message : String(error)),
}));

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
    info: vi.fn(),
  },
  Toaster: () => null,
}));

import { App } from "@/App";
import { AppShell } from "@/components/app-shell";
import { AccountsPage } from "@/pages/accounts-page";
import { AccountTypesPage } from "@/pages/account-types-page";
import { ApiKeysPage } from "@/pages/api-keys-page";
import { DashboardPage } from "@/pages/dashboard-page";
import { EmailAccountsOutlookPage } from "@/pages/email-accounts-outlook-page";
import { JobsPage } from "@/pages/jobs-page";
import { OctoModulesPage } from "@/pages/octo-modules-page";
import { SettingsPage } from "@/pages/settings-page";
import { TriggersPage } from "@/pages/triggers-page";

describe("frontend smoke renders", () => {
  beforeEach(() => {
    mockState.reset();
    window.history.replaceState({}, "", "/");
  });

  it("renders AppShell with nested outlet", async () => {
    render(
      <MemoryRouter initialEntries={["/dashboard"]}>
        <Routes>
          <Route element={<AppShell />}>
            <Route path="/dashboard" element={<div>nested dashboard</div>} />
          </Route>
        </Routes>
      </MemoryRouter>,
    );

    expect(await screen.findByText("nested dashboard")).toBeInTheDocument();
    await waitFor(() => expect(mockState.apiFn("listAccountTypes")).toHaveBeenCalled());
  });

  it("renders dashboard page", async () => {
    render(
      <MemoryRouter>
        <DashboardPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText("generic_demo")).toBeInTheDocument();
    await waitFor(() => expect(mockState.mockFetchHealth).toHaveBeenCalled());
  });

  it("renders account types page", async () => {
    render(
      <MemoryRouter>
        <AccountTypesPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText("Generic Demo")).toBeInTheDocument();
  });

  it("renders accounts page with route params", async () => {
    render(
      <MemoryRouter initialEntries={["/accounts/generic_demo"]}>
        <Routes>
          <Route path="/accounts/:typeKey" element={<AccountsPage />} />
        </Routes>
      </MemoryRouter>,
    );

    expect(await screen.findByText("alpha@example.com")).toBeInTheDocument();
    await waitFor(() => expect(mockState.apiFn("getAccountType")).toHaveBeenCalled());
  });

  it("renders email accounts outlook page", async () => {
    render(
      <MemoryRouter>
        <EmailAccountsOutlookPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText("mail@example.com")).toBeInTheDocument();
    await waitFor(() => expect(mockState.apiFn("getConfig")).toHaveBeenCalled());
  });

  it("renders jobs page", async () => {
    render(
      <MemoryRouter>
        <JobsPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText("REGISTER")).toBeInTheDocument();
  });

  it("renders api keys page", async () => {
    render(
      <MemoryRouter>
        <ApiKeysPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText("integration")).toBeInTheDocument();
  });

  it("renders settings page", async () => {
    render(
      <MemoryRouter>
        <SettingsPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText("VITE_API_BASE")).toBeInTheDocument();
    await waitFor(() => expect(mockState.mockFetchHealth).toHaveBeenCalled());
    await waitFor(() => expect(mockState.apiFn("getConfig")).toHaveBeenCalled());
  });

  it("renders triggers page", async () => {
    render(
      <MemoryRouter>
        <TriggersPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText("demo-trigger")).toBeInTheDocument();
    await waitFor(() => expect(mockState.apiFn("listAccountTypes")).toHaveBeenCalled());
  });

  it("renders octo modules page", async () => {
    render(
      <MemoryRouter>
        <OctoModulesPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText("generic_demo")).toBeInTheDocument();
  });

  it("renders oauth callback route through App", async () => {
    window.history.pushState({}, "", "/oauth/callback?code=code-123");

    render(
      <BrowserRouter>
        <App />
      </BrowserRouter>,
    );

    expect(
      await screen.findByText("Authorization succeeded, but no opener window was detected. You can close this page."),
    ).toBeInTheDocument();
  });
});
