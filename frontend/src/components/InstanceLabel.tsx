import { type MouseEventHandler, type ReactNode, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { INSTANCE_ROUTE_DETAIL } from "@/app/router/handles";
import { EngineIcon } from "@/components/EngineIcon";
import { RouterLink, type RouterLinkProps } from "@/components/RouterLink";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app";
import type {
  Instance,
  InstanceResource,
} from "@/types/proto-es/v1/instance_service_pb";
import { isValidInstanceName } from "@/types/v1/instance";
import {
  extractInstanceResourceName,
  hasWorkspacePermissionV2,
  instanceV1Name,
} from "@/utils";

export function InstanceLabel({
  children,
  instance: instanceProp,
  instanceName,
  showResourceType = false,
  className,
  link,
  onClick,
  ...linkProps
}: {
  children?: ReactNode;
  instance?: Instance | InstanceResource;
  instanceName?: string;
  showResourceType?: boolean;
  link?: boolean;
  onClick?: MouseEventHandler<HTMLAnchorElement>;
} & Pick<RouterLinkProps, "className" | "rel" | "target">) {
  const { t } = useTranslation();
  const name = instanceProp?.name ?? instanceName ?? "";
  const instanceId = extractInstanceResourceName(name);
  const validInstanceName = isValidInstanceName(name) && !!instanceId;
  const canGetInstance = hasWorkspacePermissionV2("bb.instances.get");
  const cachedInstance = useAppStore((s) =>
    validInstanceName && canGetInstance ? s.instancesByName[name] : undefined
  );
  const instance = instanceProp ?? cachedInstance;
  const shouldFetchInstance = children === undefined;

  useEffect(() => {
    if (
      shouldFetchInstance &&
      validInstanceName &&
      canGetInstance &&
      !instanceProp
    ) {
      void useAppStore.getState().fetchInstance(name);
    }
  }, [
    canGetInstance,
    instanceProp,
    name,
    shouldFetchInstance,
    validInstanceName,
  ]);

  if (!validInstanceName || !canGetInstance) {
    return <span className={className}>{children ?? (name || "-")}</span>;
  }

  const label = instance?.title ? instanceV1Name(instance) : instanceId;
  const content: ReactNode = children ?? (
    <>
      {showResourceType && (
        <span className="mr-1 text-xs text-control-light">
          {t("common.instance")}:
        </span>
      )}
      {instance && <EngineIcon engine={instance.engine} className="size-4" />}
      <span className="truncate">{label}</span>
    </>
  );

  if (!link) {
    if (className) {
      return (
        <span
          className={cn("inline-flex min-w-0 items-center gap-x-1", className)}
        >
          {content}
        </span>
      );
    }
    return (
      <span className="inline-flex min-w-0 items-center gap-x-1">
        {content}
      </span>
    );
  }

  return (
    <RouterLink
      {...linkProps}
      to={{
        name: INSTANCE_ROUTE_DETAIL,
        params: { instanceId },
      }}
      className={cn(
        "inline-flex min-w-0 max-w-full items-center gap-x-1 normal-link hover:underline",
        className
      )}
      onClick={(event) => {
        event.stopPropagation();
        onClick?.(event);
      }}
    >
      {content}
    </RouterLink>
  );
}
