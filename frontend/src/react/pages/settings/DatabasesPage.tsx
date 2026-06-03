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
  TransferProjectSheet,
} from "@/react/components/database";
import { EditEnvironmentSheet } from "@/react/components/EditEnvironmentSheet";
import { EngineIcon } from "@/react/components/EngineIcon";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import { Button } from "@/react/components/ui/button";
import type { DatabaseFilter } from "@/react/lib/databaseFilter";
import { useAppStore } from "@/react/stores/app";
import { router } from "@/router";
import { pushNotification } from "@/store";
import {
  environmentNamePrefix,
  instanceNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
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
            useAppStore.getState().serverInfo?.defaultProject ?? ""
          ),
        },
      ],
    };
  });

  const environments = useAppStore((s) => s.environmentList);

  // `serverInfo.defaultProject` is fetched asynchronously by the actuator
  // store; use a selector so the filter value updates the moment it arrives
  // instead of being captured as an empty string on first render (which
  // sends a broken `projects/` filter to the backend).
  const defaultProjectId = useAppStore((s) =>
    extractProjectResourceName(s.serverInfo?.defaultProject ?? "")
  );
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

  // Backfill the project scope once the actuator's default project ID
  // arrives. The initial `useState` initializer reads the actuator
  // synchronously — if it hasn't finished fetching yet, the project value
  // is captured as "" and the API filter becomes broken (`projects/`).
  useEffect(() => {
    if (!defaultProjectId) return;
    setSearchParams((prev) => {
      const projectScope = prev.scopes.find((s) => s.id === "project");
      if (!projectScope || projectScope.value !== "") return prev;
      return {
        ...prev,
        scopes: prev.scopes.map((s) =>
          s.id === "project" && s.value === ""
            ? { ...s, value: defaultProjectId }
            : s
        ),
      };
    });
  }, [defaultProjectId]);

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
