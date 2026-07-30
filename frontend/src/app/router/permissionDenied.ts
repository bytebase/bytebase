export type PermissionDeniedRoute = {
  fullPath: string;
  requiredPermissions: readonly string[];
};

function permissionDeniedFromPath(fullPath: string): string {
  let from = fullPath;
  for (let i = 0; i < 3; i++) {
    const url = new URL(from, window.location.origin);
    if (url.pathname !== "/403") {
      return from;
    }
    const nested = url.searchParams.get("from");
    if (!nested || !nested.startsWith("/") || nested.startsWith("//")) {
      return "";
    }
    from = nested;
  }
  return "";
}

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
    from: permissionDeniedFromPath(route.fullPath),
    api,
    permissions: (permissions ?? route.requiredPermissions).join(","),
    resources: resources.join(","),
  };
}
