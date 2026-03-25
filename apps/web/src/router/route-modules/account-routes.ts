import type { AppRouteRecord } from "../route-definitions";
import { accountCoreRoutes } from "./account-core-routes";
import { accountTypeRoutes } from "./account-type-routes";
import { emailRoutes } from "./email-routes";

export const accountRoutes: AppRouteRecord[] = [
  ...accountTypeRoutes,
  ...accountCoreRoutes,
  ...emailRoutes,
];
