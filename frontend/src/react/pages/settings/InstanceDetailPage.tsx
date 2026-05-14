import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
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
  DatabaseBatchOperationsBar,
  DatabaseTable,
  LabelEditorSheet,
  TransferProjectSheet,
} from "@/react/components/database";
import { EditEnvironmentSheet } from "@/react/components/EditEnvironmentSheet";
import { EngineIcon } from "@/react/components/EngineIcon";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import {
  InstanceActionDropdown,
  InstanceFormBody,
  InstanceFormButtons,
  InstanceFormProvider,
  InstanceRoleTable,
  InstanceSyncButton,
  useInstanceFormContext,
} from "@/react/components/instance";
import { Alert } from "@/react/components/ui/alert";
import {
  Tabs,
  TabsList,
  TabsPanel,
  TabsTrigger,
} from "@/react/components/ui/tabs";
import { useUnsavedChangesGuard } from "@/react/hooks/useUnsavedChangesGuard";
import { useVueState } from "@/react/hooks/useVueState";
import {
  pushNotification,
  useDatabaseV1Store,
  useDBSchemaV1Store,
  useEnvironmentV1Store,
  useInstanceV1Store,
  useProjectV1Store,
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
import { State } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  BatchUpdateDatabasesRequestSchema,
  DatabaseSchema$,
  UpdateDatabaseRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import {
  extractProjectResourceName,
  getDefaultPagination,
  instanceV1Name,
  setDocumentTitle,
} from "@/utils";

const instanceHashList = ["overview", "databases", "users"] as const;
type InstanceHash = (typeof instanceHashList)[number];
const isInstanceHash = (x: unknown): x is InstanceHash =>
  instanceHashList.includes(x as InstanceHash);

export function InstanceDetailPage({ instanceId }: { instanceId: string }) {
  const { t } = useTranslation();
  const instanceStore = useInstanceV1Store();
  const databaseStore = useDatabaseV1Store();
  const dbSchemaStore = useDBSchemaV1Store();
  const instanceName = `${instanceNamePrefix}${instanceId}`;
  const instance = useVueState(() =>
    instanceStore.getInstanceByName(instanceName)
  );

  const [selectedTab, setSelectedTab] = useState<InstanceHash>("overview");
  const [searchParams, setSearchParams] = useState<SearchParams>({
    query: "",
    scopes: [{ id: "instance", value: instanceId, readonly: true }],
  });

  // Selection / batch operations
  const [selectedNames, setSelectedNames] = useState<Set<string>>(new Set());
  const [visibleDatabases, setVisibleDatabases] = useState<Database[]>([]);
  const [refreshToken, setRefreshToken] = useState(0);
  const [syncing, setSyncing] = useState(false);
  const [showLabelEditor, setShowLabelEditor] = useState(false);
  const [showEditEnvDrawer, setShowEditEnvDrawer] = useState(false);
  const [showTransferDrawer, setShowTransferDrawer] = useState(false);

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
  // Trigger a Pinia-side fetch on mount. The parent `InstanceRouteShell`
  // populates the React-side `useAppStore` cache, but this page reads
  // from the Pinia v1 store (`useInstanceV1Store`) — they're separate
  // caches. Without this, hard-refreshing the page shows "Unknown
  // instance" because the Pinia cache hasn't been hydrated.
  useEffect(() => {
    void instanceStore.getOrFetchInstanceByName(instanceName);
  }, [instanceStore, instanceName]);

  // Sync tab with URL hash
  useEffect(() => {
    const hash = window.location.hash.replace(/^#?/, "");
    if (isInstanceHash(hash)) {
      setSelectedTab(hash);
    }
  }, []);

  useEffect(() => {
    const query = new URLSearchParams(window.location.search);
    query.delete("qs");
    const url = `${window.location.pathname}?${query.toString()}#${selectedTab}`;
    window.history.replaceState(null, "", url);
  }, [selectedTab]);

  // Set document title
  useEffect(() => {
    if (instance.title) {
      setDocumentTitle(instance.title);
    }
  }, [instance.title]);

  const syncSchema = useCallback(
    async (enableFullSync: boolean) => {
      await instanceStore.syncInstance(instance.name, enableFullSync);
      databaseStore.removeCacheByInstance(instance.name);
    },
    [instance.name, instanceStore, databaseStore]
  );

  // Database filter
  const envVal = getValueFromScopes(searchParams, "environment");
  const selectedEnvironment = envVal
    ? `${environmentNamePrefix}${envVal}`
    : undefined;
  const projectVal = getValueFromScopes(searchParams, "project");
  const selectedProject = projectVal
    ? `${projectNamePrefix}${projectVal}`
    : undefined;
  const selectedLabels = searchParams.scopes
    .filter((s) => s.id === "label")
    .map((s) => s.value);

  const filter: DatabaseFilter = useMemo(
    () => ({
      environment: selectedEnvironment,
      project: selectedProject,
      query: searchParams.query,
      labels: selectedLabels.length > 0 ? selectedLabels : undefined,
    }),
    [selectedEnvironment, selectedProject, searchParams.query, selectedLabels]
  );

  const environmentStore = useEnvironmentV1Store();
  const environments = useVueState(
    () => environmentStore.environmentList ?? []
  );

  const projectStore = useProjectV1Store();
  const searchProjects = useCallback(
    async (keyword: string): Promise<ValueOption[]> => {
      const { projects } = await projectStore.fetchProjectList({
        pageSize: getDefaultPagination(),
        filter: keyword.trim() ? { query: keyword } : undefined,
      });
      return projects.map((p) => {
        const id = extractProjectResourceName(p.name);
        return {
          value: id,
          keywords: [id, p.title],
        };
      });
    },
    [projectStore]
  );

  const scopeOptions: ScopeOption[] = useMemo(
    () => [
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
        id: "project",
        title: t("common.project"),
        description: t("issue.advanced-search.scope.project.description"),
        onSearch: searchProjects,
      },
      {
        id: "label",
        title: t("common.labels"),
        description: t("issue.advanced-search.scope.label.description"),
        allowMultiple: true,
      },
    ],
    [t, environments, searchProjects]
  );

  const handleTabChange = useCallback((tab: string | number | null) => {
    if (typeof tab === "string" && isInstanceHash(tab)) {
      setSelectedTab(tab);
    }
  }, []);

  return (
    <div className="p-4 flex flex-col gap-y-2">
      {/* Archive banner */}
      {instance.state === State.DELETED && (
        <div className="bg-gray-700 text-white text-center py-2 rounded-sm text-sm font-medium">
          {t("common.archived")}
        </div>
      )}

      {/* No environment warning */}
      {!instance.environment && (
        <Alert
          variant="warning"
          className="mb-4"
          description={t("instance.no-environment")}
        />
      )}

      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-x-2">
          <EngineIcon engine={instance.engine} className="h-6 w-6" />
          <span className="text-lg font-medium">
            {instanceV1Name(instance)}
          </span>
        </div>
        <div className="flex items-center gap-x-2">
          {instance.state === State.ACTIVE && (
            <InstanceSyncButton
              instanceName={instance.name}
              instanceTitle={instance.title}
              onSyncSchema={syncSchema}
            />
          )}
          <InstanceActionDropdown instance={instance} />
        </div>
      </div>

      {/* Tabs */}
      <Tabs value={selectedTab} onValueChange={handleTabChange}>
        <TabsList className="border-b-0">
          <TabsTrigger value="overview">{t("common.overview")}</TabsTrigger>
          <TabsTrigger value="databases">{t("common.databases")}</TabsTrigger>
          <TabsTrigger value="users">{t("instance.users")}</TabsTrigger>
        </TabsList>

        <TabsPanel value="overview">
          <InstanceFormProvider instance={instance}>
            <InstanceFormBody />
            <InstanceFormButtons className="sticky bottom-0 z-10" />
            <UnsavedChangesGuard />
          </InstanceFormProvider>
        </TabsPanel>

        <TabsPanel value="databases" keepMounted={false}>
          <div className="flex flex-col gap-y-2">
            <AdvancedSearch
              params={searchParams}
              onParamsChange={setSearchParams}
              placeholder={t("database.filter-database")}
              scopeOptions={scopeOptions}
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
            <DatabaseTable
              filter={filter}
              parent={instance.name}
              mode="ALL"
              selectedNames={selectedNames}
              onSelectedNamesChange={setSelectedNames}
              onDatabasesChange={setVisibleDatabases}
              refreshToken={refreshToken}
            />
            {/* Batch operations bar (sticky at bottom; rendered after the
                table so selection doesn't shift table position) */}
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
                else
                  setSelectedNames(
                    new Set(visibleDatabases.map((d) => d.name))
                  );
              }}
            />
          </div>
        </TabsPanel>

        <TabsPanel value="users">
          <InstanceRoleTable instanceRoleList={instance.roles ?? []} />
        </TabsPanel>
      </Tabs>
    </div>
  );
}

function UnsavedChangesGuard() {
  const { valueChanged } = useInstanceFormContext();
  useUnsavedChangesGuard(valueChanged);
  return null;
}
