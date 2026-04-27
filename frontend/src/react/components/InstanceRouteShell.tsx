import { LoaderCircle } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { ReactRouteShellTargets } from "@/react/dashboard-shell";
import { instanceNamePrefix } from "@/react/lib/resourceName";
import { useAppStore } from "@/react/stores/app";
import { isValidInstanceName } from "@/types/v1/instance";

interface InstanceRouteShellProps {
  instanceId: string;
  onReady?: (targets: ReactRouteShellTargets | null) => void;
}

export function InstanceRouteShell({
  instanceId,
  onReady,
}: InstanceRouteShellProps) {
  const { t } = useTranslation();
  const contentRef = useRef<HTMLDivElement>(null);
  const instanceName = `${instanceNamePrefix}${instanceId}`;
  const instance = useAppStore((state) => state.instancesByName[instanceName]);
  const fetchInstance = useAppStore((state) => state.fetchInstance);
  const [ready, setReady] = useState(false);
  const routeProps = useMemo(() => ({ instanceId }), [instanceId]);

  useEffect(() => {
    let stale = false;
    setReady(false);
    onReady?.(null);
    void fetchInstance(instanceName).then(() => {
      if (!stale) {
        setReady(true);
      }
    });
    return () => {
      stale = true;
      onReady?.(null);
    };
  }, [fetchInstance, instanceName, onReady]);

  useEffect(() => {
    if (!ready || !isValidInstanceName(instance?.name)) return;
    onReady?.({
      content: contentRef.current,
      routeProps,
    });
    return () => onReady?.(null);
  }, [instance?.name, onReady, ready, routeProps]);

  if (!ready || !isValidInstanceName(instance?.name)) {
    return (
      <div className="flex items-center gap-x-2 m-4 text-sm text-control-light">
        <LoaderCircle className="size-5 animate-spin" />
        {t("common.loading")} {t("common.instance").toLowerCase()}...
      </div>
    );
  }

  return <div ref={contentRef} className="h-full min-h-0" />;
}
