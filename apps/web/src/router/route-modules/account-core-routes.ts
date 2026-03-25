import type { AppRouteRecord } from "../route-definitions";
import { PATHS, childPath, routeNames } from "../route-definitions";

const AccountsListPage = () => import("@/pages/AccountsListPage.vue");
const AccountCreatePage = () => import("@/pages/AccountCreatePage.vue");
const AccountDetailPage = () => import("@/pages/AccountDetailPage.vue");
const AccountEditPage = () => import("@/pages/AccountEditPage.vue");

export const accountCoreRoutes: AppRouteRecord[] = [
  {
    path: childPath(PATHS.accounts.list),
    name: routeNames.accountsList,
    component: AccountsListPage,
    meta: {
      label: "账号管理",
      navGroup: "primary",
      order: 3,
      iconKey: "user",
      searchType: "page",
      searchKeywords: ["account", "账号", "账户"],
      shortcuts: [
        { key: "G then A", label: "前往账号管理", description: "跳转到账号列表" },
      ],
    },
  },
  {
    path: childPath(PATHS.accounts.byType),
    name: routeNames.accountsByType,
    component: AccountsListPage,
  },
  {
    path: childPath(PATHS.accounts.create),
    name: routeNames.accountCreate,
    component: AccountCreatePage,
    meta: {
      label: "新建账号",
      searchType: "action",
      searchKeywords: ["account", "账号", "新建", "创建"],
    },
  },
  {
    path: childPath(PATHS.accounts.detail),
    name: routeNames.accountDetail,
    component: AccountDetailPage,
  },
  {
    path: childPath(PATHS.accounts.edit),
    name: routeNames.accountEdit,
    component: AccountEditPage,
  },
];
