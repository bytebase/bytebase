import { RoleType, QuickActionType } from "./types";

export {};

declare module "vue-router" {
  interface RouteMeta {
    title?: (route: RouteLocationNormalized) => string;
    quickActionListByRole?: Map<RoleType, QuickActionType[]>;
  }
}
