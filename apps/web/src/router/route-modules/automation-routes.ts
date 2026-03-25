import type { AppRouteRecord } from "../route-definitions";
import { agentRoutes } from "./agent-routes";
import { jobRoutes } from "./job-routes";
import { triggerRoutes } from "./trigger-routes";

export const automationRoutes: AppRouteRecord[] = [
  ...agentRoutes,
  ...jobRoutes,
  ...triggerRoutes,
];
