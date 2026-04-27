import { useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import type { ReactRouteShellTargets } from "@/react/dashboard-shell";
import { useWorkspacePermission } from "@/react/hooks/useAppState";
import { useCurrentRoute } from "@/react/router";
import { setDocumentTitle } from "@/utils";

interface SettingRouteShellProps {
  onReady?: (targets: ReactRouteShellTargets | null) => void;
}

export function SettingRouteShell({ onReady }: SettingRouteShellProps) {
  const { t } = useTranslation();
  const route = useCurrentRoute();
  const contentRef = useRef<HTMLDivElement>(null);
  const allowEdit = useWorkspacePermission("bb.settings.set");
  const routeProps = useMemo(() => ({ allowEdit }), [allowEdit]);

  useEffect(() => {
    if (route.title) {
      setDocumentTitle(route.title, t("common.settings"));
    }
  }, [route.title, t]);

  useEffect(() => {
    onReady?.({
      content: contentRef.current,
      routeProps,
    });
    return () => onReady?.(null);
  }, [onReady, route.fullPath, routeProps]);

  return <div ref={contentRef} className="min-h-full" />;
}
