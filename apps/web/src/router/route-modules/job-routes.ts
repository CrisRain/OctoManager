import type { AppRouteRecord } from "../route-definitions";
import { PATHS, childPath, routeNames } from "../route-definitions";

const JobsListPage = () => import("@/pages/JobsListPage.vue");
const JobCreatePage = () => import("@/pages/JobCreatePage.vue");
const JobDetailPage = () => import("@/pages/JobDetailPage.vue");
const JobEditPage = () => import("@/pages/JobEditPage.vue");
const JobExecutionsListPage = () => import("@/pages/JobExecutionsListPage.vue");
const JobExecutionDetailPage = () => import("@/pages/JobExecutionDetailPage.vue");

export const jobRoutes: AppRouteRecord[] = [
  {
    path: childPath(PATHS.jobs.list),
    name: routeNames.jobsList,
    component: JobsListPage,
    meta: {
      label: "定时任务",
      navGroup: "primary",
      order: 5,
      iconKey: "schedule",
      searchType: "page",
      searchKeywords: ["job", "任务", "定时"],
      shortcuts: [
        { key: "G then J", label: "前往任务管理", description: "跳转到任务列表" },
      ],
    },
  },
  {
    path: childPath(PATHS.jobs.create),
    name: routeNames.jobCreate,
    component: JobCreatePage,
    meta: {
      label: "新建任务",
      searchType: "action",
      searchKeywords: ["job", "任务", "新建", "创建"],
    },
  },
  {
    path: childPath(PATHS.jobs.executions),
    name: routeNames.jobExecutionsList,
    component: JobExecutionsListPage,
    meta: {
      label: "执行记录",
      navGroup: "primary",
      navParent: routeNames.jobsList,
      order: 1,
      searchType: "page",
      searchKeywords: ["execution", "执行", "记录"],
    },
  },
  {
    path: childPath(PATHS.jobs.executionDetail),
    name: routeNames.jobExecutionDetail,
    component: JobExecutionDetailPage,
  },
  {
    path: childPath(PATHS.jobs.detail),
    name: routeNames.jobDetail,
    component: JobDetailPage,
  },
  {
    path: childPath(PATHS.jobs.edit),
    name: routeNames.jobEdit,
    component: JobEditPage,
  },
];
