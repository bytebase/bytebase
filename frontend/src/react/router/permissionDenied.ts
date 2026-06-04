export type PermissionDeniedRoute = {
  fullPath: string;
  requiredPermissions: readonly string[];
};

export function buildPermissionDeniedRouteQuery({
  route,
  api = "",
  permissions,
  resources = [],
}: {
  route: PermissionDeniedRoute;
  api?: string;
  permissions?: readonly string[];
  resources?: string[];
}): Record<string, string> {
  return {
    from: route.fullPath,
    api,
    permissions: (permissions ?? route.requiredPermissions).join(","),
    resources: resources.join(","),
  };
}
