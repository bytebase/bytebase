import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { Plus } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { preCreateIssue } from "@/components/Plan/logic/issue";
import {
  AdvancedSearch,
  getValueFromScopes,
  type ScopeOption,
  type SearchParams,
} from "@/react/components/AdvancedSearch";
import {
  CreateDatabaseDrawer,
  DatabaseBatchOperationsBar,
  DatabaseTable,
  LabelEditorDrawer,
} from "@/react/components/database";
import { EditEnvironmentDrawer } from "@/react/components/EditEnvironmentDrawer";
import { Button } from "@/react/components/ui/button";
import { useVueState } from "@/react/hooks/useVueState";
import {
  pushNotification,
  useActuatorV1Store,
  useDatabaseV1Store,
  useDBSchemaV1Store,
  useEnvironmentV1Store,
  useProjectV1Store,
} from "@/store";
import {
  environmentNamePrefix,
  instanceNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import type { DatabaseFilter } from "@/store/modules/v1/database";
import type { Permission } from "@/types";
import {
  isDefaultProject,
  isValidDatabaseName,
  UNKNOWN_ENVIRONMENT_NAME,
  unknownEnvironment,
} from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  BatchUpdateDatabasesRequestSchema,
  DatabaseSchema$,
  UpdateDatabaseRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import {
  engineNameV1,
  hasProjectPermissionV2,
  PERMISSIONS_FOR_DATABASE_CREATE_ISSUE,
  supportedEngineV1List,
} from "@/utils";
import { DataExportPrepDrawer } from "./export-center/DataExportPrepDrawer";

export function ProjectDatabasesPage({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();
  const dbSchemaStore = useDBSchemaV1Store();
  const actuatorStore = useActuatorV1Store();
  const environmentStore = useEnvironmentV1Store();
  const projectStore = useProjectV1Store();

  const projectName = `${projectNamePrefix}${projectId}`;
  const project = useVueState(() => projectStore.getProjectByName(projectName));
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

  const environments = useVueState(
    () => environmentStore.environmentList ?? []
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
        description: t("common.instance"),
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
        description: t("common.labels"),
        allowMultiple: true,
      },
    ];
  }, [t, environments]);

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

  const handleUnassign = useCallback(async () => {
    const defaultProject = actuatorStore.serverInfo?.defaultProject ?? "";
    try {
      await databaseStore.batchUpdateDatabases(
        create(BatchUpdateDatabasesRequestSchema, {
          parent: "-",
          requests: selectedDatabases.map((database) =>
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
  }, [selectedDatabases, databaseStore, actuatorStore, refresh, t]);

  const handleChangeDatabase = useCallback(() => {
    preCreateIssue(
      projectName,
      selectedDatabases.map((db) => db.name)
    );
  }, [projectName, selectedDatabases]);

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
          <Button
            disabled={
              !hasProjectPermission("bb.instances.list") ||
              !PERMISSIONS_FOR_DATABASE_CREATE_ISSUE.every(hasProjectPermission)
            }
            onClick={() => setShowCreateDrawer(true)}
          >
            <Plus className="h-4 w-4 mr-1" />
            {t("quick-action.new-db")}
          </Button>
        </div>
      </div>

      <div className="flex flex-col gap-y-4">
        <DatabaseBatchOperationsBar
          databases={selectedDatabases}
          project={project}
          onSyncSchema={handleSyncSchema}
          onEditLabels={() => setShowLabelEditor(true)}
          onEditEnvironment={() => setShowEditEnvDrawer(true)}
          onUnassign={
            isDefault ? undefined : () => setShowUnassignConfirm(true)
          }
          onChangeDatabase={isDefault ? undefined : handleChangeDatabase}
          onExportData={isDefault ? undefined : handleExportData}
        />

        <EditEnvironmentDrawer
          open={showEditEnvDrawer}
          onClose={() => setShowEditEnvDrawer(false)}
          onUpdate={handleEnvironmentUpdate}
        />
        <LabelEditorDrawer
          open={showLabelEditor}
          databases={selectedDatabases}
          onClose={() => setShowLabelEditor(false)}
          onApply={handleLabelsApply}
        />

        <DatabaseTable
          filter={filter}
          parent={projectName}
          mode="PROJECT"
          selectedNames={selectedNames}
          onSelectedNamesChange={setSelectedNames}
          refreshToken={refreshToken}
        />
      </div>

      <CreateDatabaseDrawer
        open={showCreateDrawer}
        onClose={() => setShowCreateDrawer(false)}
        projectName={projectName}
      />

      <DataExportPrepDrawer
        open={showExportDrawer}
        onClose={() => setShowExportDrawer(false)}
        projectName={projectName}
        seed={{
          selectedDatabaseNames: selectedDatabases.map((db) => db.name),
          step: 2,
        }}
      />

      {/* Unassign confirmation dialog */}
      {showUnassignConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div
            className="fixed inset-0 bg-black/50"
            onClick={() => setShowUnassignConfirm(false)}
          />
          <div className="relative bg-white rounded-sm shadow-lg p-6 max-w-md w-full mx-4">
            <h3 className="text-lg font-semibold mb-2">
              {t("database.unassign-alert-title")}
            </h3>
            <p className="text-sm text-control-light mb-6">
              {t("database.unassign-alert-description")}
            </p>
            <div className="flex justify-end items-center gap-x-2">
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
          </div>
        </div>
      )}
    </div>
  );
}
