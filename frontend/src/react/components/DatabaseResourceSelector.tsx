import {
  ChevronDown,
  ChevronRight,
  Database as DatabaseIcon,
  Layers,
  Table2,
  X,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { SearchInput } from "@/react/components/ui/search-input";
import { useDatabaseV1Store } from "@/store";
import { useDBSchemaV1Store } from "@/store/modules/v1/dbSchema";
import type { DatabaseResource } from "@/types";
import type {
  Database,
  DatabaseMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { extractDatabaseResourceName } from "@/utils";

export function DatabaseResourceSelector({
  projectName,
  value,
  onChange,
}: {
  projectName: string;
  value: DatabaseResource[];
  onChange: (resources: DatabaseResource[]) => void;
}) {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();
  const dbSchemaStore = useDBSchemaV1Store();

  const [databases, setDatabases] = useState<Database[]>([]);
  const [filter, setFilter] = useState("");
  const [expandedDatabases, setExpandedDatabases] = useState<Set<string>>(
    new Set()
  );
  const [expandedSchemas, setExpandedSchemas] = useState<Set<string>>(
    new Set()
  );
  const [metadataMap, setMetadataMap] = useState<Map<string, DatabaseMetadata>>(
    new Map()
  );
  const [loadingMetadata, setLoadingMetadata] = useState<Set<string>>(
    new Set()
  );

  // Fetch databases on mount
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
  }, [projectName, databaseStore]);

  const filteredDatabases = useMemo(() => {
    if (!filter.trim()) return databases;
    const keyword = filter.trim().toLowerCase();
    return databases.filter((db) => {
      const { databaseName } = extractDatabaseResourceName(db.name);
      return databaseName.toLowerCase().includes(keyword);
    });
  }, [databases, filter]);

  // Build a set of selected database full names for quick lookup
  const selectedResourceMap = useMemo(() => {
    const map = new Map<
      string,
      { schemas: Map<string, Set<string>>; wholeDatabaseSelected: boolean }
    >();
    for (const r of value) {
      const dbName = r.databaseFullName;
      if (!map.has(dbName)) {
        map.set(dbName, { schemas: new Map(), wholeDatabaseSelected: false });
      }
      const entry = map.get(dbName)!;
      if (!r.schema && !r.table) {
        entry.wholeDatabaseSelected = true;
      } else if (r.schema && !r.table) {
        // Schema-level selection
        entry.schemas.set(r.schema, new Set());
      } else if (r.schema !== undefined && r.table) {
        if (!entry.schemas.has(r.schema)) {
          entry.schemas.set(r.schema, new Set());
        }
        entry.schemas.get(r.schema)!.add(r.table);
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
      const schemaEntry = entry.schemas.get(schemaName);
      if (!schemaEntry) return false;
      return schemaEntry.size === 0; // Empty set means whole schema selected
    },
    [selectedResourceMap]
  );

  const isSchemaIndeterminate = useCallback(
    (dbName: string, schemaName: string): boolean => {
      const entry = selectedResourceMap.get(dbName);
      if (!entry) return false;
      if (entry.wholeDatabaseSelected) return false;
      const schemaEntry = entry.schemas.get(schemaName);
      if (!schemaEntry) return false;
      return schemaEntry.size > 0; // Has specific tables selected
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
      if (schemaEntry.size === 0) return true; // Whole schema selected
      return schemaEntry.has(tableName);
    },
    [selectedResourceMap]
  );

  const toggleDatabase = useCallback(
    (db: Database) => {
      const dbName = db.name;
      if (isDatabaseChecked(dbName)) {
        // Remove all resources for this database
        onChange(value.filter((r) => r.databaseFullName !== dbName));
      } else {
        // Remove any partial selections and add whole database
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
          // Whole database was selected: convert to individual schema selections
          // for all schemas EXCEPT the unchecked one
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
        // Remove this schema and all its tables
        const filtered = value.filter(
          (r) =>
            !(r.databaseFullName === dbName && r.schema === schemaName) &&
            !(r.databaseFullName === dbName && !r.schema && !r.table)
        );
        onChange(filtered);
      } else {
        // Remove any table-level selections for this schema, add schema-level
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
        // If whole schema was selected, we need to deselect this table
        // by selecting all other tables individually
        const entry = selectedResourceMap.get(dbName);
        if (entry?.wholeDatabaseSelected) {
          // Whole database was selected: need to expand into per-schema selections minus this table
          const metadata = metadataMap.get(dbName);
          if (!metadata) return;
          const newResources = value.filter(
            (r) => r.databaseFullName !== dbName
          );
          for (const schema of metadata.schemas) {
            const sName = schema.name;
            for (const table of schema.tables) {
              if (sName === schemaName && table.name === tableName) continue;
              newResources.push({
                databaseFullName: dbName,
                schema: sName,
                table: table.name,
              });
            }
          }
          onChange(newResources);
          return;
        }
        const schemaEntry = entry?.schemas.get(schemaName);
        if (schemaEntry && schemaEntry.size === 0) {
          // Whole schema was selected: expand into individual tables minus this one
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
        // Individual table was selected: just remove it
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
        onChange([
          ...value,
          { databaseFullName: dbName, schema: schemaName, table: tableName },
        ]);
      }
    },
    [value, onChange, isTableChecked, selectedResourceMap, metadataMap]
  );

  const toggleExpandDatabase = useCallback(
    async (dbName: string) => {
      const next = new Set(expandedDatabases);
      if (next.has(dbName)) {
        next.delete(dbName);
      } else {
        next.add(dbName);
        // Lazy load metadata
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
              const next = new Set(prev);
              next.delete(dbName);
              return next;
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

  const isAllSelected = useMemo(() => {
    if (filteredDatabases.length === 0) return false;
    return filteredDatabases.every((db) => isDatabaseChecked(db.name));
  }, [filteredDatabases, isDatabaseChecked]);

  const toggleSelectAll = useCallback(() => {
    if (isAllSelected) {
      // Remove all filtered databases
      const filteredNames = new Set(filteredDatabases.map((db) => db.name));
      onChange(value.filter((r) => !filteredNames.has(r.databaseFullName)));
    } else {
      // Add all filtered databases that aren't already selected
      const existing = new Set(
        value
          .filter((r) => !r.schema && !r.table)
          .map((r) => r.databaseFullName)
      );
      const filteredNames = new Set(filteredDatabases.map((db) => db.name));
      // Remove partial selections for filtered databases, then add whole-db selections
      const kept = value.filter((r) => !filteredNames.has(r.databaseFullName));
      const toAdd = filteredDatabases
        .filter((db) => !existing.has(db.name))
        .map((db) => ({ databaseFullName: db.name }));
      onChange([...kept, ...toAdd]);
    }
  }, [isAllSelected, filteredDatabases, value, onChange]);

  const removeResource = useCallback(
    (resource: DatabaseResource) => {
      onChange(
        value.filter(
          (r) =>
            !(
              r.databaseFullName === resource.databaseFullName &&
              r.schema === resource.schema &&
              r.table === resource.table
            )
        )
      );
    },
    [value, onChange]
  );

  const resourceLabel = (r: DatabaseResource): string => {
    const { databaseName } = extractDatabaseResourceName(r.databaseFullName);
    if (r.table) return `${databaseName}.${r.schema}.${r.table}`;
    if (r.schema) return `${databaseName}.${r.schema}`;
    return databaseName;
  };

  const hasSchemas = (metadata: DatabaseMetadata): boolean => {
    // If there's only one schema with empty name, it's a MySQL-style DB (no schema concept)
    if (metadata.schemas.length === 1 && metadata.schemas[0].name === "") {
      return false;
    }
    return metadata.schemas.length > 0;
  };

  return (
    <div className="border border-control-border rounded-sm overflow-hidden">
      {/* Filter */}
      <div className="bg-gray-50 border-b border-control-border px-2 py-1.5">
        <SearchInput
          placeholder={t("common.filter")}
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          className="h-7"
        />
      </div>
      {/* Body */}
      <div className="flex" style={{ height: "min(24rem, 60vh)" }}>
        {/* Left panel */}
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
              {t("common.total")} {filteredDatabases.length}{" "}
              {t("common.items", { count: filteredDatabases.length })}
            </span>
          </div>
          <div className="flex-1 overflow-y-auto">
            {filteredDatabases.map((db) => {
              const { databaseName } = extractDatabaseResourceName(db.name);
              const envName = db.effectiveEnvironment ?? db.environment;
              const instanceTitle = db.instanceResource?.title ?? "";
              const isExpanded = expandedDatabases.has(db.name);
              const metadata = metadataMap.get(db.name);
              const isLoading = loadingMetadata.has(db.name);
              const dbChecked = isDatabaseChecked(db.name);
              const dbIndeterminate = isDatabaseIndeterminate(db.name);

              return (
                <div key={db.name}>
                  {/* Database row */}
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
                    <input
                      type="checkbox"
                      className="shrink-0"
                      checked={dbChecked}
                      ref={(el) => {
                        if (el) el.indeterminate = dbIndeterminate;
                      }}
                      onChange={() => toggleDatabase(db)}
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
                  {/* Instance name subtitle */}
                  {instanceTitle && (
                    <div className="pl-10 text-xs text-control-light pb-0.5">
                      ({instanceTitle})
                    </div>
                  )}
                  {/* Expanded content */}
                  {isExpanded && (
                    <div className="pl-6">
                      {isLoading && (
                        <div className="px-2 py-1 text-xs text-control-light">
                          {t("common.loading")}...
                        </div>
                      )}
                      {metadata && !hasSchemas(metadata) && (
                        // MySQL-style: show tables directly under database
                        <>
                          {metadata.schemas[0]?.tables.map((table) => (
                            <TableRow
                              key={table.name}
                              tableName={table.name}
                              checked={isTableChecked(
                                db.name,
                                metadata.schemas[0].name,
                                table.name
                              )}
                              onChange={() =>
                                toggleTable(
                                  db.name,
                                  metadata.schemas[0].name,
                                  table.name
                                )
                              }
                            />
                          ))}
                        </>
                      )}
                      {metadata && hasSchemas(metadata) && (
                        // Postgres-style: show schemas then tables
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
                                  <input
                                    type="checkbox"
                                    className="shrink-0"
                                    checked={schemaChecked}
                                    ref={(el) => {
                                      if (el)
                                        el.indeterminate = schemaIndeterminate;
                                    }}
                                    onChange={() =>
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
                                    {schema.tables.map((table) => (
                                      <TableRow
                                        key={table.name}
                                        tableName={table.name}
                                        checked={isTableChecked(
                                          db.name,
                                          schema.name,
                                          table.name
                                        )}
                                        onChange={() =>
                                          toggleTable(
                                            db.name,
                                            schema.name,
                                            table.name
                                          )
                                        }
                                      />
                                    ))}
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
        {/* Right panel */}
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
                    key={`${r.databaseFullName}/${r.schema ?? ""}/${r.table ?? ""}`}
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

function TableRow({
  tableName,
  checked,
  onChange,
}: {
  tableName: string;
  checked: boolean;
  onChange: () => void;
}) {
  return (
    <div className="flex items-center gap-x-1 px-2 py-1 hover:bg-gray-50">
      <span className="shrink-0 w-4" />
      <input
        type="checkbox"
        className="shrink-0"
        checked={checked}
        onChange={onChange}
      />
      <Table2 className="w-3.5 h-3.5 text-control-light shrink-0" />
      <span className="text-sm truncate">{tableName}</span>
    </div>
  );
}
