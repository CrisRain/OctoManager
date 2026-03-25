import { type AppRouteRecord, PATHS, routeNames } from "./route-definitions";
import { accountRoutes } from "./route-modules/account-routes";
import { automationRoutes } from "./route-modules/automation-routes";
import { integrationRoutes } from "./route-modules/integration-routes";
import { shellRoutes } from "./route-modules/shell-routes";

const AppShell = () => import("@/components/AppShell.vue");

const OAuthCallbackPage = () => import("@/pages/OAuthCallbackPage.vue");

const appShellChildren: AppRouteRecord[] = [
  ...shellRoutes,
  ...automationRoutes,
  ...accountRoutes,
  ...integrationRoutes,
];

export const routes: AppRouteRecord[] = [
  {
    path: PATHS.oauthCallback,
    name: routeNames.oauthCallback,
    component: OAuthCallbackPage,
  },
  {
    path: "/",
    component: AppShell,
    children: appShellChildren,
  },
];
