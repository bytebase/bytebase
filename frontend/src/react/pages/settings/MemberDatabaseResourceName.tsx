import { ChevronRight } from "lucide-react";
import { EngineIcon } from "@/react/components/EngineIcon";
import { useVueState } from "@/react/hooks/useVueState";
import { useDatabaseV1Store, useInstanceV1Store } from "@/store";
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
  const databaseStore = useDatabaseV1Store();
  const instanceStore = useInstanceV1Store();

  const display = useVueState(() => {
    if (!resource) {
      return undefined;
    }

    const { databaseName, instanceName } = extractDatabaseResourceName(
      resource.databaseFullName
    );
    const database = databaseStore.getDatabaseByName(resource.databaseFullName);
    const expectedInstanceName = `instances/${instanceName}`;
    const instance = instanceName
      ? instanceStore.getInstanceByName(expectedInstanceName)
      : undefined;
    const validInstance =
      instance?.name === expectedInstanceName ? instance : undefined;
    const instanceResource =
      database.instanceResource?.name === expectedInstanceName
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
  });

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
