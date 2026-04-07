import { ChevronLeft, ShieldAlert } from "lucide-react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { router } from "@/router";
import { WORKSPACE_ROOT_MODULE } from "@/router/dashboard/workspaceRoutes";
import type { Permission } from "@/types";

export function Page403() {
  const { t } = useTranslation();

  const query = useMemo(() => {
    const route = router.currentRoute.value;
    return route.query as Record<string, string | undefined>;
  }, []);

  const permissions = useMemo<Permission[]>(() => {
    const raw = query.permissions;
    if (raw) return raw.split(",").filter(Boolean) as Permission[];
    return [];
  }, [query]);

  const resources = useMemo(() => {
    const raw = query.resources;
    if (raw) return raw.split(",").filter(Boolean);
    return [];
  }, [query]);

  const fromPath = query.from;
  const requestAPI = query.api;

  const goHome = () => {
    router.push({ name: WORKSPACE_ROOT_MODULE });
  };

  return (
    <div className="mx-6 my-2">
      <div className="rounded-md border border-red-200 bg-red-50 p-4">
        <div className="flex items-start gap-3">
          <ShieldAlert className="h-5 w-5 text-red-600 mt-0.5 shrink-0" />
          <div className="flex flex-col gap-2">
            <div className="font-medium text-red-800">
              {t("common.missing-required-permission", { permissions: "" })}
            </div>
            {fromPath && <div>Path: {fromPath}</div>}
            {requestAPI && <div>API: {requestAPI}</div>}
            {resources.length > 0 && (
              <div>
                {t("common.resources")}
                <ul className="list-disc pl-4">
                  {resources.map((r) => (
                    <li key={r}>{r}</li>
                  ))}
                </ul>
              </div>
            )}
            {permissions.length > 0 && (
              <div>
                {t("common.required-permission")}
                <ul className="list-disc pl-4">
                  {permissions.map((p) => (
                    <li key={p}>{p}</li>
                  ))}
                </ul>
              </div>
            )}
            <div className="mt-2">
              <Button variant="outline" size="sm" onClick={goHome}>
                <ChevronLeft className="h-4 w-4" />
                {t("error-page.go-back-home")}
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
