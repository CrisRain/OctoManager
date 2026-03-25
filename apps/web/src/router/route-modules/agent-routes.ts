import type { AppRouteRecord } from "../route-definitions";
import { PATHS, childPath, routeNames } from "../route-definitions";

const AgentsListPage = () => import("@/pages/AgentsListPage.vue");
const AgentCreatePage = () => import("@/pages/AgentCreatePage.vue");
const AgentDetailPage = () => import("@/pages/AgentDetailPage.vue");
const AgentEditPage = () => import("@/pages/AgentEditPage.vue");

export const agentRoutes: AppRouteRecord[] = [
  {
    path: childPath(PATHS.agents.list),
    name: routeNames.agentsList,
    component: AgentsListPage,
    meta: {
      label: "后台任务",
      navGroup: "primary",
      order: 6,
      iconKey: "robot",
      searchType: "page",
      searchKeywords: ["agent", "后台", "任务"],
    },
  },
  {
    path: childPath(PATHS.agents.create),
    name: routeNames.agentCreate,
    component: AgentCreatePage,
    meta: {
      label: "新建后台任务",
      searchType: "action",
      searchKeywords: ["agent", "创建", "新建", "后台"],
    },
  },
  {
    path: childPath(PATHS.agents.detail),
    name: routeNames.agentDetail,
    component: AgentDetailPage,
  },
  {
    path: childPath(PATHS.agents.edit),
    name: routeNames.agentEdit,
    component: AgentEditPage,
  },
];
