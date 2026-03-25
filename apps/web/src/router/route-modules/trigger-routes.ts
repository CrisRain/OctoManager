import type { AppRouteRecord } from "../route-definitions";
import { PATHS, childPath, routeNames } from "../route-definitions";

const TriggersListPage = () => import("@/pages/TriggersListPage.vue");
const TriggerCreatePage = () => import("@/pages/TriggerCreatePage.vue");
const TriggerEditPage = () => import("@/pages/TriggerEditPage.vue");

export const triggerRoutes: AppRouteRecord[] = [
  {
    path: childPath(PATHS.triggers.list),
    name: routeNames.triggersList,
    component: TriggersListPage,
    meta: {
      label: "触发器",
      navGroup: "primary",
      order: 10,
      iconKey: "thunderbolt",
      searchType: "page",
      searchKeywords: ["trigger", "触发器", "webhook"],
    },
  },
  {
    path: childPath(PATHS.triggers.create),
    name: routeNames.triggerCreate,
    component: TriggerCreatePage,
  },
  {
    path: childPath(PATHS.triggers.edit),
    name: routeNames.triggerEdit,
    component: TriggerEditPage,
  },
];
