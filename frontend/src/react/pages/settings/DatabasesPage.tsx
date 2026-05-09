import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { Plus } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
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
  TransferProjectSheet,
} from "@/react/components/database";
import { EditEnvironmentSheet } from "@/react/components/EditEnvironmentSheet";
import { EngineIcon } from "@/react/components/EngineIcon";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import { Button } from "@/react/components/ui/button";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import {
  pushNotification,
  useActuatorV1Store,
  useDatabaseV1Store,
  useDBSchemaV1Store,
  useEnvironmentV1Store,
  useInstanceV1Store,
  useProjectV1Store,
  useUIStateStore,
} from "@/store";
import {
  environmentNamePrefix,
  instanceNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import type { DatabaseFilter } from "@/store/modules/v1/database";
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

export function DatabasesPage() {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();
  const dbSchemaStore = useDBSchemaV1Store();
  const actuatorStore = useActuatorV1Store();
  const environmentStore = useEnvironmentV1Store();
  const uiStateStore = useUIStateStore();

  const [syncing, setSyncing] = useState(false);
  const [showCreateDrawer, setShowCreateDrawer] = useState(false);
  const [showLabelEditor, setShowLabelEditor] = useState(false);
  const [showEditEnvDrawer, setShowEditEnvDrawer] = useState(false);
  const [showTransferDrawer, setShowTransferDrawer] = useState(false);
  const [refreshToken, setRefreshToken] = useState(0);

  // Search state — default to showing unassigned databases from default project
  const [searchParams, setSearchParams] = useState<SearchParams>(() => {
    const currentRoute = router.currentRoute.value;
    const hasQ = "q" in (currentRoute.query ?? {});
    const queryString = (currentRoute.query.q as string) ?? "";
    if (hasQ) {
      // URL has an explicit `q` param (may be empty if the user cleared all
      // filters) — parse it and do NOT re-apply the default project scope.
      const scopes: { id: string; value: string }[] = [];
      const queryParts: string[] = [];
      for (const token of queryString.split(/\s+/).filter(Boolean)) {
        const colonIdx = token.indexOf(":");
        if (colonIdx > 0) {
          const id = token.substring(0, colonIdx);
          const value = token.substring(colonIdx + 1);
          if (
            value &&
            ["project", "environment", "instance", "engine", "label"].includes(
              id
            )
          ) {
            scopes.push({ id, value });
            continue;
          }
        }
        queryParts.push(token);
      }
      return { query: queryParts.join(" "), scopes };
    }
    // First visit (no `q` in URL) — default to the unassigned project.
    return {
      query: "",
      scopes: [
        {
          id: "project",
          value: extractProjectResourceName(
            actuatorStore.serverInfo?.defaultProject ?? ""
          ),
        },
      ],
    };
  });

  const environments = useVueState(
    () => environmentStore.environmentList ?? []
  );

  const projectStore = useProjectV1Store();
  const defaultProjectId = extractProjectResourceName(
    actuatorStore.serverInfo?.defaultProject ?? ""
  );
  const searchProjects = useCallback(
    async (keyword: string): Promise<ValueOption[]> => {
      const { projects } = await projectStore.fetchProjectList({
        pageSize: getDefaultPagination(),
        filter: keyword.trim() ? { query: keyword } : undefined,
      });
      const unassigned: ValueOption = {
        value: defaultProjectId,
        keywords: ["unassigned", "default"],
        render: () => (
          <span className="italic text-control-light">Unassigned</span>
        ),
      };
      const matchesUnassigned =
        !keyword.trim() || "unassigned".includes(keyword.trim().toLowerCase());
      const remote = projects
        .filter((p) => extractProjectResourceName(p.name) !== defaultProjectId)
        .map<ValueOption>((p) => {
          const id = extractProjectResourceName(p.name);
          return { value: id, keywords: [id, p.title] };
        });
      return matchesUnassigned ? [unassigned, ...remote] : remote;
    },
    [projectStore, defaultProjectId]
  );

  const instanceStore = useInstanceV1Store();
  const searchInstances = useCallback(
    async (keyword: string): Promise<ValueOption[]> => {
      if (!hasWorkspacePermissionV2("bb.instances.list")) return [];
      const { instances } = await instanceStore.fetchInstanceList({
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
    [instanceStore]
  );

  const scopeOptions: ScopeOption[] = useMemo(() => {
    return [
      {
        id: "project",
        title: t("common.project"),
        description: t("issue.advanced-search.scope.project.description"),
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
  }, [t, environments, searchInstances, searchProjects]);

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
    if (!uiStateStore.getIntroStateByKey("database.visit")) {
      uiStateStore.saveIntroStateByKey({
        key: "database.visit",
        newState: true,
      });
    }
  }, [uiStateStore]);

  // Sync search state to URL
  useEffect(() => {
    const parts: string[] = [];
    for (const scope of searchParams.scopes) {
      parts.push(`${scope.id}:${scope.value}`);
    }
    if (searchParams.query) parts.push(searchParams.query);
    const queryString = parts.join(" ");
    const currentQuery = router.currentRoute.value.query.q as string;
    if (queryString !== (currentQuery ?? "")) {
      router.replace({ query: { q: queryString } });
    }
  }, [searchParams]);

  // Selection state
  const [selectedNames, setSelectedNames] = useState<Set<string>>(new Set());
  const [visibleDatabases, setVisibleDatabases] = useState<Database[]>([]);

  const selectedDatabases = useMemo(() => {
    if (selectedNames.size === 0) return [];
    return Array.from(selectedNames)
      .filter((name) => isValidDatabaseName(name))
      .map((name) => databaseStore.getDatabaseByName(name));
  }, [selectedNames, databaseStore]);

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
      await databaseStore.batchSyncDatabases(Array.from(selectedNames));
      for (const name of selectedNames) {
        dbSchemaStore.removeCache(name);
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
  }, [syncing, selectedNames, databaseStore, dbSchemaStore, t]);

  const handleLabelsApply = useCallback(
    async (labelsList: { [key: string]: string }[]) => {
      try {
        await databaseStore.batchUpdateDatabases(
          create(BatchUpdateDatabasesRequestSchema, {
            parent: "-",
            requests: selectedDatabases.map((database, i) =>
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
    [selectedDatabases, databaseStore, refresh, t]
  );

  const handleEnvironmentUpdate = useCallback(
    async (environment: string) => {
      try {
        await databaseStore.batchUpdateDatabases(
          create(BatchUpdateDatabasesRequestSchema, {
            parent: "-",
            requests: selectedDatabases.map((database) =>
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
    [selectedDatabases, databaseStore, refresh, t]
  );

  const handleTransferProject = useCallback(
    async (projectName: string) => {
      try {
        await databaseStore.batchUpdateDatabases(
          create(BatchUpdateDatabasesRequestSchema, {
            parent: "-",
            requests: selectedDatabases.map((database) =>
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
    [selectedDatabases, databaseStore, refresh, t]
  );

  return (
    <div className="py-4 flex flex-col relative">
      <div className="w-full px-4 pb-2 flex flex-col sm:flex-row items-start sm:items-end justify-between gap-2">
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
            disabled={
              !hasWorkspacePermissionV2("bb.instances.list") ||
              !hasWorkspacePermissionV2("bb.issues.create")
            }
            onClick={() => setShowCreateDrawer(true)}
          >
            <Plus className="h-4 w-4 mr-1" />
            {t("common.create")}
          </Button>
        </PermissionGuard>
      </div>

      <DatabaseTable
        filter={filter}
        mode="ALL"
        selectedNames={selectedNames}
        onSelectedNamesChange={setSelectedNames}
        onDatabasesChange={setVisibleDatabases}
        refreshToken={refreshToken}
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
    </div>
  );
}
