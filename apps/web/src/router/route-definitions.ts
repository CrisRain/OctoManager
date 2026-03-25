import type { RouteRecordRaw } from "vue-router";

export type NavGroup = "primary";
export type SearchType = "page" | "action";

export type IconKey =
  | "dashboard"
  | "layers"
  | "user"
  | "email"
  | "schedule"
  | "robot"
  | "apps"
  | "file"
  | "settings"
  | "thunderbolt";

export interface RouteShortcut {
  key: string;
  label: string;
  description: string;
}

export interface AppRouteMeta {
  label?: string;
  navGroup?: NavGroup;
  order?: number;
  iconKey?: IconKey;
  navParent?: RouteName;
  searchKeywords?: string[];
  searchType?: SearchType;
  shortcuts?: RouteShortcut[];
}

export type AppRouteRecord = RouteRecordRaw & {
  meta?: AppRouteMeta;
};

export const routeNames = {
  oauthCallback: "oauth.callback",
  dashboard: "dashboard",

  agentsList: "agents.list",
  agentCreate: "agents.create",
  agentDetail: "agents.detail",
  agentEdit: "agents.edit",

  jobsList: "jobs.list",
  jobCreate: "jobs.create",
  jobDetail: "jobs.detail",
  jobEdit: "jobs.edit",
  jobExecutionsList: "jobs.executions",
  jobExecutionDetail: "jobs.execution-detail",

  accountTypesList: "account-types.list",
  accountTypeCreate: "account-types.create",
  accountTypeEdit: "account-types.edit",

  accountsList: "accounts.list",
  accountsByType: "accounts.by-type",
  accountCreate: "accounts.create",
  accountDetail: "accounts.detail",
  accountEdit: "accounts.edit",

  triggersList: "triggers.list",
  triggerCreate: "triggers.create",
  triggerEdit: "triggers.edit",

  emailAccountsList: "email-accounts.list",
  emailAccountCreate: "email-accounts.create",
  emailAccountEdit: "email-accounts.edit",
  emailAccountPreview: "email-accounts.preview",

  pluginsList: "plugins.list",
  pluginDetail: "plugins.detail",

  settingsRoot: "settings.root",
  logsRoot: "logs.root",
} as const;

export type RouteName = (typeof routeNames)[keyof typeof routeNames];

export const PATHS = {
  oauthCallback: "/oauth/callback",
  dashboard: "/dashboard",

  agents: {
    list: "/agents",
    create: "/agents/create",
    detail: "/agents/:id",
    edit: "/agents/:id/edit",
  },
  jobs: {
    list: "/jobs",
    create: "/jobs/create",
    executions: "/jobs/executions",
    executionDetail: "/jobs/executions/:id",
    detail: "/jobs/:id",
    edit: "/jobs/:id/edit",
  },
  accountTypes: {
    list: "/account-types",
    create: "/account-types/create",
    edit: "/account-types/:id/edit",
  },
  accounts: {
    list: "/accounts",
    byType: "/accounts/type/:typeKey",
    create: "/accounts/create",
    detail: "/accounts/:id",
    edit: "/accounts/:id/edit",
  },
  triggers: {
    list: "/triggers",
    create: "/triggers/create",
    edit: "/triggers/:id/edit",
  },
  emailAccounts: {
    list: "/email-accounts",
    create: "/email-accounts/create",
    edit: "/email-accounts/:id/edit",
    preview: "/email-accounts/:id/preview",
  },
  plugins: {
    list: "/plugins",
    detail: "/plugins/:id",
  },
  settings: {
    root: "/settings",
  },
  logs: {
    root: "/logs",
  },
} as const;

export const childPath = (path: string) => path.replace(/^\//, "");
const encodeParam = (value: string | number) => encodeURIComponent(String(value));

export const to = {
  oauthCallback: () => PATHS.oauthCallback,
  dashboard: () => PATHS.dashboard,

  agents: {
    list: () => PATHS.agents.list,
    create: () => PATHS.agents.create,
    detail: (id: string | number) => `/agents/${encodeParam(id)}`,
    edit: (id: string | number) => `/agents/${encodeParam(id)}/edit`,
  },
  jobs: {
    list: () => PATHS.jobs.list,
    create: () => PATHS.jobs.create,
    executions: () => PATHS.jobs.executions,
    executionDetail: (id: string | number) => `/jobs/executions/${encodeParam(id)}`,
    detail: (id: string | number) => `/jobs/${encodeParam(id)}`,
    edit: (id: string | number) => `/jobs/${encodeParam(id)}/edit`,
  },
  accountTypes: {
    list: () => PATHS.accountTypes.list,
    create: () => PATHS.accountTypes.create,
    edit: (id: string | number) => `/account-types/${encodeParam(id)}/edit`,
  },
  accounts: {
    list: () => PATHS.accounts.list,
    byType: (typeKey: string) => `/accounts/type/${encodeParam(typeKey)}`,
    create: () => PATHS.accounts.create,
    detail: (id: string | number) => `/accounts/${encodeParam(id)}`,
    edit: (id: string | number) => `/accounts/${encodeParam(id)}/edit`,
  },
  triggers: {
    list: () => PATHS.triggers.list,
    create: () => PATHS.triggers.create,
    edit: (id: string | number) => `/triggers/${encodeParam(id)}/edit`,
  },
  emailAccounts: {
    list: () => PATHS.emailAccounts.list,
    create: () => PATHS.emailAccounts.create,
    edit: (id: string | number) => `/email-accounts/${encodeParam(id)}/edit`,
    preview: (id: string | number) => `/email-accounts/${encodeParam(id)}/preview`,
  },
  plugins: {
    list: () => PATHS.plugins.list,
    detail: (id: string | number) => `/plugins/${encodeParam(id)}`,
  },
  settings: {
    root: () => PATHS.settings.root,
  },
  logs: {
    root: () => PATHS.logs.root,
  },
} as const;
