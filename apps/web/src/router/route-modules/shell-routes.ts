import type { AppRouteRecord } from "../route-definitions";
import { PATHS, childPath, routeNames } from "../route-definitions";

const DashboardPage = () => import("@/pages/DashboardPage.vue");
const SettingsPage = () => import("@/pages/SettingsPage.vue");
const LogsPage = () => import("@/pages/LogsPage.vue");

export const shellRoutes: AppRouteRecord[] = [
  { path: "", redirect: PATHS.dashboard },
  {
    path: childPath(PATHS.dashboard),
    name: routeNames.dashboard,
    component: DashboardPage,
    meta: {
      label: "控制台",
      navGroup: "primary",
      order: 1,
      iconKey: "dashboard",
      searchType: "page",
      searchKeywords: ["dashboard", "控制台", "首页"],
      shortcuts: [
        { key: "G then D", label: "前往控制台", description: "跳转到Dashboard" },
      ],
    },
  },
  {
    path: childPath(PATHS.settings.root),
    name: routeNames.settingsRoot,
    component: SettingsPage,
    meta: {
      label: "设置",
      navGroup: "primary",
      order: 9,
      iconKey: "settings",
      searchType: "page",
      searchKeywords: ["settings", "设置", "系统"],
    },
  },
  {
    path: childPath(PATHS.logs.root),
    name: routeNames.logsRoot,
    component: LogsPage,
    meta: {
      label: "系统日志",
      navGroup: "primary",
      order: 8,
      iconKey: "file",
      searchType: "page",
      searchKeywords: ["logs", "日志", "系统日志"],
    },
  },
];
