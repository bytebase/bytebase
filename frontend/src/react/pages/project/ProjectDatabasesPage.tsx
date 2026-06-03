import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { Plus } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  AdvancedSearch,
  getValueFromScopes,
  type ScopeOption,
  type SearchParams,
  type ValueOption,
} from "@/react/components/AdvancedSearch";
import {
  CreateDatabaseSheet,
  DatabaseBatchOperationsBar,
  DatabaseTable,
  LabelEditorSheet,
} from "@/react/components/database";
import { EditEnvironmentSheet } from "@/react/components/EditEnvironmentSheet";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogTitle,
} from "@/react/components/ui/alert-dialog";
import { Button } from "@/react/components/ui/button";
import { useProjectByName } from "@/react/hooks/useProjectByName";
import type { DatabaseFilter } from "@/react/lib/databaseFilter";
import { preCreateIssue } from "@/react/lib/plan/issue";
import { useAppStore } from "@/react/stores/app";
import { pushNotification } from "@/store";
import {
  environmentNamePrefix,
  instanceNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import type { Permission } from "@/types";
import {
  isDefaultProject,
  isValidDatabaseName,
  UNKNOWN_ENVIRONMENT_NAME,
  unknownEnvironment,
} from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  BatchUpdateDatabasesRequestSchema,
  DatabaseSchema$,
  UpdateDatabaseRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { unknownDatabase } from "@/types/v1/database";
import {
  engineNameV1,
  extractInstanceResourceName,
  getDefaultPagination,
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
  PERMISSIONS_FOR_DATABASE_CREATE_ISSUE,
  supportedEngineV1List,
} from "@/utils";
import { DataExportPrepSheet } from "./export-center/DataExportPrepSheet";

export function ProjectDatabasesPage({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const removeDatabaseMetadataCache = useAppStore(
    (s) => s.removeDatabaseMetadataCache
  );
  const databasesByName = useAppStore((s) => s.databasesByName);

  const projectName = `${projectNamePrefix}${projectId}`;
  // subscribe to re-render on project cache change
  const projectsByName = useAppStore((s) => s.projectsByName);
  void projectsByName;
  const project = useProjectByName(projectName);
  const isDefault = isDefaultProject(projectName);

  const hasProjectPermission = useCallback(
    (permission: Permission) =>
      project ? hasProjectPermissionV2(project, permission) : false,
    [project]
  );

  const [syncing, setSyncing] = useState(false);
  const [showCreateDrawer, setShowCreateDrawer] = useState(false);
  const [showLabelEditor, setShowLabelEditor] = useState(false);
  const [showEditEnvDrawer, setShowEditEnvDrawer] = useState(false);
  const [showUnassignConfirm, setShowUnassignConfirm] = useState(false);
  const [refreshToken, setRefreshToken] = useState(0);

  const [searchParams, setSearchParams] = useState<SearchParams>({
    query: "",
    scopes: [],
  });

  // Reset state when navigating between projects
  useEffect(() => {
    setSelectedNames(new Set());
    setSearchParams({ query: "", scopes: [] });
    setRefreshToken((prev) => prev + 1);
  }, [projectId]);

  const environments = useAppStore((s) => s.environmentList);

  const searchInstances = useCallback(
    async (keyword: string): Promise<ValueOption[]> => {
      if (!hasWorkspacePermissionV2("bb.instances.list")) return [];
      const { instances } = await useAppStore.getState().fetchInstanceList({
        pageSize: getDefaultPagination(),
        filter: keyword.trim() ? { query: keyword } : undefined,
      });
      return instances.map((i) => {
        const id = extractInstanceResourceName(i.name);
        return { value: id, keywords: [id, i.title] };
      });
    },
    []
  );

  const scopeOptions: ScopeOption[] = useMemo(() => {
    return [
      {
        id: "environment",
        title: t("common.environment"),
        description: t("common.environment"),
        options: [unknownEnvironment(), ...environments].map((env) => {
          const isUnknown = env.name === UNKNOWN_ENVIRONMENT_NAME;
          return {
            value: env.id,
            keywords: isUnknown
              ? ["unassigned", "none", env.id]
              : [env.id, env.title],
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
        title: t("common.instance"),
        description: t("issue.advanced-search.scope.instance.description"),
        onSearch: searchInstances,
      },
      {
        id: "engine",
        title: t("database.engine"),
        description: t("database.engine"),
        options: supportedEngineV1List().map((engine) => ({
          value: Engine[engine],
          keywords: [Engine[engine].toLowerCase(), engineNameV1(engine)],
        })),
        allowMultiple: true,
      },
      {
        id: "label",
        title: t("common.labels"),
        description: t("issue.advanced-search.scope.label.description"),
        allowMultiple: true,
      },
    ];
  }, [t, environments, searchInstances]);

  // Derived filter values
  const envVal = getValueFromScopes(searchParams, "environment");
  const selectedEnvironment = envVal
    ? `${environmentNamePrefix}${envVal}`
    : undefined;

  const instanceVal = getValueFromScopes(searchParams, "instance");
  const selectedInstance = instanceVal
    ? `${instanceNamePrefix}${instanceVal}`
    : undefined;

  const selectedEngines = useMemo(
    () =>
      searchParams.scopes
        .filter((s) => s.id === "engine")
        .map((s) => Engine[s.value as keyof typeof Engine])
        .filter((e): e is Engine => e !== undefined),
    [searchParams]
  );

  const selectedLabels = useMemo(
    () =>
      searchParams.scopes.filter((s) => s.id === "label").map((s) => s.value),
    [searchParams]
  );

  const filter: DatabaseFilter = useMemo(
    () => ({
      instance: selectedInstance,
      environment: selectedEnvironment,
      query: searchParams.query,
      labels: selectedLabels.length > 0 ? selectedLabels : undefined,
      engines: selectedEngines,
    }),
    [
      selectedInstance,
      selectedEnvironment,
      searchParams.query,
      selectedLabels,
      selectedEngines,
    ]
  );

  // Selection state
  const [selectedNames, setSelectedNames] = useState<Set<string>>(new Set());
  const [visibleDatabases, setVisibleDatabases] = useState<Database[]>([]);

  const selectedDatabases = useMemo(() => {
    if (selectedNames.size === 0) return [];
    return Array.from(selectedNames)
      .filter((name) => isValidDatabaseName(name))
      .map((name) => databasesByName[name] ?? unknownDatabase());
  }, [selectedNames, databasesByName]);

  // Stable references for downstream sheets. Without these the JSX would
  // rebuild a fresh array + fresh object literal on every parent re-render
  // (e.g. opening any drawer flips state on this page) — DataExportPrepSheet
  // would then see a new `seed` prop reference each render and re-run its
  // `seedKey = selectedDatabaseNames.join(",")` work (O(N) on every render).
  const selectedDatabaseNames = useMemo(
    () => selectedDatabases.map((db) => db.name),
    [selectedDatabases]
  );
  const exportPrepSeed = useMemo(
    () => ({ selectedDatabaseNames, step: 2 as const }),
    [selectedDatabaseNames]
  );

  // Mirror `selectedDatabases` into a ref so the batch-operation handlers
  // below can read the latest value without listing it as a dep. Otherwise
  // every selection toggle re-creates the handler closures, which cascades
  // down as fresh prop refs into `DatabaseBatchOperationsBar` and forces
  // it to re-render (with N selected items, this compounds quickly).
  const selectedDatabasesRef = useRef(selectedDatabases);
  selectedDatabasesRef.current = selectedDatabases;

  const refresh = useCallback(() => {
    setRefreshToken((prev) => prev + 1);
    setSelectedNames(new Set());
  }, []);

  // Batch operation handlers
  const handleSyncSchema = useCallback(async () => {
    if (syncing) return;
    setSyncing(true);
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: t("db.start-to-sync-schema"),
    });
    try {
      await useAppStore
        .getState()
        .batchSyncDatabases(Array.from(selectedNames));
      for (const name of selectedNames) {
        removeDatabaseMetadataCache(name);
      }
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("db.successfully-synced-schema"),
      });
      setSelectedNames(new Set());
    } catch {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("db.failed-to-sync-schema"),
      });
    } finally {
      setSyncing(false);
    }
  }, [syncing, selectedNames, removeDatabaseMetadataCache, t]);

  const handleLabelsApply = useCallback(
    async (labelsList: { [key: string]: string }[]) => {
      try {
        await useAppStore.getState().batchUpdateDatabases(
          create(BatchUpdateDatabasesRequestSchema, {
            parent: "-",
            requests: selectedDatabasesRef.current.map((database, i) =>
              create(UpdateDatabaseRequestSchema, {
                database: create(DatabaseSchema$, {
                  ...database,
                  labels: labelsList[i],
                }),
                updateMask: create(FieldMaskSchema, { paths: ["labels"] }),
              })
            ),
          })
        );
        refresh();
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.updated"),
        });
      } catch {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("common.failed"),
        });
      }
    },
    [refresh, t]
  );

  const handleEnvironmentUpdate = useCallback(
    async (environment: string) => {
      try {
        await useAppStore.getState().batchUpdateDatabases(
          create(BatchUpdateDatabasesRequestSchema, {
            parent: "-",
            requests: selectedDatabasesRef.current.map((database) =>
              create(UpdateDatabaseRequestSchema, {
                database: create(DatabaseSchema$, {
                  name: database.name,
                  environment,
                }),
                updateMask: create(FieldMaskSchema, { paths: ["environment"] }),
              })
            ),
          })
        );
        refresh();
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.updated"),
        });
      } catch {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("common.failed"),
        });
      }
    },
    [refresh, t]
  );

  const handleUnassign = useCallback(async () => {
    const defaultProject =
      useAppStore.getState().serverInfo?.defaultProject ?? "";
    try {
      await useAppStore.getState().batchUpdateDatabases(
        create(BatchUpdateDatabasesRequestSchema, {
          parent: "-",
          requests: selectedDatabasesRef.current.map((database) =>
            create(UpdateDatabaseRequestSchema, {
              database: create(DatabaseSchema$, {
                name: database.name,
                project: defaultProject,
              }),
              updateMask: create(FieldMaskSchema, { paths: ["project"] }),
            })
          ),
        })
      );
      refresh();
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("database.successfully-transferred-databases"),
      });
    } catch {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
      });
    }
  }, [refresh, t]);

  const handleChangeDatabase = useCallback(() => {
    preCreateIssue(projectName, selectedDatabaseNames);
  }, [projectName, selectedDatabaseNames]);

  const [showExportDrawer, setShowExportDrawer] = useState(false);

  const handleExportData = useCallback(() => {
    setShowExportDrawer(true);
  }, []);

  return (
    <div className="py-4 flex flex-col">
      <div className="px-4 flex flex-col gap-y-2 pb-2">
        <div className="w-full flex flex-col sm:flex-row items-start sm:items-end justify-between gap-2">
          <AdvancedSearch
            params={searchParams}
            onParamsChange={setSearchParams}
            placeholder={t("database.filter-database")}
            scopeOptions={scopeOptions}
          />
          <PermissionGuard
            permissions={[
              "bb.instances.list",
              "bb.plans.create",
              "bb.sheets.create",
            ]}
            project={project}
          >
            <Button
              disabled={
                !hasProjectPermission("bb.instances.list") ||
                !PERMISSIONS_FOR_DATABASE_CREATE_ISSUE.every(
                  hasProjectPermission
                )
              }
              onClick={() => setShowCreateDrawer(true)}
            >
              <Plus className="size-4 mr-1" />
              {t("common.create")}
            </Button>
          </PermissionGuard>
        </div>
      </div>

      <DatabaseTable
        filter={filter}
        parent={projectName}
        mode="PROJECT"
        selectedNames={selectedNames}
        onSelectedNamesChange={setSelectedNames}
        onDatabasesChange={setVisibleDatabases}
        refreshToken={refreshToken}
      />

      {/* Batch operations bar */}
      <DatabaseBatchOperationsBar
        databases={selectedDatabases}
        project={project}
        onSyncSchema={handleSyncSchema}
        onEditLabels={() => setShowLabelEditor(true)}
        onEditEnvironment={() => setShowEditEnvDrawer(true)}
        onUnassign={isDefault ? undefined : () => setShowUnassignConfirm(true)}
        onChangeDatabase={isDefault ? undefined : handleChangeDatabase}
        onExportData={isDefault ? undefined : handleExportData}
        allSelected={
          visibleDatabases.length > 0 &&
          visibleDatabases.every((d) => selectedNames.has(d.name))
        }
        onToggleSelectAll={() => {
          const allOnPage =
            visibleDatabases.length > 0 &&
            visibleDatabases.every((d) => selectedNames.has(d.name));
          if (allOnPage) setSelectedNames(new Set());
          else setSelectedNames(new Set(visibleDatabases.map((d) => d.name)));
        }}
      />

      {/* Modals (portaled, position-independent) */}
      <CreateDatabaseSheet
        open={showCreateDrawer}
        onClose={() => setShowCreateDrawer(false)}
        projectName={projectName}
      />
      <EditEnvironmentSheet
        open={showEditEnvDrawer}
        onClose={() => setShowEditEnvDrawer(false)}
        onUpdate={handleEnvironmentUpdate}
      />
      <LabelEditorSheet
        open={showLabelEditor}
        databases={selectedDatabases}
        onClose={() => setShowLabelEditor(false)}
        onApply={handleLabelsApply}
      />

      <DataExportPrepSheet
        open={showExportDrawer}
        onClose={() => setShowExportDrawer(false)}
        projectName={projectName}
        seed={exportPrepSeed}
      />

      {/* Unassign confirmation dialog */}
      {showUnassignConfirm && (
        <AlertDialog
          open
          onOpenChange={(nextOpen) =>
            !nextOpen && setShowUnassignConfirm(false)
          }
        >
          <AlertDialogContent>
            <AlertDialogTitle>
              {t("database.unassign-alert-title")}
            </AlertDialogTitle>
            <AlertDialogDescription className="mt-2">
              {t("database.unassign-alert-description")}
            </AlertDialogDescription>
            <div className="mt-6 flex items-center justify-end gap-x-2">
              <Button
                variant="ghost"
                onClick={() => setShowUnassignConfirm(false)}
              >
                {t("common.cancel")}
              </Button>
              <Button
                onClick={async () => {
                  setShowUnassignConfirm(false);
                  await handleUnassign();
                }}
              >
                {t("common.confirm")}
              </Button>
            </div>
          </AlertDialogContent>
        </AlertDialog>
      )}
    </div>
  );
}
