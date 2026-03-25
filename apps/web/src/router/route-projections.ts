import type {
  AppRouteMeta,
  AppRouteRecord,
  IconKey,
  NavGroup,
  RouteName,
  RouteShortcut,
  SearchType,
} from "./route-definitions";
import { routes } from "./route-tree";

interface FlatRoute {
  name?: RouteName;
  path: string;
  meta?: AppRouteMeta;
}

const joinPaths = (parentPath: string, childPathValue: string) => {
  if (!parentPath || parentPath === "/") {
    return `/${childPathValue}`.replace(/\/+/g, "/");
  }
  return `${parentPath.replace(/\/$/, "")}/${childPathValue}`.replace(/\/+/g, "/");
};

const flattenRoutes = (items: AppRouteRecord[], parentPath = ""): FlatRoute[] => {
  const result: FlatRoute[] = [];
  for (const route of items) {
    const normalizedPath = route.path?.startsWith("/")
      ? route.path
      : joinPaths(parentPath, route.path ?? "");
    if (route.name || route.meta) {
      result.push({
        name: route.name as RouteName | undefined,
        path: normalizedPath,
        meta: route.meta,
      });
    }
    if (route.children?.length) {
      result.push(...flattenRoutes(route.children as AppRouteRecord[], normalizedPath));
    }
  }
  return result;
};

export const flatRoutes = flattenRoutes(routes);

export interface NavRoute {
  name: RouteName;
  path: string;
  label: string;
  navGroup: NavGroup;
  order: number;
  iconKey?: IconKey;
  navParent?: RouteName;
}

export const navRoutes: NavRoute[] = flatRoutes
  .filter((route): route is FlatRoute & { name: RouteName; meta: AppRouteMeta } =>
    Boolean(route.name && route.meta?.navGroup)
  )
  .map((route) => ({
    name: route.name,
    path: route.path,
    label: route.meta?.label ?? route.name,
    navGroup: route.meta?.navGroup ?? "primary",
    order: route.meta?.order ?? 0,
    iconKey: route.meta?.iconKey,
    navParent: route.meta?.navParent,
  }))
  .sort((a, b) => a.order - b.order);

export interface SearchRoute {
  name: RouteName;
  path: string;
  label: string;
  type: SearchType;
  keywords: string[];
}

export const searchRoutes: SearchRoute[] = flatRoutes
  .filter((route): route is FlatRoute & { name: RouteName; meta: AppRouteMeta } =>
    Boolean(route.name && route.meta?.searchKeywords?.length)
  )
  .map((route) => ({
    name: route.name,
    path: route.path,
    label: route.meta?.label ?? route.name,
    type: route.meta?.searchType ?? "page",
    keywords: route.meta?.searchKeywords ?? [],
  }));

export interface ShortcutRoute extends RouteShortcut {
  name: RouteName;
  path: string;
}

export const shortcutRoutes: ShortcutRoute[] = flatRoutes.flatMap((route) =>
  (route.meta?.shortcuts ?? []).map((shortcut) => ({
    ...shortcut,
    name: route.name as RouteName,
    path: route.path,
  }))
);
