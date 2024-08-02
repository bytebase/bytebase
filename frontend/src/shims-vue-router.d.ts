import type { QuickActionType, Permission } from "./types";

export {};

declare module "vue-router" {
  interface RouteMeta {
    title?: (route: RouteLocationNormalized) => string;
    getQuickActionList?: (route: RouteLocationNormalized) => QuickActionType[];
    overrideTitle?: boolean;
    requiredWorkspacePermissionList?: () => Permission[];
    requiredProjectPermissionList?: () => Permission[];
  }
}
