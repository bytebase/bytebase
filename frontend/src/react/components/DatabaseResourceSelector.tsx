import {
  ChevronDown,
  ChevronRight,
  Columns3,
  Database as DatabaseIcon,
  Layers,
  Table2,
  X,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  AdvancedSearch,
  getValueFromScopes,
  type ScopeOption,
  type SearchParams,
  type ValueOption,
} from "@/react/components/AdvancedSearch";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { Checkbox } from "@/react/components/ui/checkbox";
import { useVueState } from "@/react/hooks/useVueState";
import {
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useInstanceV1Store,
} from "@/store";
import { useDBSchemaV1Store } from "@/store/modules/v1/dbSchema";
import type { DatabaseResource } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type {
  ColumnMetadata,
  Database,
  DatabaseMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import {
  extractDatabaseResourceName,
  extractInstanceResourceName,
  supportedEngineV1List,
} from "@/utils";

interface TableSelection {
  wholeTableSelected: boolean;
  columns: Set<string>;
}

interface SchemaSelection {
  wholeSchemaSelected: boolean;
  tables: Map<string, TableSelection>;
}

interface DatabaseSelection {
  wholeDatabaseSelected: boolean;
  schemas: Map<string, SchemaSelection>;
}

const emptyTableSelection = (): TableSelection => ({
  wholeTableSelected: false,
  columns: new Set(),
});

const emptySchemaSelection = (): SchemaSelection => ({
  wholeSchemaSelected: false,
  tables: new Map(),
});

const tableKey = (dbName: string, schemaName: string, tableName: string) =>
  `${dbName}/schemas/${schemaName}/tables/${tableName}`;

const environmentNamePrefix = "environments/";
const instanceNamePrefix = "instances/";
const UNKNOWN_ENVIRONMENT_ID = "-1";
const UNKNOWN_ENVIRONMENT_NAME = `${environmentNamePrefix}${UNKNOWN_ENVIRONMENT_ID}`;

interface DatabaseFilter {
  instance?: string;
  environment?: string;
  query?: string;
  labels?: string[];
  engines?: Engine[];
  table?: string;
}

const sameResource = (a: DatabaseResource, b: DatabaseResource) =>
  a.databaseFullName === b.databaseFullName &&
  a.schema === b.schema &&
  a.table === b.table &&
  (a.columns ?? []).join("\0") === (b.columns ?? []).join("\0");

export function DatabaseResourceSelector({
  projectName,
  value,
  onChange,
  includeColumns = false,
}: {
  projectName: string;
  value: DatabaseResource[];
  onChange: (resources: DatabaseResource[]) => void;
  includeColumns?: boolean;
}) {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();
  const dbSchemaStore = useDBSchemaV1Store();
  const environmentStore = useEnvironmentV1Store();
  const instanceStore = useInstanceV1Store();

  const [databases, setDatabases] = useState<Database[]>([]);
  const [searchParams, setSearchParams] = useState<SearchParams>({
    query: "",
    scopes: [],
  });
  const [expandedDatabases, setExpandedDatabases] = useState<Set<string>>(
    new Set()
  );
  const [expandedSchemas, setExpandedSchemas] = useState<Set<string>>(
    new Set()
  );
  const [expandedTables, setExpandedTables] = useState<Set<string>>(new Set());
  const [metadataMap, setMetadataMap] = useState<Map<string, DatabaseMetadata>>(
    new Map()
  );
  const [loadingMetadata, setLoadingMetadata] = useState<Set<string>>(
    new Set()
  );
  const environments = useVueState(
    () => environmentStore.environmentList ?? []
  );

  const searchInstances = useCallback(
    async (keyword: string): Promise<ValueOption[]> => {
      const result = await instanceStore.fetchInstanceList({
        pageSize: 1000,
        filter: keyword.trim() ? { query: keyword } : undefined,
        silent: true,
      });
      return result.instances.map((instance) => {
        const id = extractInstanceResourceName(instance.name);
        return {
          value: id,
          keywords: [id, instance.title],
        };
      });
    },
    [instanceStore]
  );

  const scopeOptions: ScopeOption[] = useMemo(
    () => [
      {
        id: "environment",
        title: t("issue.advanced-search.scope.environment.title"),
        description: t("issue.advanced-search.scope.environment.description"),
        options: [
          {
            id: UNKNOWN_ENVIRONMENT_ID,
            name: UNKNOWN_ENVIRONMENT_NAME,
            title: "",
          },
          ...environments,
        ].map((env) => {
          const isUnknown = env.name === UNKNOWN_ENVIRONMENT_NAME;
          return {
            value: env.id,
            keywords: isUnknown
              ? ["unassigned", "none", env.id]
              : [`${environmentNamePrefix}${env.id}`, env.title],
            render: isUnknown
              ? () => (
                  <span className="italic text-control-light">
                    {t("common.unassigned")}
                  </span>
                )
              : undefined,
            custom: isUnknown,
          };
        }),
      },
      {
        id: "instance",
        title: t("issue.advanced-search.scope.instance.title"),
        description: t("issue.advanced-search.scope.instance.description"),
        onSearch: searchInstances,
      },
      {
        id: "label",
        title: t("common.labels"),
        description: t("issue.advanced-search.scope.label.description"),
        allowMultiple: true,
      },
      {
        id: "engine",
        title: t("issue.advanced-search.scope.engine.title"),
        description: t("issue.advanced-search.scope.engine.description"),
        allowMultiple: true,
        options: supportedEngineV1List().map((engine) => ({
          value: Engine[engine],
          keywords: [Engine[engine].toLowerCase()],
        })),
      },
      {
        id: "table",
        title: t("issue.advanced-search.scope.table.title"),
        description: t("issue.advanced-search.scope.table.description"),
      },
    ],
    [environments, searchInstances, t]
  );

  const databaseFilter: DatabaseFilter = useMemo(() => {
    const environmentId = getValueFromScopes(searchParams, "environment");
    const instanceId = getValueFromScopes(searchParams, "instance");
    const labels = searchParams.scopes
      .filter((scope) => scope.id === "label")
      .map((scope) => scope.value);
    const engines = searchParams.scopes
      .filter((scope) => scope.id === "engine")
      .map((scope) => Engine[scope.value as keyof typeof Engine])
      .filter((engine): engine is Engine => engine !== undefined);
    const table = getValueFromScopes(searchParams, "table");

    return {
      query: searchParams.query,
      environment: environmentId
        ? `${environmentNamePrefix}${environmentId}`
        : undefined,
      instance: instanceId ? `${instanceNamePrefix}${instanceId}` : undefined,
      labels: labels.length > 0 ? labels : undefined,
      engines: engines.length > 0 ? engines : undefined,
      table: table || undefined,
    };
  }, [searchParams]);

  useEffect(() => {
    let cancelled = false;
    const fetchAll = async () => {
      let allDatabases: Database[] = [];
      let pageToken = "";
      do {
        const result = await databaseStore.fetchDatabases({
          parent: projectName,
          pageSize: 1000,
          pageToken,
          filter: databaseFilter,
        });
        allDatabases = [...allDatabases, ...result.databases];
        pageToken = result.nextPageToken;
      } while (pageToken);
      if (!cancelled) setDatabases(allDatabases);
    };
    fetchAll();
    return () => {
      cancelled = true;
    };
  }, [projectName, databaseStore, databaseFilter]);

  const selectedResourceMap = useMemo(() => {
    const map = new Map<string, DatabaseSelection>();
    for (const resource of value) {
      const dbName = resource.databaseFullName;
      if (!map.has(dbName)) {
        map.set(dbName, {
          schemas: new Map(),
          wholeDatabaseSelected: false,
        });
      }

      const dbEntry = map.get(dbName)!;
      if (resource.schema === undefined && !resource.table) {
        dbEntry.wholeDatabaseSelected = true;
        continue;
      }

      if (resource.schema === undefined) {
        continue;
      }

      if (!dbEntry.schemas.has(resource.schema)) {
        dbEntry.schemas.set(resource.schema, emptySchemaSelection());
      }
      const schemaEntry = dbEntry.schemas.get(resource.schema)!;

      if (!resource.table) {
        schemaEntry.wholeSchemaSelected = true;
        continue;
      }

      if (!schemaEntry.tables.has(resource.table)) {
        schemaEntry.tables.set(resource.table, emptyTableSelection());
      }
      const tableEntry = schemaEntry.tables.get(resource.table)!;
      if (resource.columns !== undefined && resource.columns.length > 0) {
        for (const column of resource.columns) {
          tableEntry.columns.add(column);
        }
      } else {
        tableEntry.wholeTableSelected = true;
      }
    }
    return map;
  }, [value]);

  const isDatabaseChecked = useCallback(
    (dbName: string): boolean => {
      const entry = selectedResourceMap.get(dbName);
      return entry?.wholeDatabaseSelected === true;
    },
    [selectedResourceMap]
  );

  const isDatabaseIndeterminate = useCallback(
    (dbName: string): boolean => {
      const entry = selectedResourceMap.get(dbName);
      if (!entry) return false;
      if (entry.wholeDatabaseSelected) return false;
      return entry.schemas.size > 0;
    },
    [selectedResourceMap]
  );

  const isSchemaChecked = useCallback(
    (dbName: string, schemaName: string): boolean => {
      const entry = selectedResourceMap.get(dbName);
      if (!entry) return false;
      if (entry.wholeDatabaseSelected) return true;
      return entry.schemas.get(schemaName)?.wholeSchemaSelected === true;
    },
    [selectedResourceMap]
  );

  const isSchemaIndeterminate = useCallback(
    (dbName: string, schemaName: string): boolean => {
      const entry = selectedResourceMap.get(dbName);
      if (!entry || entry.wholeDatabaseSelected) return false;
      const schemaEntry = entry.schemas.get(schemaName);
      if (!schemaEntry || schemaEntry.wholeSchemaSelected) return false;
      return schemaEntry.tables.size > 0;
    },
    [selectedResourceMap]
  );

  const isTableChecked = useCallback(
    (dbName: string, schemaName: string, tableName: string): boolean => {
      const entry = selectedResourceMap.get(dbName);
      if (!entry) return false;
      if (entry.wholeDatabaseSelected) return true;
      const schemaEntry = entry.schemas.get(schemaName);
      if (!schemaEntry) return false;
      if (schemaEntry.wholeSchemaSelected) return true;
      return schemaEntry.tables.get(tableName)?.wholeTableSelected === true;
    },
    [selectedResourceMap]
  );

  const isTableIndeterminate = useCallback(
    (dbName: string, schemaName: string, tableName: string): boolean => {
      const entry = selectedResourceMap.get(dbName);
      if (!entry || entry.wholeDatabaseSelected) return false;
      const schemaEntry = entry.schemas.get(schemaName);
      if (!schemaEntry || schemaEntry.wholeSchemaSelected) return false;
      const tableEntry = schemaEntry.tables.get(tableName);
      if (!tableEntry || tableEntry.wholeTableSelected) return false;
      return tableEntry.columns.size > 0;
    },
    [selectedResourceMap]
  );

  const isColumnChecked = useCallback(
    (
      dbName: string,
      schemaName: string,
      tableName: string,
      columnName: string
    ): boolean => {
      const entry = selectedResourceMap.get(dbName);
      if (!entry) return false;
      if (entry.wholeDatabaseSelected) return true;
      const schemaEntry = entry.schemas.get(schemaName);
      if (!schemaEntry) return false;
      if (schemaEntry.wholeSchemaSelected) return true;
      const tableEntry = schemaEntry.tables.get(tableName);
      if (!tableEntry) return false;
      if (tableEntry.wholeTableSelected) return true;
      return tableEntry.columns.has(columnName);
    },
    [selectedResourceMap]
  );

  const addColumnsExcept = useCallback(
    (
      resources: DatabaseResource[],
      dbName: string,
      schemaName: string,
      tableName: string,
      columns: ColumnMetadata[],
      columnName: string
    ) => {
      const remainingColumns = columns
        .map((column) => column.name)
        .filter((name) => name !== columnName);
      if (remainingColumns.length === columns.length) {
        resources.push({
          databaseFullName: dbName,
          schema: schemaName,
          table: tableName,
        });
      } else if (remainingColumns.length > 0) {
        resources.push({
          databaseFullName: dbName,
          schema: schemaName,
          table: tableName,
          columns: remainingColumns,
        });
      }
    },
    []
  );

  const toggleDatabase = useCallback(
    (db: Database) => {
      const dbName = db.name;
      if (isDatabaseChecked(dbName)) {
        onChange(value.filter((r) => r.databaseFullName !== dbName));
      } else {
        const filtered = value.filter((r) => r.databaseFullName !== dbName);
        onChange([...filtered, { databaseFullName: dbName }]);
      }
    },
    [value, onChange, isDatabaseChecked]
  );

  const toggleSchema = useCallback(
    (dbName: string, schemaName: string) => {
      if (isSchemaChecked(dbName, schemaName)) {
        const entry = selectedResourceMap.get(dbName);
        if (entry?.wholeDatabaseSelected) {
          const metadata = metadataMap.get(dbName);
          if (metadata) {
            const filtered = value.filter((r) => r.databaseFullName !== dbName);
            const remaining = metadata.schemas
              .filter((s) => s.name !== schemaName)
              .map((s) => ({ databaseFullName: dbName, schema: s.name }));
            onChange([...filtered, ...remaining]);
            return;
          }
        }
        const filtered = value.filter(
          (r) =>
            !(r.databaseFullName === dbName && r.schema === schemaName) &&
            !(
              r.databaseFullName === dbName &&
              r.schema === undefined &&
              !r.table
            )
        );
        onChange(filtered);
      } else {
        const filtered = value.filter(
          (r) => !(r.databaseFullName === dbName && r.schema === schemaName)
        );
        onChange([
          ...filtered,
          { databaseFullName: dbName, schema: schemaName },
        ]);
      }
    },
    [value, onChange, isSchemaChecked, selectedResourceMap, metadataMap]
  );

  const toggleTable = useCallback(
    (dbName: string, schemaName: string, tableName: string) => {
      if (isTableChecked(dbName, schemaName, tableName)) {
        const entry = selectedResourceMap.get(dbName);
        if (entry?.wholeDatabaseSelected) {
          const metadata = metadataMap.get(dbName);
          if (!metadata) return;
          const newResources = value.filter(
            (r) => r.databaseFullName !== dbName
          );
          for (const schema of metadata.schemas) {
            for (const table of schema.tables) {
              if (schema.name === schemaName && table.name === tableName) {
                continue;
              }
              newResources.push({
                databaseFullName: dbName,
                schema: schema.name,
                table: table.name,
              });
            }
          }
          onChange(newResources);
          return;
        }

        const schemaEntry = entry?.schemas.get(schemaName);
        if (schemaEntry?.wholeSchemaSelected) {
          const metadata = metadataMap.get(dbName);
          if (!metadata) return;
          const schema = metadata.schemas.find((s) => s.name === schemaName);
          if (!schema) return;
          const newResources = value.filter(
            (r) => !(r.databaseFullName === dbName && r.schema === schemaName)
          );
          for (const table of schema.tables) {
            if (table.name === tableName) continue;
            newResources.push({
              databaseFullName: dbName,
              schema: schemaName,
              table: table.name,
            });
          }
          onChange(newResources);
          return;
        }

        onChange(
          value.filter(
            (r) =>
              !(
                r.databaseFullName === dbName &&
                r.schema === schemaName &&
                r.table === tableName
              )
          )
        );
      } else {
        const filtered = value.filter(
          (r) =>
            !(
              r.databaseFullName === dbName &&
              r.schema === schemaName &&
              r.table === tableName
            )
        );
        onChange([
          ...filtered,
          { databaseFullName: dbName, schema: schemaName, table: tableName },
        ]);
      }
    },
    [value, onChange, isTableChecked, selectedResourceMap, metadataMap]
  );

  const toggleColumn = useCallback(
    (
      dbName: string,
      schemaName: string,
      tableName: string,
      columnName: string
    ) => {
      const metadata = metadataMap.get(dbName);
      const schema = metadata?.schemas.find((s) => s.name === schemaName);
      const table = schema?.tables.find((t) => t.name === tableName);
      const entry = selectedResourceMap.get(dbName);
      const checked = isColumnChecked(
        dbName,
        schemaName,
        tableName,
        columnName
      );

      if (checked) {
        const newResources = value.filter((r) => r.databaseFullName !== dbName);
        if (entry?.wholeDatabaseSelected && metadata && table) {
          for (const s of metadata.schemas) {
            for (const t of s.tables) {
              if (s.name === schemaName && t.name === tableName) {
                addColumnsExcept(
                  newResources,
                  dbName,
                  s.name,
                  t.name,
                  t.columns,
                  columnName
                );
              } else {
                newResources.push({
                  databaseFullName: dbName,
                  schema: s.name,
                  table: t.name,
                });
              }
            }
          }
          onChange(newResources);
          return;
        }

        const schemaEntry = entry?.schemas.get(schemaName);
        if (schemaEntry?.wholeSchemaSelected && schema && table) {
          for (const t of schema.tables) {
            if (t.name === tableName) {
              addColumnsExcept(
                newResources,
                dbName,
                schemaName,
                t.name,
                t.columns,
                columnName
              );
            } else {
              newResources.push({
                databaseFullName: dbName,
                schema: schemaName,
                table: t.name,
              });
            }
          }
          onChange(newResources);
          return;
        }

        const tableEntry = schemaEntry?.tables.get(tableName);
        const otherResources = value.filter(
          (r) =>
            !(
              r.databaseFullName === dbName &&
              r.schema === schemaName &&
              r.table === tableName
            )
        );
        if (tableEntry?.wholeTableSelected && table) {
          const nextResources = [...otherResources];
          addColumnsExcept(
            nextResources,
            dbName,
            schemaName,
            tableName,
            table.columns,
            columnName
          );
          onChange(nextResources);
          return;
        }

        const remainingColumns = [
          ...(tableEntry?.columns ?? new Set<string>()),
        ].filter((column) => column !== columnName);
        if (remainingColumns.length === 0) {
          onChange(otherResources);
          return;
        }
        onChange([
          ...otherResources,
          {
            databaseFullName: dbName,
            schema: schemaName,
            table: tableName,
            columns: remainingColumns,
          },
        ]);
        return;
      }

      const tableEntry = entry?.schemas.get(schemaName)?.tables.get(tableName);
      const nextColumns = [
        ...new Set([...(tableEntry?.columns ?? new Set<string>()), columnName]),
      ];
      const otherResources = value.filter(
        (r) =>
          !(
            r.databaseFullName === dbName &&
            r.schema === schemaName &&
            r.table === tableName
          )
      );
      onChange([
        ...otherResources,
        {
          databaseFullName: dbName,
          schema: schemaName,
          table: tableName,
          columns: nextColumns,
        },
      ]);
    },
    [
      addColumnsExcept,
      isColumnChecked,
      metadataMap,
      onChange,
      selectedResourceMap,
      value,
    ]
  );

  const toggleExpandDatabase = useCallback(
    async (dbName: string) => {
      const next = new Set(expandedDatabases);
      if (next.has(dbName)) {
        next.delete(dbName);
      } else {
        next.add(dbName);
        if (!metadataMap.has(dbName) && !loadingMetadata.has(dbName)) {
          setLoadingMetadata((prev) => new Set(prev).add(dbName));
          try {
            const metadata = await dbSchemaStore.getOrFetchDatabaseMetadata({
              database: dbName,
            });
            setMetadataMap((prev) => new Map(prev).set(dbName, metadata));
          } catch {
            // error shown by store
          } finally {
            setLoadingMetadata((prev) => {
              const nextLoading = new Set(prev);
              nextLoading.delete(dbName);
              return nextLoading;
            });
          }
        }
      }
      setExpandedDatabases(next);
    },
    [expandedDatabases, metadataMap, loadingMetadata, dbSchemaStore]
  );

  const toggleExpandSchema = useCallback(
    (key: string) => {
      const next = new Set(expandedSchemas);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      setExpandedSchemas(next);
    },
    [expandedSchemas]
  );

  const toggleExpandTable = useCallback(
    (key: string) => {
      const next = new Set(expandedTables);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      setExpandedTables(next);
    },
    [expandedTables]
  );

  const isAllSelected = useMemo(() => {
    if (databases.length === 0) return false;
    return databases.every((db) => isDatabaseChecked(db.name));
  }, [databases, isDatabaseChecked]);

  const toggleSelectAll = useCallback(() => {
    if (isAllSelected) {
      const filteredNames = new Set(databases.map((db) => db.name));
      onChange(value.filter((r) => !filteredNames.has(r.databaseFullName)));
    } else {
      const existing = new Set(
        value
          .filter((r) => r.schema === undefined && !r.table)
          .map((r) => r.databaseFullName)
      );
      const filteredNames = new Set(databases.map((db) => db.name));
      const kept = value.filter((r) => !filteredNames.has(r.databaseFullName));
      const toAdd = databases
        .filter((db) => !existing.has(db.name))
        .map((db) => ({ databaseFullName: db.name }));
      onChange([...kept, ...toAdd]);
    }
  }, [isAllSelected, databases, value, onChange]);

  const removeResource = useCallback(
    (resource: DatabaseResource) => {
      onChange(value.filter((r) => !sameResource(r, resource)));
    },
    [value, onChange]
  );

  const resourceLabel = (r: DatabaseResource): string => {
    const { databaseName } = extractDatabaseResourceName(r.databaseFullName);
    const path = [databaseName, r.schema, r.table].filter(Boolean).join(".");
    const columns = r.columns ?? [];
    if (r.table && columns.length > 0) {
      return `${path}: ${columns.join(", ")}`;
    }
    if (r.table) return path;
    if (r.schema) return `${databaseName}.${r.schema}`;
    return databaseName;
  };

  const hasSchemas = (metadata: DatabaseMetadata): boolean => {
    if (metadata.schemas.length === 1 && metadata.schemas[0].name === "") {
      return false;
    }
    return metadata.schemas.length > 0;
  };

  const renderTable = (
    dbName: string,
    schemaName: string,
    table: TableMetadata
  ) => {
    const key = tableKey(dbName, schemaName, table.name);
    return (
      <TableNode
        key={table.name}
        tableName={table.name}
        columns={table.columns}
        includeColumns={includeColumns}
        checked={isTableChecked(dbName, schemaName, table.name)}
        indeterminate={isTableIndeterminate(dbName, schemaName, table.name)}
        expanded={expandedTables.has(key)}
        onToggleExpand={() => toggleExpandTable(key)}
        onChange={() => toggleTable(dbName, schemaName, table.name)}
        isColumnChecked={(column) =>
          isColumnChecked(dbName, schemaName, table.name, column)
        }
        onColumnChange={(column) =>
          toggleColumn(dbName, schemaName, table.name, column)
        }
      />
    );
  };

  return (
    <div className="border border-control-border rounded-sm overflow-hidden">
      <div className="bg-gray-50 border-b border-control-border px-2 py-1.5">
        <AdvancedSearch
          placeholder={t("common.filter")}
          params={searchParams}
          onParamsChange={setSearchParams}
          scopeOptions={scopeOptions}
        />
      </div>
      <div className="flex" style={{ height: "min(24rem, 60vh)" }}>
        <div className="flex-1 flex flex-col border-r border-control-border min-w-0">
          <div className="flex items-center justify-between px-3 py-1.5 bg-gray-50 border-b border-control-border text-xs text-control-light">
            <button
              type="button"
              className="text-accent hover:underline cursor-pointer"
              onClick={toggleSelectAll}
            >
              {isAllSelected
                ? t("common.deselect-all")
                : t("common.select-all")}
            </button>
            <span>
              {t("common.total")} {databases.length}{" "}
              {t("common.items", { count: databases.length })}
            </span>
          </div>
          <div className="flex-1 overflow-y-auto">
            {databases.map((db) => {
              const { databaseName } = extractDatabaseResourceName(db.name);
              const envName = db.effectiveEnvironment ?? db.environment;
              const instanceTitle = db.instanceResource?.title ?? "";
              const isExpanded = expandedDatabases.has(db.name);
              const metadata = metadataMap.get(db.name);
              const isLoading = loadingMetadata.has(db.name);
              const dbChecked = isDatabaseChecked(db.name);
              const dbIndeterminate = isDatabaseIndeterminate(db.name);
              const firstSchema = metadata?.schemas[0];

              return (
                <div key={db.name}>
                  <div className="flex items-center gap-x-1 px-2 py-1 hover:bg-gray-50 group">
                    <button
                      type="button"
                      className="shrink-0 w-4 h-4 flex items-center justify-center text-control-light hover:text-control cursor-pointer"
                      onClick={() => toggleExpandDatabase(db.name)}
                    >
                      {isExpanded ? (
                        <ChevronDown className="w-3.5 h-3.5" />
                      ) : (
                        <ChevronRight className="w-3.5 h-3.5" />
                      )}
                    </button>
                    <Checkbox
                      className="shrink-0"
                      checked={
                        dbChecked
                          ? true
                          : dbIndeterminate
                            ? "indeterminate"
                            : false
                      }
                      onCheckedChange={() => toggleDatabase(db)}
                    />
                    {envName && (
                      <EnvironmentLabel
                        environmentName={envName}
                        className="text-xs shrink-0"
                      />
                    )}
                    <DatabaseIcon className="w-3.5 h-3.5 text-control-light shrink-0" />
                    <span className="text-sm truncate">{databaseName}</span>
                  </div>
                  {instanceTitle && (
                    <div className="pl-10 text-xs text-control-light pb-0.5">
                      ({instanceTitle})
                    </div>
                  )}
                  {isExpanded && (
                    <div className="pl-6">
                      {isLoading && (
                        <div className="px-2 py-1 text-xs text-control-light">
                          {t("common.loading")}...
                        </div>
                      )}
                      {metadata && !hasSchemas(metadata) && firstSchema && (
                        <>
                          {firstSchema.tables.map((table) =>
                            renderTable(db.name, firstSchema.name, table)
                          )}
                        </>
                      )}
                      {metadata && hasSchemas(metadata) && (
                        <>
                          {metadata.schemas.map((schema) => {
                            const schemaKey = `${db.name}/${schema.name}`;
                            const schemaExpanded =
                              expandedSchemas.has(schemaKey);
                            const schemaChecked = isSchemaChecked(
                              db.name,
                              schema.name
                            );
                            const schemaIndeterminate = isSchemaIndeterminate(
                              db.name,
                              schema.name
                            );

                            return (
                              <div key={schema.name}>
                                <div className="flex items-center gap-x-1 px-2 py-1 hover:bg-gray-50">
                                  <button
                                    type="button"
                                    className="shrink-0 w-4 h-4 flex items-center justify-center text-control-light hover:text-control cursor-pointer"
                                    onClick={() =>
                                      toggleExpandSchema(schemaKey)
                                    }
                                  >
                                    {schemaExpanded ? (
                                      <ChevronDown className="w-3.5 h-3.5" />
                                    ) : (
                                      <ChevronRight className="w-3.5 h-3.5" />
                                    )}
                                  </button>
                                  <Checkbox
                                    className="shrink-0"
                                    checked={
                                      schemaChecked
                                        ? true
                                        : schemaIndeterminate
                                          ? "indeterminate"
                                          : false
                                    }
                                    onCheckedChange={() =>
                                      toggleSchema(db.name, schema.name)
                                    }
                                  />
                                  <Layers className="w-3.5 h-3.5 text-control-light shrink-0" />
                                  <span className="text-sm truncate">
                                    {schema.name}
                                  </span>
                                </div>
                                {schemaExpanded && (
                                  <div className="pl-6">
                                    {schema.tables.map((table) =>
                                      renderTable(db.name, schema.name, table)
                                    )}
                                  </div>
                                )}
                              </div>
                            );
                          })}
                        </>
                      )}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
        <div className="flex-1 flex flex-col min-w-0">
          <div className="flex items-center px-3 py-1.5 bg-gray-50 border-b border-control-border text-xs text-control-light">
            <span>
              {value.length} {t("common.items", { count: value.length })}{" "}
              {t("common.selected").toLowerCase()}
            </span>
          </div>
          <div className="flex-1 overflow-y-auto">
            {value.length === 0 ? (
              <div className="flex items-center justify-center h-full text-sm text-control-placeholder">
                {t("common.no-data")}
              </div>
            ) : (
              <div className="p-1">
                {value.map((r) => (
                  <div
                    key={`${r.databaseFullName}/${r.schema ?? ""}/${r.table ?? ""}/${(r.columns ?? []).join("\0")}`}
                    className="flex items-center gap-x-1 px-2 py-1 text-sm hover:bg-gray-50 group rounded-sm"
                  >
                    <span className="flex-1 truncate">{resourceLabel(r)}</span>
                    <button
                      type="button"
                      className="shrink-0 w-4 h-4 text-control-light hover:text-control opacity-0 group-hover:opacity-100 cursor-pointer"
                      onClick={() => removeResource(r)}
                    >
                      <X className="w-3.5 h-3.5" />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function TableNode({
  tableName,
  columns,
  includeColumns,
  checked,
  indeterminate,
  expanded,
  onToggleExpand,
  onChange,
  isColumnChecked,
  onColumnChange,
}: {
  tableName: string;
  columns: ColumnMetadata[];
  includeColumns: boolean;
  checked: boolean;
  indeterminate: boolean;
  expanded: boolean;
  onToggleExpand: () => void;
  onChange: () => void;
  isColumnChecked: (column: string) => boolean;
  onColumnChange: (column: string) => void;
}) {
  return (
    <div>
      <div className="flex items-center gap-x-1 px-2 py-1 hover:bg-gray-50">
        {includeColumns && columns.length > 0 ? (
          <button
            type="button"
            className="shrink-0 w-4 h-4 flex items-center justify-center text-control-light hover:text-control cursor-pointer"
            onClick={onToggleExpand}
          >
            {expanded ? (
              <ChevronDown className="w-3.5 h-3.5" />
            ) : (
              <ChevronRight className="w-3.5 h-3.5" />
            )}
          </button>
        ) : (
          <span className="shrink-0 w-4" />
        )}
        <Checkbox
          className="shrink-0"
          checked={checked ? true : indeterminate ? "indeterminate" : false}
          onCheckedChange={() => onChange()}
        />
        <Table2 className="w-3.5 h-3.5 text-control-light shrink-0" />
        <span className="text-sm truncate">{tableName}</span>
      </div>
      {includeColumns && expanded && columns.length > 0 && (
        <div className="pl-6">
          {columns.map((column) => (
            <div
              key={column.name}
              className="flex items-center gap-x-1 px-2 py-1 hover:bg-gray-50"
            >
              <span className="shrink-0 w-4" />
              <Checkbox
                className="shrink-0"
                checked={isColumnChecked(column.name)}
                onCheckedChange={() => onColumnChange(column.name)}
              />
              <Columns3 className="w-3.5 h-3.5 text-control-light shrink-0" />
              <span className="text-sm truncate">{column.name}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
