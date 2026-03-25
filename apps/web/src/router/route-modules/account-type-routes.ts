import type { AppRouteRecord } from "../route-definitions";
import { PATHS, childPath, routeNames } from "../route-definitions";

const AccountTypesListPage = () => import("@/pages/AccountTypesListPage.vue");
const AccountTypeCreatePage = () => import("@/pages/AccountTypeCreatePage.vue");
const AccountTypeEditPage = () => import("@/pages/AccountTypeEditPage.vue");

export const accountTypeRoutes: AppRouteRecord[] = [
  {
    path: childPath(PATHS.accountTypes.list),
    name: routeNames.accountTypesList,
    component: AccountTypesListPage,
    meta: {
      label: "账号类型",
      navGroup: "primary",
      order: 2,
      iconKey: "layers",
      searchType: "page",
      searchKeywords: ["type", "类型", "账号类型"],
    },
  },
  {
    path: childPath(PATHS.accountTypes.create),
    name: routeNames.accountTypeCreate,
    component: AccountTypeCreatePage,
  },
  {
    path: childPath(PATHS.accountTypes.edit),
    name: routeNames.accountTypeEdit,
    component: AccountTypeEditPage,
  },
];
