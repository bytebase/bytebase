import { ChevronRight } from "lucide-react";
import { useMemo } from "react";
import { EngineIcon } from "@/react/components/EngineIcon";
import { useAppStore } from "@/react/stores/app";
import type { DatabaseResource } from "@/types";
import {
  extractDatabaseResourceName,
  extractInstanceResourceName,
} from "@/utils";

export function MemberDatabaseResourceName({
  resource,
}: {
  resource?: DatabaseResource;
}) {
  const { instanceName } = resource
    ? extractDatabaseResourceName(resource.databaseFullName)
    : { instanceName: "" };
  const expectedInstanceName = instanceName ? `instances/${instanceName}` : "";
  const database = useAppStore((s) =>
    resource ? s.databasesByName[resource.databaseFullName] : undefined
  );
  const instance = useAppStore((s) =>
    expectedInstanceName ? s.instancesByName[expectedInstanceName] : undefined
  );
  const display = useMemo(() => {
    if (!resource) {
      return undefined;
    }

    const { databaseName } = extractDatabaseResourceName(
      resource.databaseFullName
    );
    const validInstance =
      instance?.name === expectedInstanceName ? instance : undefined;
    const instanceResource =
      database?.instanceResource?.name === expectedInstanceName
        ? database.instanceResource
        : undefined;
    const instanceTitle =
      validInstance?.title ||
      instanceResource?.title ||
      extractInstanceResourceName(resource.databaseFullName);

    return {
      databaseName,
      engine: instanceResource?.engine || validInstance?.engine,
      instanceTitle,
    };
  }, [database, expectedInstanceName, instance, resource]);

  if (!display) {
    return <span>*</span>;
  }

  return (
    <div className="flex min-w-0 items-center text-sm">
      {display.engine !== undefined && (
        <EngineIcon engine={display.engine} className="mr-1 h-4 w-4" />
      )}
      <span className="truncate text-gray-600">{display.instanceTitle}</span>
      <ChevronRight className="h-4 w-4 shrink-0 text-gray-500 opacity-60" />
      <span className="truncate text-gray-800">{display.databaseName}</span>
    </div>
  );
}
