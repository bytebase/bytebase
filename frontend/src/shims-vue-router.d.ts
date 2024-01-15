import { QuickActionType, ProjectPermission } from "./types";

export {};

declare module "vue-router" {
  interface RouteMeta {
    title?: (route: RouteLocationNormalized) => string;
    getQuickActionList?: (route: RouteLocationNormalized) => QuickActionType[];
    overrideTitle?: boolean;
    requiredProjectPermissionList?: () => ProjectPermission[];
  }
}
