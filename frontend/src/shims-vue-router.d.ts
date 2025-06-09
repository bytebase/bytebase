import type { Permission } from "./types";

export {};

declare module "vue-router" {
  interface RouteMeta {
    title?: (route: RouteLocationNormalized) => string;
    requiredPermissionList?: () => Permission[];
  }
}
