import type { AppRouteRecord } from "../route-definitions";
import { PATHS, childPath, routeNames } from "../route-definitions";

const PluginsListPage = () => import("@/pages/PluginsListPage.vue");
const PluginDetailPage = () => import("@/pages/PluginDetailPage.vue");

export const integrationRoutes: AppRouteRecord[] = [
  {
    path: childPath(PATHS.plugins.list),
    name: routeNames.pluginsList,
    component: PluginsListPage,
    meta: {
      label: "插件管理",
      navGroup: "primary",
      order: 7,
      iconKey: "apps",
      searchType: "page",
      searchKeywords: ["plugin", "插件"],
    },
  },
  {
    path: childPath(PATHS.plugins.detail),
    name: routeNames.pluginDetail,
    component: PluginDetailPage,
  },
];
