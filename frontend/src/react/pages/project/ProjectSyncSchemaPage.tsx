import { create } from "@bufbuild/protobuf";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import {
  ArrowDown,
  ArrowUp,
  ChevronDown,
  ChevronLeft,
  Copy,
  ExternalLink,
  Maximize2,
  Minus,
  Plus,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import { EngineIconPath } from "@/components/InstanceForm/constants";
import {
  createMonacoDiffEditor,
  createMonacoEditor,
} from "@/components/MonacoEditor/editor";
import { DatabaseSelect } from "@/react/components/DatabaseSelect";
import { EnvironmentSelect } from "@/react/components/EnvironmentSelect";
import { Button } from "@/react/components/ui/button";
import { Combobox, type ComboboxOption } from "@/react/components/ui/combobox";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { useClickOutside } from "@/react/hooks/useClickOutside";
import { useEscapeKey } from "@/react/hooks/useEscapeKey";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL } from "@/router/dashboard/projectV1";
import {
  pushNotification,
  useChangelogStore,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useProjectV1Store,
  useStorageStore,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import {
  dialectOfEngineV1,
  getDateForPbTimestampProtoEs,
  isValidDatabaseName,
  isValidEnvironmentName,
} from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  Changelog_Status,
  ChangelogView,
  DiffSchemaRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  databaseV1Url,
  engineNameV1,
  extractChangelogUID,
  extractDatabaseResourceName,
  extractProjectResourceName,
  generatePlanTitle,
  getDatabaseEnvironment,
  getInstanceResource,
  humanizeDate,
} from "@/utils";
import {
  extractDatabaseNameAndChangelogUID,
  isValidChangelogName,
  mockLatestChangelog,
} from "@/utils/v1/changelog";

// ============================================================
// Types
// ============================================================

enum SourceSchemaType {
  SCHEMA_HISTORY_VERSION,
  RAW_SQL,
}

interface ChangelogSourceSchema {
  environmentName?: string;
  databaseName?: string;
  changelogName?: string;
  targetChangelogName?: string;
}

interface RawSQLState {
  engine: Engine;
  statement: string;
}

interface SchemaDiffEntry {
  raw: string;
  edited: string;
}

const ALLOWED_ENGINES: Engine[] = [
  Engine.MYSQL,
  Engine.POSTGRES,
  Engine.TIDB,
  Engine.ORACLE,
  Engine.MSSQL,
  Engine.COCKROACHDB,
];

enum Step {
  SELECT_SOURCE_SCHEMA = 0,
  SELECT_TARGET_DATABASE_LIST = 1,
}

// ============================================================
// Main Page
// ============================================================

