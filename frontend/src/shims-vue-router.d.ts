import type { Permission } from "./types";

export {};

declare module "vue-router" {
  interface RouteMeta {
    title?: (route: RouteLocationNormalized) => string;
    requiredPermissionList?: () => Permission[];
    // When true, a React route shell manages document.title for this route,
    // so the router afterEach skips it.
    overrideDocumentTitle?: boolean;
  }
}
