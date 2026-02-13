import type { Permission } from "./types";

export {};

declare module "vue-router" {
  interface RouteMeta {
    title?: (route: RouteLocationNormalized) => string;
    requiredPermissionList?: () => Permission[];
    // When true, a layout component (e.g. ProjectV1Layout, SettingLayout)
    // manages document.title for this route, so the router afterEach skips it.
    overrideDocumentTitle?: boolean;
  }
}
