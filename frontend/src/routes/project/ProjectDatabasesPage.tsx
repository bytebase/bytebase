import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { Plus, SquareTerminal } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { createBehaviorMetric } from "@/app/analytics/behavior";
import { behaviorAnalytics } from "@/app/analytics/provider";
import { router } from "@/app/router";
import {
  INSTANCE_ROUTE_CREATE,
  SQL_EDITOR_DATABASE_MODULE,
} from "@/app/router/handles";
import { markListScrollRestorationEntry } from "@/app/router/NavigationScrollRestoration";
import {
  AdvancedSearch,
  getValueFromScopes,
  type ScopeOption,
  type SearchParams,
  type ValueOption,
} from "@/components/AdvancedSearch";
import {
  CreateDatabaseSheet,
  DatabaseBatchOperationsBar,
  DatabaseTable,
  LabelEditorSheet,
  TransferProjectSheet,
} from "@/components/database";
import { EditEnvironmentSheet } from "@/components/EditEnvironmentSheet";
import { InstanceLabel } from "@/components/InstanceLabel";
import { PermissionGuard } from "@/components/PermissionGuard";
import {
  ProjectPageLayout,
  ProjectPageToolbar,
} from "@/components/ProjectPageLayout";
import { Alert } from "@/components/ui/alert";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { useProjectByName } from "@/hooks/useProjectByName";
import type { DatabaseFilter } from "@/lib/databaseFilter";
import { preCreateIssue } from "@/lib/plan/issue";
import {
  CONNECT_DATABASE_PRODUCT_INTRO,
  PROJECT_INSTANCE_SYNCED_PRODUCT_INTRO,
  useProductIntro,
} from "@/lib/productIntro";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import {
  environmentNamePrefix,
  instanceNamePrefix,
  projectNamePrefix,
} from "@/stores/modules/v1/common";
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
  extractDatabaseResourceName,
  extractInstanceResourceName,
  getDefaultPagination,
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
  PERMISSIONS_FOR_DATABASE_CREATE_ISSUE,
  supportedEngineV1List,
} from "@/utils";

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
  const [showTransferDrawer, setShowTransferDrawer] = useState(false);
  const [showUnassignConfirm, setShowUnassignConfirm] = useState(false);
  const [refreshToken, setRefreshToken] = useState(0);
  const [workspaceHasInstance, setWorkspaceHasInstance] = useState<
    boolean | undefined
  >(undefined);
  const [syncingRefreshExhausted, setSyncingRefreshExhausted] = useState(false);
  const autoRefreshCountRef = useRef(0);

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
  const syncingInstanceId = useMemo(() => {
    const { syncingInstance } = router.currentRoute.value.query;
    return typeof syncingInstance === "string" && syncingInstance
      ? syncingInstance
      : undefined;
  }, []);
  const syncingInstanceName = syncingInstanceId
    ? `${instanceNamePrefix}${syncingInstanceId}`
    : undefined;

  const selectedDatabases = useMemo(() => {
    if (selectedNames.size === 0) return [];
    return Array.from(selectedNames)
      .filter((name) => isValidDatabaseName(name))
      .map((name) => databasesByName[name] ?? unknownDatabase());
  }, [selectedNames, databasesByName]);

  const selectedDatabaseNames = useMemo(
    () => selectedDatabases.map((db) => db.name),
    [selectedDatabases]
  );
  const canCreateInstance = hasWorkspacePermissionV2("bb.instances.create");

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

  useEffect(() => {
    if (!syncingInstanceId || visibleDatabases.length > 0) {
      setSyncingRefreshExhausted(false);
      return;
    }
    setSyncingRefreshExhausted(false);
    autoRefreshCountRef.current = 0;
    const timer = window.setInterval(() => {
      autoRefreshCountRef.current += 1;
      refresh();
      if (autoRefreshCountRef.current >= 12) {
        setSyncingRefreshExhausted(true);
        window.clearInterval(timer);
      }
    }, 5000);
    return () => window.clearInterval(timer);
  }, [syncingInstanceId, visibleDatabases.length, refresh]);

  useEffect(() => {
    if (!hasWorkspacePermissionV2("bb.instances.list")) {
      setWorkspaceHasInstance(false);
      return;
    }

    let cancelled = false;
    useAppStore
      .getState()
      .fetchInstanceList({ pageSize: 1 })
      .then(({ instances }) => {
        if (!cancelled) {
          setWorkspaceHasInstance(instances.length > 0);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setWorkspaceHasInstance(false);
        }
      });

    return () => {
      cancelled = true;
    };
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

  const handleTransferProject = useCallback(
    async (projectName: string) => {
      try {
        await useAppStore.getState().batchUpdateDatabases(
          create(BatchUpdateDatabasesRequestSchema, {
            parent: "-",
            requests: selectedDatabasesRef.current.map((database) =>
              create(UpdateDatabaseRequestSchema, {
                database: create(DatabaseSchema$, {
                  name: database.name,
                  project: projectName,
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

  const hasVisibleDatabase = visibleDatabases.length > 0;
  const showSyncingInstanceHint =
    !!syncingInstanceId && !hasVisibleDatabase && !syncingRefreshExhausted;
  const showPostSyncNextAction = !!syncingInstanceId && hasVisibleDatabase;
  const checkingWorkspaceInstance =
    !hasVisibleDatabase &&
    !showSyncingInstanceHint &&
    !syncingRefreshExhausted &&
    workspaceHasInstance === undefined;
  const emptyProjectHasInstance =
    !hasVisibleDatabase &&
    (workspaceHasInstance === true || syncingRefreshExhausted) &&
    !showSyncingInstanceHint;
  const emptyProjectShouldConnectInstance =
    !hasVisibleDatabase &&
    workspaceHasInstance === false &&
    !syncingRefreshExhausted &&
    !showSyncingInstanceHint;

  const handleCreateFirstChange = useCallback(() => {
    const firstDatabase = visibleDatabases[0];
    if (!firstDatabase) return;
    behaviorAnalytics.captureMetric(
      createBehaviorMetric("post sync first change clicked", {
        routeId: router.currentRoute.value.name?.toString(),
        resource: projectName,
      })
    );
    preCreateIssue(projectName, [firstDatabase.name]);
  }, [projectName, visibleDatabases]);

  const handleOpenFirstDatabaseInSQLEditor = useCallback(() => {
    const firstDatabase = visibleDatabases[0];
    if (!firstDatabase) return;
    behaviorAnalytics.captureMetric(
      createBehaviorMetric("post sync sql editor clicked", {
        routeId: router.currentRoute.value.name?.toString(),
        resource: projectName,
      })
    );
    const { instanceName, databaseName } = extractDatabaseResourceName(
      firstDatabase.name
    );
    router.push({
      name: SQL_EDITOR_DATABASE_MODULE,
      params: {
        project: projectId,
        instance: instanceName,
        database: databaseName,
      },
    });
  }, [projectId, projectName, visibleDatabases]);

  useProductIntro({
    id: CONNECT_DATABASE_PRODUCT_INTRO,
    title: t("project.connect-instance-intro-title"),
    description: t("project.connect-instance-intro-description"),
    disabled:
      showSyncingInstanceHint ||
      hasVisibleDatabase ||
      workspaceHasInstance !== false ||
      !canCreateInstance,
  });
  useProductIntro({
    id: PROJECT_INSTANCE_SYNCED_PRODUCT_INTRO,
    title: t("db.project-instance-synced-title"),
    description: t("db.project-instance-synced-description"),
    disabled: !showPostSyncNextAction,
  });

  const handleCreateDatabaseAction = useCallback(() => {
    if (checkingWorkspaceInstance) return;
    if (emptyProjectShouldConnectInstance) {
      if (!hasWorkspacePermissionV2("bb.instances.create")) return;
      behaviorAnalytics.captureMetric(
        createBehaviorMetric("connect database clicked", {
          routeId: router.currentRoute.value.name?.toString(),
          resource: projectName,
        })
      );
      router.push({
        name: INSTANCE_ROUTE_CREATE,
        query: { project: projectId },
      });
      return;
    }
    setShowCreateDrawer(true);
  }, [
    checkingWorkspaceInstance,
    emptyProjectShouldConnectInstance,
    projectId,
    projectName,
  ]);

  return (
    <ProjectPageLayout>
      <ProjectPageToolbar className="flex-col items-start gap-2 sm:flex-row sm:items-end">
        <AdvancedSearch
          params={searchParams}
          onParamsChange={setSearchParams}
          placeholder={t("database.filter-database")}
          scopeOptions={scopeOptions}
        />
        {!showSyncingInstanceHint && (
          <PermissionGuard
            permissions={
              hasVisibleDatabase || emptyProjectHasInstance
                ? ["bb.instances.list", "bb.plans.create", "bb.sheets.create"]
                : ["bb.instances.create"]
            }
            project={
              hasVisibleDatabase || emptyProjectHasInstance
                ? project
                : undefined
            }
          >
            <Button
              data-product-intro-target={
                emptyProjectShouldConnectInstance
                  ? CONNECT_DATABASE_PRODUCT_INTRO
                  : undefined
              }
              disabled={
                checkingWorkspaceInstance
                  ? true
                  : hasVisibleDatabase || emptyProjectHasInstance
                    ? !hasProjectPermission("bb.instances.list") ||
                      !PERMISSIONS_FOR_DATABASE_CREATE_ISSUE.every(
                        (permission) => hasProjectPermission(permission)
                      )
                    : !canCreateInstance
              }
              onClick={handleCreateDatabaseAction}
            >
              <Plus className="size-4 mr-1" />
              {hasVisibleDatabase
                ? t("common.create")
                : emptyProjectHasInstance
                  ? t("project.add-database")
                  : t("project.connect-instance")}
            </Button>
          </PermissionGuard>
        )}
      </ProjectPageToolbar>

      {showSyncingInstanceHint && (
        <Alert
          variant="info"
          title={
            <Trans
              t={t}
              i18nKey="db.project-instance-syncing-title"
              components={{
                instance: (
                  <InstanceLabel
                    instanceName={syncingInstanceName ?? ""}
                    link
                  />
                ),
              }}
            />
          }
          description={
            <div className="flex flex-col gap-y-3 sm:flex-row sm:items-center sm:justify-between sm:gap-x-4">
              <span>{t("db.project-instance-syncing-description")}</span>
              <Button size="sm" appearance="outline" onClick={refresh}>
                {t("common.refresh")}
              </Button>
            </div>
          }
        />
      )}

      {showPostSyncNextAction && (
        <Alert
          variant="info"
          data-product-intro-target={PROJECT_INSTANCE_SYNCED_PRODUCT_INTRO}
          title={t("db.project-instance-synced-title")}
          description={
            <div className="flex flex-col gap-y-3">
              <span>{t("db.project-instance-synced-description")}</span>
              <div className="ml-auto flex flex-wrap items-center gap-x-2 gap-y-2">
                <PermissionGuard
                  permissions={PERMISSIONS_FOR_DATABASE_CREATE_ISSUE}
                  project={project}
                >
                  <Button
                    size="sm"
                    appearance="outline"
                    onClick={handleCreateFirstChange}
                  >
                    {t("db.project-instance-synced-action")}
                  </Button>
                </PermissionGuard>
                <PermissionGuard
                  permissions={["bb.sql.select"]}
                  project={project}
                >
                  <Button
                    size="sm"
                    onClick={handleOpenFirstDatabaseInSQLEditor}
                  >
                    <SquareTerminal className="size-4" />
                    {t("db.project-instance-synced-sql-editor-action")}
                  </Button>
                </PermissionGuard>
              </div>
            </div>
          }
        />
      )}

      <DatabaseTable
        filter={filter}
        parent={projectName}
        mode="PROJECT"
        onOpenDatabase={markListScrollRestorationEntry}
        selectedNames={selectedNames}
        onSelectedNamesChange={setSelectedNames}
        onDatabasesChange={setVisibleDatabases}
        refreshToken={refreshToken}
        emptyPlaceholder={
          showSyncingInstanceHint ? (
            <div className="flex flex-col items-center gap-y-3 text-center">
              <div className="text-sm text-control-light">
                {t("db.project-instance-syncing-empty")}
              </div>
              <Button size="sm" appearance="outline" onClick={refresh}>
                {t("common.refresh")}
              </Button>
            </div>
          ) : (
            <span className="text-sm text-control-light">
              {emptyProjectHasInstance
                ? t("project.add-database-empty-placeholder")
                : t("project.connect-instance-empty-placeholder")}
            </span>
          )
        }
      />

      {/* Batch operations bar */}
      <DatabaseBatchOperationsBar
        databases={selectedDatabases}
        project={project}
        onSyncSchema={handleSyncSchema}
        onEditLabels={() => setShowLabelEditor(true)}
        onEditEnvironment={() => setShowEditEnvDrawer(true)}
        onTransferProject={
          isDefault ? () => setShowTransferDrawer(true) : undefined
        }
        onUnassign={isDefault ? undefined : () => setShowUnassignConfirm(true)}
        onChangeDatabase={isDefault ? undefined : handleChangeDatabase}
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
      <TransferProjectSheet
        open={showTransferDrawer}
        databases={selectedDatabases}
        onClose={() => setShowTransferDrawer(false)}
        onTransfer={handleTransferProject}
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
                appearance="secondary"
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
    </ProjectPageLayout>
  );
}