export function ProjectSyncSchemaPage({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const changelogStore = useChangelogStore();
  const databaseStore = useDatabaseV1Store();
  const projectStore = useProjectV1Store();

  const projectName = `${projectNamePrefix}${projectId}`;
  const project = useVueState(() => projectStore.getProjectByName(projectName));

  const [isLoading, setIsLoading] = useState(true);
  const [currentStep, setCurrentStep] = useState<Step>(
    Step.SELECT_SOURCE_SCHEMA
  );
  const [sourceSchemaType, setSourceSchemaType] = useState<SourceSchemaType>(
    SourceSchemaType.SCHEMA_HISTORY_VERSION
  );
  const [changelogSource, setChangelogSource] = useState<ChangelogSourceSchema>(
    {}
  );
  const [rawSQLState, setRawSQLState] = useState<RawSQLState>({
    engine: Engine.MYSQL,
    statement: "",
  });

  // Reset all state when projectId changes (e.g. navigating between projects)
  useEffect(() => {
    setIsLoading(true);
    setCurrentStep(Step.SELECT_SOURCE_SCHEMA);
    setSourceSchemaType(SourceSchemaType.SCHEMA_HISTORY_VERSION);
    setChangelogSource({});
    setRawSQLState({ engine: Engine.MYSQL, statement: "" });
    setSelectedDatabaseNameList([]);
    setSchemaDiffCache({});
    setSelectedDatabaseName(undefined);
    setSourceSchemaString("");
  }, [projectId]);

  // Route change guard: warn when leaving with unsaved raw SQL
  useEffect(() => {
    const shouldGuard =
      sourceSchemaType === SourceSchemaType.RAW_SQL &&
      rawSQLState.statement !== "";
    if (!shouldGuard) return;

    // Guard browser/tab unload
    const beforeUnload = (e: BeforeUnloadEvent) => {
      e.preventDefault();
      e.returnValue = "";
    };
    window.addEventListener("beforeunload", beforeUnload);

    // Guard in-app Vue router navigation
    const removeRouterGuard = router.beforeEach((_to, _from, next) => {
      const answer = window.confirm(t("common.leave-without-saving"));
      if (answer) {
        next();
      } else {
        next(false);
      }
    });

    return () => {
      window.removeEventListener("beforeunload", beforeUnload);
      removeRouterGuard();
    };
  }, [sourceSchemaType, rawSQLState.statement, t]);

  // Target databases state (lifted up so we can access from step controls)
  const [selectedDatabaseNameList, setSelectedDatabaseNameList] = useState<
    string[]
  >([]);
  const [schemaDiffCache, setSchemaDiffCache] = useState<
    Record<string, SchemaDiffEntry>
  >({});
  const [selectedDatabaseName, setSelectedDatabaseName] = useState<
    string | undefined
  >();

  // Computed source schema string
  const [sourceSchemaString, setSourceSchemaString] = useState("");

  // Recompute source schema when source changes
  useEffect(() => {
    let cancelled = false;
    (async () => {
      if (sourceSchemaType === SourceSchemaType.SCHEMA_HISTORY_VERSION) {
        if (isValidChangelogName(changelogSource.changelogName)) {
          const changelog = changelogStore.getChangelogByName(
            changelogSource.changelogName || ""
          );
          if (changelog) {
            if (!cancelled) setSourceSchemaString(changelog.schema);
            return;
          }
          if (!cancelled) setSourceSchemaString("");
          return;
        } else if (isValidDatabaseName(changelogSource.databaseName)) {
          const databaseSchema = await databaseStore.fetchDatabaseSchema(
            changelogSource.databaseName
          );
          if (!cancelled) setSourceSchemaString(databaseSchema.schema);
          return;
        }
        if (!cancelled) setSourceSchemaString("");
      } else {
        if (!cancelled) setSourceSchemaString(rawSQLState.statement);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [
    sourceSchemaType,
    changelogSource.changelogName,
    changelogSource.databaseName,
    rawSQLState.statement,
  ]);

  const sourceEngine = useMemo(() => {
    if (sourceSchemaType === SourceSchemaType.SCHEMA_HISTORY_VERSION) {
      if (!changelogSource.databaseName) return Engine.ENGINE_UNSPECIFIED;
      const database = databaseStore.getDatabaseByName(
        changelogSource.databaseName
      );
      return getInstanceResource(database).engine;
    }
    return rawSQLState.engine;
  }, [sourceSchemaType, changelogSource.databaseName, rawSQLState.engine]);

  // On mount / projectId change: check query params for changelog
  useEffect(() => {
    let cancelled = false;

    (async () => {
      const currentRoute = router.currentRoute.value;
      const changelogName = currentRoute.query.changelog as string;
      const isRollback = currentRoute.query.rollback === "true";

      if (isValidChangelogName(changelogName)) {
        await changelogStore.getOrFetchChangelogByName(
          changelogName,
          ChangelogView.FULL
        );
        if (cancelled) return;

        const sourceChangelogName = changelogName;
        let targetChangelogName: string | undefined = undefined;

        if (isRollback) {
          const previousChangelog =
            await changelogStore.fetchPreviousChangelog(changelogName);
          if (cancelled) return;
          if (previousChangelog) {
            targetChangelogName = previousChangelog.name;
          }
        }

        const { databaseName } =
          extractDatabaseNameAndChangelogUID(changelogName);
        await databaseStore.getOrFetchDatabaseByName(databaseName);
        if (cancelled) return;
        const database = databaseStore.getDatabaseByName(databaseName);
        setChangelogSource({
          environmentName: database.effectiveEnvironment,
          databaseName: databaseName,
          changelogName: sourceChangelogName,
          targetChangelogName: targetChangelogName,
        });
        setTimeout(() => {
          if (!cancelled) setCurrentStep(Step.SELECT_TARGET_DATABASE_LIST);
        }, 0);
      }
      if (!cancelled) setIsLoading(false);
    })();

    return () => {
      cancelled = true;
    };
  }, [projectId]);

  const stepList = useMemo(
    () => [
      { title: t("database.sync-schema.select-source-schema") },
      { title: t("database.sync-schema.select-target-databases") },
    ],
    [t]
  );

  const allowNext = useMemo(() => {
    if (currentStep === Step.SELECT_SOURCE_SCHEMA) {
      if (sourceSchemaType === SourceSchemaType.SCHEMA_HISTORY_VERSION) {
        return (
          isValidEnvironmentName(changelogSource.environmentName) &&
          isValidDatabaseName(changelogSource.databaseName) &&
          !!changelogSource.changelogName
        );
      }
      return rawSQLState.statement !== "";
    }
    // Step 2: at least one target database with diff
    const targetDatabaseDiffList = selectedDatabaseNameList
      .map((name) => ({
        name,
        diff: schemaDiffCache[name]?.edited || "",
      }))
      .filter((item) => item.diff !== "");
    return targetDatabaseDiffList.length > 0;
  }, [
    currentStep,
    sourceSchemaType,
    changelogSource,
    rawSQLState.statement,
    selectedDatabaseNameList,
    schemaDiffCache,
  ]);

  const [showConfirmDialog, setShowConfirmDialog] = useState(false);

  const handleStepChange = useCallback(
    async (nextStep: number) => {
      if (
        currentStep === Step.SELECT_TARGET_DATABASE_LIST &&
        nextStep === Step.SELECT_SOURCE_SCHEMA
      ) {
        if (selectedDatabaseNameList.length > 0) {
          setShowConfirmDialog(true);
          return;
        }
        // Even with no databases selected, clear stale diff cache
        setSchemaDiffCache({});
        setSelectedDatabaseName(undefined);
      } else if (
        currentStep === Step.SELECT_SOURCE_SCHEMA &&
        nextStep === Step.SELECT_TARGET_DATABASE_LIST
      ) {
        if (changelogSource.changelogName) {
          await changelogStore.getOrFetchChangelogByName(
            changelogSource.changelogName,
            ChangelogView.FULL
          );
        }
      }
      setCurrentStep(nextStep as Step);
    },
    [currentStep, selectedDatabaseNameList, changelogSource.changelogName]
  );

  const handleConfirmRevert = useCallback(() => {
    setShowConfirmDialog(false);
    setSelectedDatabaseNameList([]);
    setSchemaDiffCache({});
    setSelectedDatabaseName(undefined);
    setCurrentStep(Step.SELECT_SOURCE_SCHEMA);
  }, []);

  const handleFinish = useCallback(async () => {
    const targetDatabases = selectedDatabaseNameList.map((name) =>
      databaseStore.getDatabaseByName(name)
    );
    const query: Record<string, string> = {
      template: "bb.plan.change-database",
      mode: "normal",
    };
    const sqlMap: Record<string, string> = {};
    targetDatabases.forEach((db) => {
      const diff = schemaDiffCache[db.name];
      if (diff?.edited) {
        sqlMap[db.name] = diff.edited;
      }
    });
    query.databaseList = Object.keys(sqlMap).join(",");
    const sqlMapStorageKey = `bb.issues.sql-map.${uuidv4()}`;
    useStorageStore().put(sqlMapStorageKey, sqlMap);
    query.sqlMapStorageKey = sqlMapStorageKey;
    query.name = generatePlanTitle(
      "bb.plan.change-database",
      targetDatabases.map(
        (db) => extractDatabaseResourceName(db.name).databaseName
      )
    );

    router.push({
      name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
      params: {
        projectId: extractProjectResourceName(project.name),
        planId: "create",
        specId: "placeholder",
      },
      query,
    });
  }, [selectedDatabaseNameList, schemaDiffCache, project]);

  if (!project) return null;

  return (
    <div className="w-full h-full overflow-hidden flex flex-col px-4 py-4">
      <p className="text-sm text-gray-500">
        {t("database.sync-schema.description")}{" "}
        <a
          className="inline-flex items-center normal-link"
          href="https://docs.bytebase.com/change-database/synchronize-schema?source=console"
          target="__BLANK"
        >
          {t("common.learn-more")}
          <ExternalLink className="w-4 h-4 ml-1" />
        </a>
      </p>

      {isLoading ? (
        <div className="flex items-center justify-center py-2 text-gray-400 text-sm">
          <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-gray-400" />
        </div>
      ) : (
        <div className="pt-4 flex-1 overflow-hidden flex flex-col gap-y-4">
          {/* Step indicator */}
          <StepIndicator steps={stepList} currentIndex={currentStep} />

          {/* Step content */}
          <div className="flex-1 overflow-y-auto">
            {currentStep === Step.SELECT_SOURCE_SCHEMA && (
              <SourceSchemaStep
                project={project}
                sourceSchemaType={sourceSchemaType}
                onSourceSchemaTypeChange={setSourceSchemaType}
                changelogSource={changelogSource}
                onChangelogSourceChange={setChangelogSource}
                rawSQLState={rawSQLState}
                onRawSQLStateChange={setRawSQLState}
              />
            )}
            {currentStep === Step.SELECT_TARGET_DATABASE_LIST && (
              <SelectTargetDatabasesView
                project={project}
                sourceSchemaString={sourceSchemaString}
                sourceEngine={sourceEngine}
                changelogSourceSchema={
                  sourceSchemaType === SourceSchemaType.SCHEMA_HISTORY_VERSION
                    ? changelogSource
                    : undefined
                }
                selectedDatabaseNameList={selectedDatabaseNameList}
                onSelectedDatabaseNameListChange={setSelectedDatabaseNameList}
                schemaDiffCache={schemaDiffCache}
                onSchemaDiffCacheChange={setSchemaDiffCache}
                selectedDatabaseName={selectedDatabaseName}
                onSelectedDatabaseNameChange={setSelectedDatabaseName}
              />
            )}
          </div>

          {/* Footer */}
          <div className="pt-4 border-t border-block-border flex items-center gap-x-2 justify-between">
            <div />
            <div className="flex items-center justify-between gap-x-2">
              {currentStep !== 0 && (
                <Button
                  variant="outline"
                  onClick={() => handleStepChange(currentStep - 1)}
                >
                  <ChevronLeft className="-ml-1 mr-1 h-5 w-5 text-control-light" />
                  <span>{t("common.back")}</span>
                </Button>
              )}
              {currentStep === stepList.length - 1 ? (
                <Button disabled={!allowNext} onClick={handleFinish}>
                  {t("database.sync-schema.preview-issue")}
                </Button>
              ) : (
                <Button
                  disabled={!allowNext}
                  onClick={() => handleStepChange(currentStep + 1)}
                >
                  {t("common.next")}
                </Button>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Confirm revert dialog */}
      <Dialog open={showConfirmDialog} onOpenChange={setShowConfirmDialog}>
        <DialogContent className="p-6">
          <DialogTitle>{t("common.confirm-to-revert")}</DialogTitle>
          <div className="flex justify-end gap-x-2 mt-4">
            <Button
              variant="outline"
              onClick={() => setShowConfirmDialog(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button onClick={handleConfirmRevert}>{t("common.confirm")}</Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}

// ============================================================
// StepIndicator
// ============================================================

function StepIndicator({
  steps,
  currentIndex,
}: {
  steps: { title: string }[];
  currentIndex: number;
}) {
  return (
    <div className="flex items-center gap-x-2 px-0.5">
      {steps.map((step, i) => (
        <div key={i} className="flex items-center gap-x-2">
          {i > 0 && <div className="w-8 h-px bg-gray-300" />}
          <div className="flex items-center gap-x-2">
            <div
              className={cn(
                "w-6 h-6 rounded-full flex items-center justify-center text-xs font-medium",
                i <= currentIndex
                  ? "bg-accent text-white"
                  : "bg-gray-200 text-gray-500"
              )}
            >
              {i + 1}
            </div>
            <span
              className={cn(
                "text-sm",
                i <= currentIndex ? "text-accent font-medium" : "text-gray-500"
              )}
            >
              {step.title}
            </span>
          </div>
        </div>
      ))}
    </div>
  );
}

// ============================================================
// SourceSchemaStep
// ============================================================

function SourceSchemaStep({
  project,
  sourceSchemaType,
  onSourceSchemaTypeChange,
  changelogSource,
  onChangelogSourceChange,
  rawSQLState,
  onRawSQLStateChange,
}: {
  project: Project;
  sourceSchemaType: SourceSchemaType;
  onSourceSchemaTypeChange: (type: SourceSchemaType) => void;
  changelogSource: ChangelogSourceSchema;
  onChangelogSourceChange: (source: ChangelogSourceSchema) => void;
  rawSQLState: RawSQLState;
  onRawSQLStateChange: (state: RawSQLState) => void;
}) {
  const { t } = useTranslation();

  return (
    <>
      <div className="mb-4">
        <div className="flex gap-x-4">
          <label className="flex items-center gap-x-2 cursor-pointer">
            <input
              type="radio"
              name="source-schema-type"
              checked={
                sourceSchemaType === SourceSchemaType.SCHEMA_HISTORY_VERSION
              }
              onChange={() =>
                onSourceSchemaTypeChange(
                  SourceSchemaType.SCHEMA_HISTORY_VERSION
                )
              }
              className="accent-accent"
            />
            <span className="text-sm">{t("common.changelog")}</span>
          </label>
          <label className="flex items-center gap-x-2 cursor-pointer">
            <input
              type="radio"
              name="source-schema-type"
              checked={sourceSchemaType === SourceSchemaType.RAW_SQL}
              onChange={() =>
                onSourceSchemaTypeChange(SourceSchemaType.RAW_SQL)
              }
              className="accent-accent"
            />
            <span className="text-sm">
              {t("database.sync-schema.copy-schema")}
            </span>
          </label>
        </div>
      </div>
      {sourceSchemaType === SourceSchemaType.SCHEMA_HISTORY_VERSION && (
        <DatabaseSchemaSelector
          project={project}
          sourceSchema={changelogSource}
          onUpdate={onChangelogSourceChange}
        />
      )}
      {sourceSchemaType === SourceSchemaType.RAW_SQL && (
        <RawSQLEditor
          engine={rawSQLState.engine}
          statement={rawSQLState.statement}
          onUpdate={onRawSQLStateChange}
        />
      )}
    </>
  );
}

// ============================================================
// DatabaseSchemaSelector
// ============================================================

function DatabaseSchemaSelector({
  project,
  sourceSchema,
  onUpdate,
}: {
  project: Project;
  sourceSchema?: ChangelogSourceSchema;
  onUpdate: (source: ChangelogSourceSchema) => void;
}) {
  const { t } = useTranslation();

  // Fully controlled — derive values from props, call onUpdate directly.
  const environmentName = sourceSchema?.environmentName || "";
  const databaseName = sourceSchema?.databaseName || "";
  const changelogName = sourceSchema?.changelogName || "";

  const handleEnvironmentChange = useCallback(
    (name: string) => {
      if (name !== environmentName) {
        onUpdate({
          environmentName: name,
          databaseName: "",
          changelogName: "",
        });
      } else {
        onUpdate({ ...sourceSchema, environmentName: name });
      }
    },
    [environmentName, sourceSchema, onUpdate]
  );

  const handleDatabaseChange = useCallback(
    (name: string, database: Database | undefined) => {
      if (isValidDatabaseName(name) && database) {
        onUpdate({
          environmentName: database.effectiveEnvironment ?? "",
          databaseName: name,
          changelogName: "",
        });
        return;
      }
      onUpdate({ ...sourceSchema, databaseName: "", changelogName: "" });
    },
    [sourceSchema, onUpdate]
  );

  const handleChangelogChange = useCallback(
    (name: string) => {
      onUpdate({ ...sourceSchema, changelogName: name });
    },
    [sourceSchema, onUpdate]
  );

  return (
    <div className="w-full mx-auto flex flex-col justify-start items-start gap-y-3 mb-6">
      <div className="w-full flex flex-col gap-y-2 lg:flex-row justify-start items-start lg:items-center">
        <span className="flex w-40 items-center shrink-0 text-sm">
          {t("common.database")}
        </span>
        <EnvironmentSelect
          className="w-52 shrink-0 mr-3"
          value={environmentName}
          onChange={handleEnvironmentChange}
        />
        <DatabaseSelect
          className="flex-1 min-w-0"
          value={databaseName}
          onChange={handleDatabaseChange}
          projectName={project.name}
          environmentName={environmentName}
          allowedEngineTypeList={ALLOWED_ENGINES}
        />
      </div>
      <div className="w-full flex flex-col gap-y-2">
        <div className="text-sm">
          {t("common.changelog")}
          <div className="textinfolabel">{t("changelog.select")}</div>
        </div>
        <div className="w-full flex flex-row justify-start items-center relative">
          <ChangelogSelector
            database={databaseName}
            value={changelogName}
            onChange={handleChangelogChange}
          />
        </div>
      </div>
    </div>
  );
}

// ============================================================
// ChangelogSelector
// ============================================================

interface ChangelogEntry {
  name: string;
  date: Date | undefined;
  planTitle: string;
}

function ChangelogSelector({
  database,
  value,
  onChange,
}: {
  database?: string;
  value?: string;
  onChange: (value: string) => void;
}) {
  const { t } = useTranslation();
  const changelogStore = useChangelogStore();
  const databaseStore = useDatabaseV1Store();
  const [entries, setEntries] = useState<ChangelogEntry[]>([]);
  const [nextPageToken, setNextPageToken] = useState("");
  const [loadingMore, setLoadingMore] = useState(false);
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const disabled = !isValidDatabaseName(database);

  useClickOutside(containerRef, open, () => setOpen(false));
  useEscapeKey(open, () => setOpen(false));

  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  const valueRef = useRef(value);
  valueRef.current = value;

  const toEntry = useCallback(
    (changelog: {
      name: string;
      createTime?: Timestamp;
      planTitle: string;
    }): ChangelogEntry => ({
      name: changelog.name,
      date: getDateForPbTimestampProtoEs(changelog.createTime)
        ? new Date(getDateForPbTimestampProtoEs(changelog.createTime)!)
        : undefined,
      planTitle: changelog.planTitle,
    }),
    []
  );

  useEffect(() => {
    if (!isValidDatabaseName(database)) {
      setEntries([]);
      setNextPageToken("");
      return;
    }

    let cancelled = false;

    (async () => {
      const { changelogs: fetchedChangelogs, nextPageToken: token } =
        await changelogStore.fetchChangelogList({
          parent: database,
          pageSize: 50,
          filter: `status == "${Changelog_Status[Changelog_Status.DONE]}"`,
        });

      if (cancelled) return;

      let items = fetchedChangelogs.map(toEntry);

      if (items.length === 0) {
        const db = databaseStore.getDatabaseByName(database);
        const changelog = mockLatestChangelog(db);
        items = [{ name: changelog.name, date: undefined, planTitle: "" }];
      }

      setEntries(items);
      setNextPageToken(token);

      // Auto-select first when no value is set
      if (items.length > 0 && !valueRef.current) {
        onChangeRef.current(items[0].name);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [database]);

  const loadMore = useCallback(async () => {
    if (!nextPageToken || !isValidDatabaseName(database) || loadingMore) return;
    setLoadingMore(true);
    const { changelogs: more, nextPageToken: token } =
      await changelogStore.fetchChangelogList({
        parent: database,
        pageToken: nextPageToken,
        pageSize: 50,
        filter: `status == "${Changelog_Status[Changelog_Status.DONE]}"`,
      });
    setEntries((prev) => [...prev, ...more.map(toEntry)]);
    setNextPageToken(token);
    setLoadingMore(false);
  }, [nextPageToken, database, loadingMore, toEntry]);

  const selectedEntry = entries.find((e) => e.name === value);

  return (
    <div ref={containerRef} className="relative w-full">
      <button
        type="button"
        disabled={disabled}
        className={cn(
          "w-full flex items-center justify-between gap-2 border border-gray-300 rounded-xs h-9 px-3 text-sm bg-white text-left transition-colors",
          "hover:border-gray-400",
          "disabled:opacity-50 disabled:pointer-events-none",
          open && "border-accent shadow-[0_0_0_1px_var(--color-accent)]"
        )}
        onClick={() => !disabled && setOpen(!open)}
      >
        <span className={cn("truncate", !selectedEntry && "text-gray-400")}>
          {selectedEntry ? (
            <ChangelogLabel entry={selectedEntry} />
          ) : (
            t("changelog.self")
          )}
        </span>
        <ChevronDown
          className={cn(
            "w-4 h-4 text-gray-400 shrink-0 transition-transform",
            open && "rotate-180"
          )}
        />
      </button>
      {open && (
        <div className="absolute z-50 mt-1 min-w-full w-max bg-white border border-gray-200 rounded-sm shadow-lg overflow-hidden">
          <div className="max-h-60 overflow-y-auto">
            {entries.map((entry) => (
              <button
                key={entry.name}
                type="button"
                className={cn(
                  "w-full text-left px-3 py-2 text-sm flex items-center gap-2 transition-colors",
                  "hover:bg-gray-50",
                  entry.name === value && "bg-accent/5"
                )}
                onClick={() => {
                  onChange(entry.name);
                  setOpen(false);
                }}
              >
                <ChangelogLabel entry={entry} />
              </button>
            ))}
            {nextPageToken && (
              <button
                type="button"
                className="w-full text-center px-3 py-2 text-sm text-accent hover:bg-gray-50 transition-colors"
                onClick={(e) => {
                  e.stopPropagation();
                  loadMore();
                }}
                disabled={loadingMore}
              >
                {loadingMore ? t("common.loading") : t("common.load-more")}
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

function ChangelogLabel({ entry }: { entry: ChangelogEntry }) {
  return (
    <span className="flex items-center gap-1.5 truncate">
      <span className="text-control-light">
        {entry.date ? humanizeDate(entry.date) : "Latest version"}
      </span>
      {entry.planTitle && (
        <span className="inline-flex items-center px-1.5 py-0 rounded-full bg-gray-100 text-xs">
          {entry.planTitle}
        </span>
      )}
    </span>
  );
}

// ============================================================
// RawSQLEditor
// ============================================================

function RawSQLEditor({
  engine,
  statement,
  onUpdate,
}: {
  engine: Engine;
  statement?: string;
  onUpdate: (state: RawSQLState) => void;
}) {
  const { t } = useTranslation();
  const [localEngine, setLocalEngine] = useState<Engine>(
    engine || Engine.MYSQL
  );
  const [localStatement, setLocalStatement] = useState(statement || "");
  const containerRef = useRef<HTMLDivElement>(null);
  // biome-ignore lint/suspicious/noExplicitAny: Monaco editor instance type
  const editorRef = useRef<any>(null); // eslint-disable-line @typescript-eslint/no-explicit-any
  const onUpdateRef = useRef(onUpdate);
  onUpdateRef.current = onUpdate;
  const localEngineRef = useRef(localEngine);
  localEngineRef.current = localEngine;

  const engineOptions: ComboboxOption[] = useMemo(
    () =>
      ALLOWED_ENGINES.map((e) => ({
        value: String(e),
        label: engineNameV1(e),
      })),
    []
  );

  useEffect(() => {
    let disposed = false;
    (async () => {
      if (!containerRef.current) return;
      const editor = await createMonacoEditor({
        container: containerRef.current,
        options: {
          language: dialectOfEngineV1(localEngine),
          value: localStatement,
        },
      });
      if (disposed) {
        editor.dispose();
        return;
      }
      editorRef.current = editor;
      editor.onDidChangeModelContent(() => {
        const val = editor.getValue();
        setLocalStatement(val);
        onUpdateRef.current({ engine: localEngineRef.current, statement: val });
      });
      editor.focus();
    })();
    return () => {
      disposed = true;
      editorRef.current?.dispose();
      editorRef.current = null;
    };
  }, []);

  // Update language when engine changes
  useEffect(() => {
    const model = editorRef.current?.getModel();
    if (model) {
      (async () => {
        const { editor: monacoEditor } = await import("monaco-editor");
        monacoEditor.setModelLanguage(model, dialectOfEngineV1(localEngine));
      })();
    }
  }, [localEngine]);

  const handleEngineChange = useCallback(
    (val: string) => {
      const eng = Number(val) as Engine;
      setLocalEngine(eng);
      onUpdateRef.current({ engine: eng, statement: localStatement });
    },
    [localStatement]
  );

  const handleFileUpload = useCallback(() => {
    const input = document.createElement("input");
    input.type = "file";
    input.accept = ".sql,.txt";
    input.onchange = async () => {
      const file = input.files?.[0];
      if (!file) return;
      const text = await file.text();
      setLocalStatement(text);
      editorRef.current?.setValue(text);
      onUpdateRef.current({ engine: localEngine, statement: text });
    };
    input.click();
  }, [localEngine]);

  return (
    <div className="w-full h-auto flex flex-col justify-start items-start">
      <div className="w-full h-auto shrink-0 flex flex-row justify-between items-end">
        <div className="flex flex-col justify-start items-start gap-y-2">
          <div className="flex flex-row justify-start items-center">
            <span className="mr-2 shrink-0 text-sm">
              {t("database.engine")}
            </span>
            <Combobox
              className="w-48"
              value={String(localEngine)}
              options={engineOptions}
              placeholder={t("database.engine")}
              onChange={handleEngineChange}
            />
          </div>
        </div>
        <div className="flex flex-row justify-end items-center gap-x-3">
          <Button variant="outline" size="sm" onClick={handleFileUpload}>
            {t("issue.upload-sql")}
          </Button>
        </div>
      </div>
      <div className="mt-4 w-full h-96 overflow-hidden">
        <div ref={containerRef} className="w-full h-full border" />
      </div>
    </div>
  );
}

// ============================================================
// SourceSchemaInfo
// ============================================================

function SourceSchemaInfo({
  sourceEngine,
  changelogSourceSchema,
}: {
  sourceEngine: Engine;
  changelogSourceSchema?: ChangelogSourceSchema;
}) {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();

  const databaseFromChangelog = useMemo(() => {
    return databaseStore.getDatabaseByName(
      changelogSourceSchema?.databaseName || ""
    );
  }, [changelogSourceSchema?.databaseName]);

  const changelogUID = useMemo(() => {
    const name = changelogSourceSchema?.changelogName || "";
    if (!isValidChangelogName(name)) return undefined;
    return extractChangelogUID(name);
  }, [changelogSourceSchema?.changelogName]);

  const gotoDatabase = useCallback(() => {
    if (isValidDatabaseName(databaseFromChangelog.name)) {
      window.open(databaseV1Url(databaseFromChangelog));
    }
  }, [databaseFromChangelog]);

  const gotoChangelog = useCallback(() => {
    if (isValidChangelogName(changelogSourceSchema?.changelogName || "")) {
      window.open(
        `${databaseV1Url(databaseFromChangelog)}/changelogs/${changelogUID}`
      );
    }
  }, [databaseFromChangelog, changelogUID, changelogSourceSchema]);

  return (
    <div className="w-full flex flex-row justify-start items-center flex-wrap gap-2 text-sm">
      <span>{t("database.sync-schema.source-schema")}</span>
      {changelogSourceSchema ? (
        <>
          <button
            className="inline-flex items-center gap-x-1 px-2.5 py-0.5 rounded-full bg-gray-100 hover:bg-gray-200 text-sm transition-colors"
            onClick={gotoDatabase}
          >
            <span className="opacity-60">{t("common.database")}</span>
            {EngineIconPath[sourceEngine] && (
              <img
                src={EngineIconPath[sourceEngine]}
                className="w-4 h-auto"
                alt=""
              />
            )}
            <span>
              {
                extractDatabaseResourceName(databaseFromChangelog.name)
                  .databaseName
              }
            </span>
          </button>
          <button
            className="inline-flex items-center gap-x-1 px-2.5 py-0.5 rounded-full bg-gray-100 hover:bg-gray-200 text-sm transition-colors"
            onClick={gotoChangelog}
          >
            <span className="opacity-60 mr-1">{t("common.changelog")}</span>
            <span>{changelogUID ? `#${changelogUID}` : "Latest"}</span>
          </button>
        </>
      ) : (
        <>
          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full bg-gray-100 text-sm">
            {t("schema-editor.raw-sql")}
          </span>
          <span className="inline-flex items-center gap-x-1 px-2.5 py-0.5 rounded-full bg-gray-100 text-sm">
            <span className="opacity-60 mr-1">{t("database.engine")}</span>
            {EngineIconPath[sourceEngine] && (
              <img
                src={EngineIconPath[sourceEngine]}
                className="w-4 h-auto"
                alt=""
              />
            )}
            <span>{engineNameV1(sourceEngine)}</span>
          </span>
        </>
      )}
    </div>
  );
}

// ============================================================
// SelectTargetDatabasesView
// ============================================================

function SelectTargetDatabasesView({
  project,
  sourceSchemaString,
  sourceEngine,
  changelogSourceSchema,
  selectedDatabaseNameList,
  onSelectedDatabaseNameListChange,
  schemaDiffCache,
  onSchemaDiffCacheChange,
  selectedDatabaseName,
  onSelectedDatabaseNameChange,
}: {
  project: Project;
  sourceSchemaString: string;
  sourceEngine: Engine;
  changelogSourceSchema?: ChangelogSourceSchema;
  selectedDatabaseNameList: string[];
  onSelectedDatabaseNameListChange: (list: string[]) => void;
  schemaDiffCache: Record<string, SchemaDiffEntry>;
  onSchemaDiffCacheChange: (cache: Record<string, SchemaDiffEntry>) => void;
  selectedDatabaseName?: string;
  onSelectedDatabaseNameChange: (name: string | undefined) => void;
}) {
  const { t } = useTranslation();
  const changelogStore = useChangelogStore();
  const environmentStore = useEnvironmentV1Store();
  const databaseStore = useDatabaseV1Store();
  const [isLoadingDiff, setIsLoadingDiff] = useState(false);

  // Refs for values read inside the diff-fetching effect without triggering re-runs
  const sourceSchemaStringRef = useRef(sourceSchemaString);
  sourceSchemaStringRef.current = sourceSchemaString;
  const changelogSourceSchemaRef = useRef(changelogSourceSchema);
  changelogSourceSchemaRef.current = changelogSourceSchema;
  const [showDatabaseWithDiff, setShowDatabaseWithDiff] = useState(true);
  const [showSelectPanel, setShowSelectPanel] = useState(false);
  const [databaseSchemaCache, setDatabaseSchemaCache] = useState<
    Record<string, string>
  >({});

  const targetDatabaseList = useMemo(
    () =>
      selectedDatabaseNameList.map((name) =>
        databaseStore.getDatabaseByName(name)
      ),
    [selectedDatabaseNameList]
  );

  const sourceSchemaDisplayString = useMemo(() => {
    const isRollback = isValidChangelogName(
      changelogSourceSchema?.targetChangelogName
    );
    if (isRollback && selectedDatabaseName) {
      return databaseSchemaCache[selectedDatabaseName] || "";
    }
    return sourceSchemaString;
  }, [
    changelogSourceSchema,
    selectedDatabaseName,
    databaseSchemaCache,
    sourceSchemaString,
  ]);

  const targetSchemaDisplayString = useMemo(() => {
    const isRollback = isValidChangelogName(
      changelogSourceSchema?.targetChangelogName
    );
    if (isRollback) {
      return sourceSchemaString;
    }
    return selectedDatabaseName
      ? databaseSchemaCache[selectedDatabaseName] || ""
      : "";
  }, [
    changelogSourceSchema,
    sourceSchemaString,
    selectedDatabaseName,
    databaseSchemaCache,
  ]);

  const shouldShowDiff = useMemo(() => {
    return !!(
      selectedDatabaseName && schemaDiffCache[selectedDatabaseName]?.raw !== ""
    );
  }, [selectedDatabaseName, schemaDiffCache]);

  const previewSchemaChangeMessage = useMemo(() => {
    if (!selectedDatabaseName) return "";
    const database = targetDatabaseList.find(
      (db) => db.name === selectedDatabaseName
    );
    if (!database) return "";
    const environment = environmentStore.getEnvironmentByName(
      database.effectiveEnvironment ?? ""
    );
    return t("database.sync-schema.schema-change-preview", {
      database: `${extractDatabaseResourceName(database.name).databaseName} (${environment?.title} - ${getInstanceResource(database).title})`,
    });
  }, [selectedDatabaseName, targetDatabaseList, t]);

  const databaseListWithDiff = useMemo(
    () =>
      targetDatabaseList.filter((db) => schemaDiffCache[db.name]?.raw !== ""),
    [targetDatabaseList, schemaDiffCache]
  );
  const databaseListWithoutDiff = useMemo(
    () =>
      targetDatabaseList.filter((db) => schemaDiffCache[db.name]?.raw === ""),
    [targetDatabaseList, schemaDiffCache]
  );
  const shownDatabaseList = showDatabaseWithDiff
    ? databaseListWithDiff
    : databaseListWithoutDiff;

  // Fetch schemas and diffs when database list changes
  useEffect(() => {
    let cancelled = false;
    const loadTimeout = setTimeout(() => {
      if (!cancelled) setIsLoadingDiff(true);
    }, 300);

    (async () => {
      const newSchemaCache = { ...databaseSchemaCache };
      const newDiffCache = { ...schemaDiffCache };

      try {
        const clsRef = changelogSourceSchemaRef.current;
        const srcRef = sourceSchemaStringRef.current;

        for (const name of selectedDatabaseNameList) {
          if (cancelled) break;

          if (!newSchemaCache[name]) {
            const db = databaseStore.getDatabaseByName(name);
            const isRollback = isValidChangelogName(
              clsRef?.targetChangelogName
            );
            if (isRollback) {
              const previousChangelog =
                await changelogStore.getOrFetchChangelogByName(
                  clsRef?.targetChangelogName ?? "",
                  ChangelogView.FULL
                );
              newSchemaCache[name] = previousChangelog?.schema ?? "";
            } else {
              const schema = await databaseStore.fetchDatabaseSchema(db.name);
              newSchemaCache[name] = schema.schema;
            }
          }

          if (!newDiffCache[name]) {
            const db = databaseStore.getDatabaseByName(name);
            const isRollback = isValidChangelogName(
              clsRef?.targetChangelogName
            );
            const diffRequest = isRollback
              ? create(DiffSchemaRequestSchema, {
                  name: clsRef?.changelogName ?? "",
                  target: {
                    case: "changelog",
                    value: clsRef?.targetChangelogName ?? "",
                  },
                })
              : isValidChangelogName(clsRef?.changelogName)
                ? create(DiffSchemaRequestSchema, {
                    name: db.name,
                    target: {
                      case: "changelog",
                      value: clsRef?.changelogName ?? "",
                    },
                  })
                : create(DiffSchemaRequestSchema, {
                    name: db.name,
                    target: {
                      case: "schema",
                      value: srcRef,
                    },
                  });

            const diffResp = await databaseStore.diffSchema(diffRequest);
            const schemaDiff = diffResp.diff ?? "";
            newDiffCache[name] = { raw: schemaDiff, edited: schemaDiff };
          }
        }
      } catch {
        // Error is already surfaced via the gRPC error notification.
        // Ensure we still update caches with whatever we loaded so far.
      }

      if (!cancelled) {
        setDatabaseSchemaCache(newSchemaCache);
        onSchemaDiffCacheChange(newDiffCache);
        clearTimeout(loadTimeout);
        setIsLoadingDiff(false);

        // Auto select first with diff
        if (
          selectedDatabaseName &&
          !selectedDatabaseNameList.includes(selectedDatabaseName)
        ) {
          onSelectedDatabaseNameChange(undefined);
        }
        if (
          !selectedDatabaseName ||
          !selectedDatabaseNameList.includes(selectedDatabaseName)
        ) {
          const firstWithDiff = selectedDatabaseNameList.find(
            (name) => newDiffCache[name]?.raw !== ""
          );
          if (firstWithDiff) {
            onSelectedDatabaseNameChange(firstWithDiff);
          }
        }
      }
    })();

    return () => {
      cancelled = true;
      clearTimeout(loadTimeout);
    };
  }, [selectedDatabaseNameList]);

  // On mount: check for target query param
  useEffect(() => {
    (async () => {
      const currentRoute = router.currentRoute.value;
      const targetDatabaseName = currentRoute.query.target as string;
      if (isValidDatabaseName(targetDatabaseName)) {
        const database =
          await databaseStore.getOrFetchDatabaseByName(targetDatabaseName);
        if (database && getInstanceResource(database).engine === sourceEngine) {
          onSelectedDatabaseNameListChange([targetDatabaseName]);
          onSelectedDatabaseNameChange(targetDatabaseName);
        }
      }
    })();
  }, []);

  const handleUnselectDatabase = useCallback(
    (database: Database) => {
      onSelectedDatabaseNameListChange(
        selectedDatabaseNameList.filter((name) => name !== database.name)
      );
      if (selectedDatabaseName === database.name) {
        onSelectedDatabaseNameChange(undefined);
      }
    },
    [
      selectedDatabaseNameList,
      selectedDatabaseName,
      onSelectedDatabaseNameListChange,
      onSelectedDatabaseNameChange,
    ]
  );

  const onStatementChange = useCallback(
    (statement: string) => {
      if (selectedDatabaseName) {
        onSchemaDiffCacheChange({
          ...schemaDiffCache,
          [selectedDatabaseName]: {
            ...schemaDiffCache[selectedDatabaseName],
            edited: statement,
          },
        });
      }
    },
    [selectedDatabaseName, schemaDiffCache, onSchemaDiffCacheChange]
  );

  const handleSelectedDatabases = useCallback(
    (nameList: string[]) => {
      onSelectedDatabaseNameListChange(nameList);
      setShowSelectPanel(false);
    },
    [onSelectedDatabaseNameListChange]
  );

  return (
    <div className="select-target-database-view h-full overflow-y-hidden flex flex-col gap-y-2">
      <SourceSchemaInfo
        sourceEngine={sourceEngine}
        changelogSourceSchema={changelogSourceSchema}
      />
      <div className="relative border rounded-lg w-full flex flex-row flex-1 overflow-hidden">
        {/* Left panel: target database list */}
        <div className="w-1/4 min-w-[256px] max-w-xs h-full border-r">
          <div className="w-full h-full relative flex flex-col justify-start items-start overflow-y-auto pb-2">
            <div className="w-full h-auto flex flex-col justify-start items-start sticky top-0 z-[1]">
              <div className="w-full bg-white border-b p-2 px-3 flex flex-row justify-between items-center sticky top-0 z-[1]">
                <span className="text-sm">
                  {t("database.sync-schema.target-databases")}
                </span>
                <button
                  className="p-0.5 rounded-sm bg-gray-100 hover:shadow-sm hover:opacity-80"
                  onClick={() => setShowSelectPanel(true)}
                >
                  <Plus className="w-4 h-auto" />
                </button>
              </div>
              {targetDatabaseList.length > 0 && (
                <div className="w-full mt-2 px-2">
                  <div className="flex rounded-xs bg-gray-100 p-0.5">
                    <button
                      className={cn(
                        "flex-1 text-xs px-2 py-1 rounded-xs transition-colors",
                        showDatabaseWithDiff
                          ? "bg-white shadow-sm font-medium"
                          : "text-gray-500 hover:text-gray-700"
                      )}
                      onClick={() => setShowDatabaseWithDiff(true)}
                    >
                      {t("database.sync-schema.with-diff")}{" "}
                      <span className="text-gray-400">
                        ({databaseListWithDiff.length})
                      </span>
                    </button>
                    <button
                      className={cn(
                        "flex-1 text-xs px-2 py-1 rounded-xs transition-colors",
                        !showDatabaseWithDiff
                          ? "bg-white shadow-sm font-medium"
                          : "text-gray-500 hover:text-gray-700"
                      )}
                      onClick={() => setShowDatabaseWithDiff(false)}
                    >
                      {t("database.sync-schema.no-diff")}{" "}
                      <span className="text-gray-400">
                        ({databaseListWithoutDiff.length})
                      </span>
                    </button>
                  </div>
                </div>
              )}
            </div>
            <div className="w-full grow flex flex-col justify-start items-start px-2 gap-1">
              {shownDatabaseList.map((database) => (
                <div
                  key={database.name}
                  className={cn(
                    "w-full group flex flex-row justify-start items-center px-2 py-1 leading-8 cursor-pointer text-sm text-ellipsis whitespace-nowrap rounded-sm hover:bg-gray-50",
                    database.name === selectedDatabaseName ? "bg-gray-100" : ""
                  )}
                  onClick={() => onSelectedDatabaseNameChange(database.name)}
                >
                  {EngineIconPath[getInstanceResource(database).engine] && (
                    <img
                      src={EngineIconPath[getInstanceResource(database).engine]}
                      className="w-4 h-auto shrink-0"
                      alt=""
                    />
                  )}
                  <span className="truncate ml-1">
                    <span className="mx-0.5 text-gray-400">
                      ({getDatabaseEnvironment(database).title})
                    </span>
                    <span>
                      {extractDatabaseResourceName(database.name).databaseName}
                    </span>
                    <span className="ml-0.5 text-gray-400">
                      ({getInstanceResource(database).title})
                    </span>
                  </span>
                  <div className="grow" />
                  <button
                    className="hidden shrink-0 group-hover:block ml-1 p-0.5 rounded-sm bg-white hover:shadow-sm"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleUnselectDatabase(database);
                    }}
                  >
                    <Minus className="w-4 h-auto text-gray-500" />
                  </button>
                </div>
              ))}
              {targetDatabaseList.length === 0 && (
                <div className="w-full h-full -mt-4 flex flex-col justify-center items-center">
                  <span className="text-gray-400">
                    {t("database.sync-schema.message.no-target-databases")}
                  </span>
                  <Button
                    className="mt-2"
                    onClick={() => setShowSelectPanel(true)}
                  >
                    <Plus className="w-4 h-auto mr-1" />
                    {t("common.select")}
                  </Button>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Right panel: diff view */}
        <div className="flex-1 h-full overflow-hidden p-2">
          {selectedDatabaseName ? (
            <DiffViewPanel
              statement={schemaDiffCache[selectedDatabaseName]?.edited ?? ""}
              engine={sourceEngine}
              targetDatabaseSchema={targetSchemaDisplayString}
              sourceDatabaseSchema={sourceSchemaDisplayString}
              shouldShowDiff={shouldShowDiff}
              previewSchemaChangeMessage={previewSchemaChangeMessage}
              onStatementChange={onStatementChange}
            />
          ) : (
            <div className="w-full h-full flex flex-col justify-center items-center">
              {t("database.sync-schema.message.select-a-target-database-first")}
            </div>
          )}
          {isLoadingDiff && (
            <div className="absolute inset-0 z-10 bg-white/40 backdrop-blur-xs w-full h-full flex flex-col justify-center items-center">
              <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-gray-400" />
              <span className="mt-1">{t("common.loading")}</span>
            </div>
          )}
        </div>
      </div>

      {showSelectPanel && (
        <TargetDatabasesSelectPanel
          project={project.name}
          engine={sourceEngine}
          selectedDatabaseNameList={selectedDatabaseNameList}
          onClose={() => setShowSelectPanel(false)}
          onUpdate={handleSelectedDatabases}
        />
      )}
    </div>
  );
}

// ============================================================
// DiffViewPanel
// ============================================================

function DiffViewPanel({
  statement,
  engine,
  targetDatabaseSchema,
  sourceDatabaseSchema,
  shouldShowDiff,
  previewSchemaChangeMessage,
  onStatementChange,
}: {
  statement: string;
  engine: Engine;
  targetDatabaseSchema: string;
  sourceDatabaseSchema: string;
  shouldShowDiff: boolean;
  previewSchemaChangeMessage: string;
  onStatementChange: (statement: string) => void;
}) {
  const { t } = useTranslation();
  const [tab, setTab] = useState<"diff" | "ddl">("diff");

  return (
    <div className="w-full h-full flex flex-col gap-y-2">
      {/* Tabs */}
      <div className="flex border-b border-control-border gap-x-4">
        <button
          className={cn(
            "relative px-1 pb-2 text-sm font-medium transition-colors cursor-pointer",
            tab === "diff"
              ? "text-accent after:absolute after:inset-x-0 after:-bottom-px after:h-0.5 after:bg-accent"
              : "text-control-light hover:text-control"
          )}
          onClick={() => setTab("diff")}
        >
          {t("database.sync-schema.schema-change")}
        </button>
        <button
          className={cn(
            "relative px-1 pb-2 text-sm font-medium transition-colors cursor-pointer",
            tab === "ddl"
              ? "text-accent after:absolute after:inset-x-0 after:-bottom-px after:h-0.5 after:bg-accent"
              : "text-control-light hover:text-control"
          )}
          onClick={() => setTab("ddl")}
        >
          {t("database.sync-schema.generated-ddl-statement")}
        </button>
      </div>

      <div className="flex-1 w-full flex flex-col gap-y-2 overflow-hidden">
        {tab === "diff" &&
          (shouldShowDiff ? (
            <SchemaDiffViewer
              title={previewSchemaChangeMessage}
              original={targetDatabaseSchema}
              modified={sourceDatabaseSchema}
              showFullscreen
            />
          ) : (
            <div className="w-full flex-1 border flex items-center justify-center">
              <p>{t("database.sync-schema.message.no-diff-found")}</p>
            </div>
          ))}
        {tab === "ddl" && (
          <>
            <div className="w-full flex flex-col justify-start">
              <div className="flex flex-row justify-start items-center gap-x-2">
                <span>{t("database.sync-schema.synchronize-statements")}</span>
                <CopyButton content={statement} />
              </div>
              <div className="textinfolabel">
                {t("database.sync-schema.synchronize-statements-description")}
              </div>
            </div>
            <MonacoEditorPanel
              content={statement}
              language={dialectOfEngineV1(engine)}
              onChange={onStatementChange}
            />
          </>
        )}
      </div>
    </div>
  );
}

// ============================================================
// SchemaDiffViewer
// ============================================================

function SchemaDiffViewer({
  title,
  original,
  modified,
  showFullscreen,
}: {
  title: string;
  original: string;
  modified: string;
  showFullscreen?: boolean;
}) {
  const normalizedOriginal = useMemo(
    () => original.replace(/\r\n?/g, "\n"),
    [original]
  );
  const normalizedModified = useMemo(
    () => modified.replace(/\r\n?/g, "\n"),
    [modified]
  );
  const containerRef = useRef<HTMLDivElement>(null);
  // biome-ignore lint/suspicious/noExplicitAny: Monaco diff editor instance
  const editorRef = useRef<any>(null); // eslint-disable-line @typescript-eslint/no-explicit-any
  const [showModal, setShowModal] = useState(false);

  useEffect(() => {
    let disposed = false;
    (async () => {
      if (!containerRef.current) return;
      const editor = await createMonacoDiffEditor({
        container: containerRef.current,
        options: {
          readOnly: true,
          ignoreTrimWhitespace: true,
        },
      });
      if (disposed) {
        editor.dispose();
        return;
      }
      editorRef.current = editor;
      const { editor: monacoEditor } = await import("monaco-editor");
      editor.setModel({
        original: monacoEditor.createModel(normalizedOriginal, "sql"),
        modified: monacoEditor.createModel(normalizedModified, "sql"),
      });
    })();
    return () => {
      disposed = true;
      const model = editorRef.current?.getModel();
      model?.original?.dispose();
      model?.modified?.dispose();
      editorRef.current?.dispose();
      editorRef.current = null;
    };
  }, []);

  // Update models when content changes
  useEffect(() => {
    const editor = editorRef.current;
    if (!editor) return;
    const model = editor.getModel();
    if (model) {
      model.original.setValue(normalizedOriginal);
      model.modified.setValue(normalizedModified);
    }
  }, [normalizedOriginal, normalizedModified]);

  const handleNavigate = useCallback((direction: "next" | "previous") => {
    editorRef.current?.goToDiff(direction);
  }, []);

  return (
    <div className="w-full h-full flex flex-col gap-2">
      <div className="w-full flex flex-row justify-between items-center">
        <span>{title}</span>
        <div className="flex gap-x-2 shrink-0">
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleNavigate("previous")}
          >
            <ArrowUp className="w-5 h-auto" />
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleNavigate("next")}
          >
            <ArrowDown className="w-5 h-auto" />
          </Button>
          {showFullscreen && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowModal(true)}
            >
              <Maximize2 className="w-5 h-auto" />
            </Button>
          )}
        </div>
      </div>
      <div className="w-full flex-1 overflow-y-scroll border">
        <div ref={containerRef} className="h-full" />
      </div>

      {showModal && (
        <SchemaDiffViewerModal
          title={title}
          original={normalizedOriginal}
          modified={normalizedModified}
          onClose={() => setShowModal(false)}
        />
      )}
    </div>
  );
}

// ============================================================
// SchemaDiffViewerModal
// ============================================================

function SchemaDiffViewerModal({
  title,
  original,
  modified,
  onClose,
}: {
  title: string;
  original: string;
  modified: string;
  onClose: () => void;
}) {
  useEscapeKey(true, onClose);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="relative bg-white w-full h-screen flex flex-col p-4">
        <div className="flex items-center justify-between mb-2">
          <h2 className="text-lg font-semibold">{title}</h2>
          <Button variant="ghost" size="sm" onClick={onClose}>
            &times;
          </Button>
        </div>
        <div className="flex-1 overflow-hidden">
          <SchemaDiffViewer title="" original={original} modified={modified} />
        </div>
      </div>
    </div>
  );
}

// ============================================================
// MonacoEditorPanel (for DDL editor)
// ============================================================

function MonacoEditorPanel({
  content,
  language,
  onChange,
}: {
  content: string;
  language: string;
  onChange: (value: string) => void;
}) {
  const containerRef = useRef<HTMLDivElement>(null);
  // biome-ignore lint/suspicious/noExplicitAny: Monaco editor instance
  const editorRef = useRef<any>(null); // eslint-disable-line @typescript-eslint/no-explicit-any
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  const contentRef = useRef(content);

  useEffect(() => {
    let disposed = false;
    (async () => {
      if (!containerRef.current) return;
      const editor = await createMonacoEditor({
        container: containerRef.current,
        options: {
          language,
          value: content,
        },
      });
      if (disposed) {
        editor.dispose();
        return;
      }
      editorRef.current = editor;
      editor.onDidChangeModelContent(() => {
        const val = editor.getValue();
        contentRef.current = val;
        onChangeRef.current(val);
      });
    })();
    return () => {
      disposed = true;
      editorRef.current?.dispose();
      editorRef.current = null;
    };
  }, []);

  // Sync content from outside
  useEffect(() => {
    if (editorRef.current && content !== contentRef.current) {
      contentRef.current = content;
      editorRef.current.setValue(content);
    }
  }, [content]);

  return <div ref={containerRef} className="w-full flex-1 border" />;
}

// ============================================================
// CopyButton
// ============================================================

function CopyButton({ content }: { content: string }) {
  const { t } = useTranslation();

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(content);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.copied"),
      });
    } catch {
      // ignore
    }
  }, [content, t]);

  return (
    <Button variant="ghost" size="sm" onClick={handleCopy}>
      <Copy className="w-4 h-4" />
    </Button>
  );
}

// ============================================================
// TargetDatabasesSelectPanel (Drawer)
// ============================================================

function TargetDatabasesSelectPanel({
  project,
  engine,
  selectedDatabaseNameList: initialSelected,
  onClose,
  onUpdate,
}: {
  project: string;
  engine: Engine;
  selectedDatabaseNameList?: string[];
  onClose: () => void;
  onUpdate: (databaseNameList: string[]) => void;
}) {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();
  const [selected, setSelected] = useState<Set<string>>(
    new Set(initialSelected || [])
  );
  const [searchQuery, setSearchQuery] = useState("");
  const [databases, setDatabases] = useState<Database[]>([]);
  const [loading, setLoading] = useState(true);
  const [dbNextPageToken, setDbNextPageToken] = useState("");
  const [loadingMoreDbs, setLoadingMoreDbs] = useState(false);

  useEscapeKey(true, onClose);

  // Fetch first page of databases
  useEffect(() => {
    (async () => {
      setLoading(true);
      const { databases: fetched, nextPageToken: token } =
        await databaseStore.fetchDatabases({
          parent: project,
          pageSize: 100,
        });
      setDatabases(fetched);
      setDbNextPageToken(token);
      setLoading(false);
    })();
  }, [project]);

  const loadMoreDatabases = useCallback(async () => {
    if (!dbNextPageToken || loadingMoreDbs) return;
    setLoadingMoreDbs(true);
    const { databases: more, nextPageToken: token } =
      await databaseStore.fetchDatabases({
        parent: project,
        pageSize: 100,
        pageToken: dbNextPageToken,
      });
    setDatabases((prev) => [...prev, ...more]);
    setDbNextPageToken(token);
    setLoadingMoreDbs(false);
  }, [dbNextPageToken, loadingMoreDbs, project]);

  const filteredDatabases = useMemo(() => {
    const q = searchQuery.toLowerCase();
    return databases.filter((db) => {
      const dbName = extractDatabaseResourceName(
        db.name
      ).databaseName.toLowerCase();
      const envName = getDatabaseEnvironment(db).title.toLowerCase();
      const instName = getInstanceResource(db).title.toLowerCase();
      const matchesQuery =
        !q || dbName.includes(q) || envName.includes(q) || instName.includes(q);
      const matchesEngine =
        engine === Engine.ENGINE_UNSPECIFIED ||
        getInstanceResource(db).engine === engine;
      return matchesQuery && matchesEngine;
    });
  }, [databases, searchQuery, engine]);

  const toggleDatabase = useCallback((name: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(name)) {
        next.delete(name);
      } else {
        next.add(name);
      }
      return next;
    });
  }, []);

  const toggleAll = useCallback(() => {
    const allFilteredNames = filteredDatabases.map((db) => db.name);
    setSelected((prev) => {
      const allSelected = allFilteredNames.every((name) => prev.has(name));
      const next = new Set(prev);
      if (allSelected) {
        allFilteredNames.forEach((name) => next.delete(name));
      } else {
        allFilteredNames.forEach((name) => next.add(name));
      }
      return next;
    });
  }, [filteredDatabases]);

  const handleConfirm = useCallback(() => {
    onUpdate(Array.from(selected));
  }, [selected, onUpdate]);

  return (
    <div className="fixed inset-0 z-50 flex">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="ml-auto relative bg-white w-[64rem] max-w-[100vw] h-full shadow-lg flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-control-border">
          <h2 className="text-lg font-semibold">
            {t("database.sync-schema.target-databases")}
          </h2>
          <button
            className="p-1 hover:bg-control-bg rounded-xs"
            onClick={onClose}
          >
            &times;
          </button>
        </div>

        {/* Search */}
        <div className="px-6 pt-4">
          <input
            type="text"
            className="w-full border border-gray-300 rounded-xs h-9 px-3 text-sm"
            placeholder={t("database.filter-database")}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        </div>

        {/* Database list */}
        <div className="flex-1 overflow-y-auto px-6 py-2">
          {loading ? (
            <div className="flex items-center justify-center py-8">
              <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-gray-400" />
            </div>
          ) : (
            <>
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b">
                    <th className="py-2 px-2 w-8 text-left">
                      <input
                        type="checkbox"
                        checked={
                          filteredDatabases.length > 0 &&
                          filteredDatabases.every((db) => selected.has(db.name))
                        }
                        onChange={toggleAll}
                      />
                    </th>
                    <th className="py-2 px-2 text-left font-medium">
                      {t("common.database")}
                    </th>
                    <th className="py-2 px-2 text-left font-medium">
                      {t("common.environment")}
                    </th>
                    <th className="py-2 px-2 text-left font-medium">
                      {t("common.instance")}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {filteredDatabases.map((db) => (
                    <tr
                      key={db.name}
                      className="border-b hover:bg-gray-50 cursor-pointer"
                      onClick={() => toggleDatabase(db.name)}
                    >
                      <td
                        className="py-2 px-2"
                        onClick={(e) => e.stopPropagation()}
                      >
                        <input
                          type="checkbox"
                          checked={selected.has(db.name)}
                          onChange={() => toggleDatabase(db.name)}
                        />
                      </td>
                      <td className="py-2 px-2">
                        <div className="flex items-center gap-x-1">
                          {EngineIconPath[getInstanceResource(db).engine] && (
                            <img
                              src={
                                EngineIconPath[getInstanceResource(db).engine]
                              }
                              className="w-4 h-auto"
                              alt=""
                            />
                          )}
                          {extractDatabaseResourceName(db.name).databaseName}
                        </div>
                      </td>
                      <td className="py-2 px-2">
                        {getDatabaseEnvironment(db).title}
                      </td>
                      <td className="py-2 px-2">
                        {getInstanceResource(db).title}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              {dbNextPageToken && (
                <div className="flex justify-center py-3">
                  <Button
                    variant="ghost"
                    size="sm"
                    disabled={loadingMoreDbs}
                    onClick={loadMoreDatabases}
                  >
                    {loadingMoreDbs
                      ? t("common.loading")
                      : t("common.load-more")}
                  </Button>
                </div>
              )}
            </>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between px-6 py-4 border-t border-control-border">
          <div className="textinfolabel">
            {t("database.selected-n-databases", { n: selected.size })}
          </div>
          <div className="flex items-center justify-end gap-x-2">
            <Button variant="outline" onClick={onClose}>
              {t("common.cancel")}
            </Button>
            <Button disabled={selected.size === 0} onClick={handleConfirm}>
              {t("common.select")}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
