import { RoleType, QuickActionType } from "./types";

export {};

declare module "vue-router" {
  interface RouteMeta {
    allowBookmark?: boolean;
    title?: (route: RouteLocationNormalized) => string;
    quickActionListByRole?: (
      route: RouteLocationNormalized
    ) => Map<RoleType, QuickActionType[]>;
    overrideBreadcrumb?: (route: RouteLocationNormalized) => boolean;
    overrideTitle?: boolean;
  }
}
