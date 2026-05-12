import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { EngineIcon } from "@/react/components/EngineIcon";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { Combobox } from "@/react/components/ui/combobox";
import { useActuatorV1Store, useDatabaseV1Store } from "@/store";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  extractDatabaseResourceName,
  getDatabaseEnvironment,
  getDefaultPagination,
  getInstanceResource,
} from "@/utils";

export interface DatabaseSelectProps {
  value: string;
  onChange: (value: string, database: Database | undefined) => void;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  portal?: boolean;
  projectName?: string;
  environmentName?: string;
  allowedEngineTypeList?: Engine[];
}

export function DatabaseSelect({
  value,
  onChange,
  placeholder,
  disabled,
  className,
  portal,
  projectName,
  environmentName,
  allowedEngineTypeList,
}: DatabaseSelectProps) {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();
  const actuatorStore = useActuatorV1Store();
  const [databases, setDatabases] = useState<Database[]>([]);

  // Stabilize engines array to avoid re-fetching on every render
  const enginesRef = useRef(allowedEngineTypeList);
  const stableEngines = useMemo(() => {
    const prev = enginesRef.current;
    if (
      prev &&
      allowedEngineTypeList &&
      prev.length === allowedEngineTypeList.length &&
      prev.every((e, i) => e === allowedEngineTypeList[i])
    ) {
      return prev;
    }
    enginesRef.current = allowedEngineTypeList;
    return allowedEngineTypeList;
  }, [allowedEngineTypeList]);

  const fetchDatabases = useCallback(
    (query: string) => {
      databaseStore
        .fetchDatabases({
          parent: projectName ?? actuatorStore.workspaceResourceName,
          filter: {
            environment: environmentName,
            engines: stableEngines,
            query,
          },
          pageSize: getDefaultPagination(),
          silent: true,
        })
        .then((result) => setDatabases(result.databases));
    },
    [databaseStore, projectName, environmentName, stableEngines, actuatorStore]
  );

  useEffect(() => {
    fetchDatabases("");
  }, [fetchDatabases]);

  const handleChange = useCallback(
    (name: string) => {
      const db = databases.find((d) => d.name === name);
      onChange(name, db);
    },
    [databases, onChange]
  );

  return (
    <Combobox
      value={value}
      onChange={handleChange}
      placeholder={placeholder ?? t("database.select")}
      noResultsText={t("common.no-data")}
      onSearch={fetchDatabases}
      disabled={disabled}
      className={className}
      portal={portal}
      renderValue={(opt) => {
        const db = databases.find((d) => d.name === opt.value);
        if (!db) return opt.label;
        const inst = getInstanceResource(db);
        return (
          <span className="flex items-center gap-1.5 truncate">
            <EngineIcon engine={inst.engine} className="h-4 w-4" />
            {extractDatabaseResourceName(db.name).databaseName}
          </span>
        );
      }}
      options={databases.map((db) => {
        const inst = getInstanceResource(db);
        return {
          value: db.name,
          label: extractDatabaseResourceName(db.name).databaseName,
          description: extractDatabaseResourceName(db.name).instance,
          render: () => (
            <div className="flex flex-col gap-0.5">
              <div className="flex items-center gap-1.5">
                {inst.title && (
                  <>
                    <EngineIcon engine={inst.engine} className="h-4 w-4" />
                    <span>{inst.title}</span>
                    <span className="text-control-placeholder">&gt;</span>
                  </>
                )}
                <EnvironmentLabel
                  environmentName={getDatabaseEnvironment(db).name}
                />
                <span className="text-control-placeholder">&gt;</span>
                <span>{extractDatabaseResourceName(db.name).databaseName}</span>
              </div>
              <span className="text-xs text-control-placeholder">
                {extractDatabaseResourceName(db.name).instance}/databases/
                {extractDatabaseResourceName(db.name).databaseName}
              </span>
            </div>
          ),
        };
      })}
    />
  );
}
