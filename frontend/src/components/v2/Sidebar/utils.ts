import type { RouteRecordRaw } from "vue-router";
import type { Permission } from "@/types";

export const getFlattenRoutes = (
  routes: RouteRecordRaw[],
  permissions: Permission[] = []
): {
  name: string;
  permissions: Permission[];
}[] => {
  return routes.reduce(
    (list, v1Route) => {
      const requiredPermissionListFunc = v1Route.meta?.requiredPermissionList;
      const requiredPermissionList = requiredPermissionListFunc
        ? requiredPermissionListFunc()
        : [];

      if (v1Route.name && v1Route.name.toString() !== "") {
        list.push({
          name: v1Route.name.toString(),
          permissions: [...permissions, ...requiredPermissionList],
        });
      }
      if (v1Route.children) {
        list.push(
          ...getFlattenRoutes(v1Route.children, [
            ...permissions,
            ...requiredPermissionList,
          ])
        );
      }
      return list;
    },
    [] as { name: string; permissions: Permission[] }[]
  );
};
