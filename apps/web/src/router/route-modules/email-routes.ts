import type { AppRouteRecord } from "../route-definitions";
import { PATHS, childPath, routeNames } from "../route-definitions";

const EmailAccountsListPage = () => import("@/pages/EmailAccountsListPage.vue");
const EmailAccountCreatePage = () => import("@/pages/EmailAccountCreatePage.vue");
const EmailAccountEditPage = () => import("@/pages/EmailAccountEditPage.vue");
const EmailAccountPreviewPage = () => import("@/pages/EmailAccountPreviewPage.vue");

export const emailRoutes: AppRouteRecord[] = [
  {
    path: childPath(PATHS.emailAccounts.list),
    name: routeNames.emailAccountsList,
    component: EmailAccountsListPage,
    meta: {
      label: "邮箱账号",
      navGroup: "primary",
      order: 4,
      iconKey: "email",
      searchType: "page",
      searchKeywords: ["email", "邮箱", "邮件"],
    },
  },
  {
    path: childPath(PATHS.emailAccounts.create),
    name: routeNames.emailAccountCreate,
    component: EmailAccountCreatePage,
    meta: {
      label: "新建邮箱",
      searchType: "action",
      searchKeywords: ["email", "邮箱", "新建", "创建"],
    },
  },
  {
    path: childPath(PATHS.emailAccounts.edit),
    name: routeNames.emailAccountEdit,
    component: EmailAccountEditPage,
  },
  {
    path: childPath(PATHS.emailAccounts.preview),
    name: routeNames.emailAccountPreview,
    component: EmailAccountPreviewPage,
  },
];
