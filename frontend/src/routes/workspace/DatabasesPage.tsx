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
} from "@/components/AdvancedSearch";
import {
  CreateDatabaseSheet,
  DatabaseBatchOperationsBar,
  DatabaseTable,
  LabelEditorSheet,
  TransferProjectSheet,
} from "@/components/database";
import { EditEnvironmentSheet } from "@/components/EditEnvironmentSheet";
import { EngineIcon } from "@/components/EngineIcon";
import { EnvironmentLabel } from "@/components/EnvironmentLabel";
import { PermissionGuard } from "@/components/PermissionGuard";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  WorkspacePageLayout,
  WorkspacePageToolbar,
} from "@/components/WorkspacePageLayout";
import {
  createAdvancedSearchParser,
  serializeAdvancedSearch,
  useURLSearchParam,
} from "@/hooks/useURLSearchParam";
import type { DatabaseFilter } from "@/lib/databaseFilter";
import {
  PREPARE_DATABASE_PRODUCT_INTRO,
  PREPARE_DATABASE_TRANSFER_TIP,
  PRODUCT_INTRO_TIP_QUERY_KEY,
  useProductIntro,
} from "@/lib/productIntro";
import { useCurrentRoute } from "@/app/router";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import {
  environmentNamePrefix,
  instanceNamePrefix,
  projectNamePrefix,
} from "@/stores/modules/v1/common";
import {
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
import {
  engineNameV1,
  extractInstanceResourceName,
  extractProjectResourceName,
  getDefaultPagination,
  hasWorkspacePermissionV2,
  supportedEngineV1List,
} from "@/utils";

const parseDatabaseSearch = createAdvancedSearchParser([
  "project",
  "environment",
  "instance",
  "engine",
  "label",
]);

export function DatabasesPage() {
  const { t } = useTranslation();
  const currentRoute = useCurrentRoute();
  const databasesByName = useAppStore((s) => s.databasesByName);
  const getDatabaseByName = useAppStore((s) => s.getDatabaseByName);
  const removeDatabaseMetadataCache = useAppStore(
    (s) => s.removeDatabaseMetadataCache
  );

  const [syncing, setSyncing] = useState(false);
  const [showCreateDrawer, setShowCreateDrawer] = useState(false);
  const [showLabelEditor, setShowLabelEditor] = useState(false);
  const [showEditEnvDrawer, setShowEditEnvDrawer] = useState(false);
  const [showTransferDrawer, setShowTransferDrawer] = useState(false);
  const [refreshToken, setRefreshToken] = useState(0);

  const environments = useAppStore((s) => s.environmentList);

  // `serverInfo.defaultProject` is fetched asynchronously by the actuator
  // store; use a selector so the filter value updates the moment it arrives
  // instead of being captured as an empty string on first render (which
  // sends a broken `projects/` filter to the backend).
  const defaultProjectId = useAppStore((s) =>
    extractProjectResourceName(s.serverInfo?.defaultProject ?? "")
  );
  const defaultSearchParams = useMemo<SearchParams>(
    () => ({
      query: "",
      scopes: defaultProjectId
        ? [{ id: "project", value: defaultProjectId }]
        : [],
    }),
    [defaultProjectId]
  );
  const [searchParams, setSearchParams] = useURLSearchParam<SearchParams>({
    param: "q",
    parse: parseDatabaseSearch,
    serialize: serializeAdvancedSearch,
    defaultValue: defaultSearchParams,
  });
  // Shared "Unassigned" option used both by the dropdown (via onSearch) and
  // by the selected-tag display (via the scope's static `options`). `custom`
  // hides the raw "default-<random>" id from the dropdown so users only see
  // the friendly label.
  const unassignedProjectOption = useMemo<ValueOption>(
    () => ({
      value: defaultProjectId,
      keywords: ["unassigned", "default"],
      custom: true,
      render: () => (
        <span className="italic text-control-light">
          {t("common.unassigned")}
        </span>
      ),
    }),
    [defaultProjectId, t]
  );
  const searchProjects = useCallback(
    async (keyword: string): Promise<ValueOption[]> => {
      const { projects } = await useAppStore.getState().fetchProjectList({
        pageSize: getDefaultPagination(),
        filter: keyword.trim() ? { query: keyword } : undefined,
      });
      const matchesUnassigned =
        !keyword.trim() || "unassigned".includes(keyword.trim().toLowerCase());
      const remote = projects
        .filter((p) => extractProjectResourceName(p.name) !== defaultProjectId)
        .map<ValueOption>((p) => {
          const id = extractProjectResourceName(p.name);
          return { value: id, keywords: [id, p.title] };
        });
      return matchesUnassigned ? [unassignedProjectOption, ...remote] : remote;
    },
    [defaultProjectId, unassignedProjectOption]
  );

  const searchInstances = useCallback(
    async (keyword: string): Promise<ValueOption[]> => {
      if (!hasWorkspacePermissionV2("bb.instances.list")) return [];
      const { instances } = await useAppStore.getState().fetchInstanceList({
        pageSize: getDefaultPagination(),
        filter: keyword.trim() ? { query: keyword } : undefined,
      });
      return instances.map((i) => {
        const id = extractInstanceResourceName(i.name);
        return {
          value: id,
          keywords: [id, i.title],
        };
      });
    },
    []
  );

  const scopeOptions: ScopeOption[] = useMemo(() => {
    return [
      {
        id: "project",
        title: t("common.project"),
        description: t("issue.advanced-search.scope.project.description"),
        // Static option lets the selected-tag display resolve the default
        // project id to "Unassigned" — the tag renderer only looks at
        // `options`, not async results.
        options: [unassignedProjectOption],
        onSearch: searchProjects,
      },
      {
        id: "environment",
        title: t("common.environment"),
        description: t("issue.advanced-search.scope.environment.description"),
        options: [unknownEnvironment(), ...environments].map((env) => {
          const isUnknown = env.name === UNKNOWN_ENVIRONMENT_NAME;
          return {
            value: env.id,
            keywords: isUnknown
              ? ["unassigned", "none", env.id]
              : [env.id, env.title],
            custom: true,
            render: () => <EnvironmentLabel environment={env} />,
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
        description: t("issue.advanced-search.scope.engine.description"),
        options: supportedEngineV1List().map((engine) => ({
          value: Engine[engine],
          keywords: [Engine[engine].toLowerCase(), engineNameV1(engine)],
          custom: true,
          render: () => (
            <span className="inline-flex items-center gap-x-1.5">
              <EngineIcon engine={engine} className="h-4 w-4" />
              <span>{engineNameV1(engine)}</span>
            </span>
          ),
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
  }, [
    t,
    environments,
    searchInstances,
    searchProjects,
    unassignedProjectOption,
  ]);

  // Derived filter values
  const projectVal = getValueFromScopes(searchParams, "project");
  const selectedProject = projectVal
    ? `${projectNamePrefix}${projectVal}`
    : undefined;

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
      project: selectedProject,
      instance: selectedInstance,
      environment: selectedEnvironment,
      query: searchParams.query,
      labels: selectedLabels.length > 0 ? selectedLabels : undefined,
      excludeUnassigned: false,
      engines: selectedEngines,
    }),
    [
      selectedProject,
      selectedInstance,
      selectedEnvironment,
      searchParams.query,
      selectedLabels,
      selectedEngines,
    ]
  );

  // Mark database visit on mount
  useEffect(() => {
    const store = useAppStore.getState();
    if (!store.getIntroStateByKey("database.visit")) {
      store.saveIntroStateByKey({
        key: "database.visit",
        newState: true,
      });
    }
  }, []);

  const showPrepareDatabaseTip =
    currentRoute.query[PRODUCT_INTRO_TIP_QUERY_KEY] ===
    PREPARE_DATABASE_TRANSFER_TIP;

  // Selection state
  const [selectedNames, setSelectedNames] = useState<Set<string>>(new Set());
  const [visibleDatabases, setVisibleDatabases] = useState<Database[]>([]);
  const [databaseTableLoading, setDatabaseTableLoading] = useState(true);

  const shouldShowTransferIntro =
    showPrepareDatabaseTip &&
    !databaseTableLoading &&
    visibleDatabases.length > 0;
  const shouldShowCreateDatabaseIntro =
    !showPrepareDatabaseTip ||
    (!databaseTableLoading && visibleDatabases.length === 0);
  useProductIntro({
    id: PREPARE_DATABASE_PRODUCT_INTRO,
    title: shouldShowTransferIntro
      ? t("workspace-setup-guide.intro.transfer-title")
      : t("workspace-setup-guide.intro.database-title"),
    description: shouldShowTransferIntro
      ? t("workspace-setup-guide.intro.transfer-description")
      : t("workspace-setup-guide.intro.database-description"),
    ...(showPrepareDatabaseTip && databaseTableLoading
      ? { disabled: true }
      : {}),
  });

  const selectedDatabases = useMemo(() => {
    if (selectedNames.size === 0) return [];
    return Array.from(selectedNames)
      .filter((name) => isValidDatabaseName(name))
      .map((name) => getDatabaseByName(name));
  }, [selectedNames, getDatabaseByName, databasesByName]);

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

  return (
    <WorkspacePageLayout padding="flush" className="relative">
      <WorkspacePageToolbar className="px-4 flex-col items-start gap-2 sm:flex-row sm:items-end">
        <AdvancedSearch
          params={searchParams}
          onParamsChange={setSearchParams}
          placeholder={t("database.filter-database")}
          scopeOptions={scopeOptions}
        />
        <PermissionGuard
          permissions={["bb.instances.list", "bb.issues.create"]}
        >
          <Button
            data-product-intro-target={
              shouldShowCreateDatabaseIntro
                ? PREPARE_DATABASE_PRODUCT_INTRO
                : undefined
            }
            disabled={
              !hasWorkspacePermissionV2("bb.instances.list") ||
              !hasWorkspacePermissionV2("bb.issues.create")
            }
            onClick={() => setShowCreateDrawer(true)}
          >
            <Plus className="h-4 w-4 mr-1" />
            {t("database.create-database")}
          </Button>
        </PermissionGuard>
      </WorkspacePageToolbar>

      {shouldShowTransferIntro && (
        <Alert className="mx-4 w-auto" variant="info">
          {t("workspace-setup-guide.prepare-database-tip")}
        </Alert>
      )}

      <DatabaseTable
        filter={filter}
        mode="ALL"
        selectedNames={selectedNames}
        onSelectedNamesChange={setSelectedNames}
        onDatabasesChange={setVisibleDatabases}
        onLoadingChange={setDatabaseTableLoading}
        refreshToken={refreshToken}
        selectionColumnIntroTarget={
          shouldShowTransferIntro ? PREPARE_DATABASE_PRODUCT_INTRO : undefined
        }
      />

      {/* Batch operations bar */}
      <DatabaseBatchOperationsBar
        databases={selectedDatabases}
        onSyncSchema={handleSyncSchema}
        onEditLabels={() => setShowLabelEditor(true)}
        onEditEnvironment={() => setShowEditEnvDrawer(true)}
        onTransferProject={() => setShowTransferDrawer(true)}
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
    </WorkspacePageLayout>
  );
}
