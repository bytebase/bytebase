import { create } from "@bufbuild/protobuf";
import { Database as DatabaseIcon, FolderTree, Loader2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import { EngineIcon } from "@/react/components/EngineIcon";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { IssueLabelSelect } from "@/react/components/IssueLabelSelect";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import { Input } from "@/react/components/ui/input";
import { SearchInput } from "@/react/components/ui/search-input";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { Switch } from "@/react/components/ui/switch";
import { useSessionPageSize } from "@/react/hooks/useSessionPageSize";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  DEFAULT_MAX_RESULT_SIZE_IN_MB,
  experimentalCreateIssueByPlan,
  pushNotification,
  useCurrentUserV1,
  useDatabaseV1Store,
  useDBGroupStore,
  useProjectV1Store,
  useSettingV1Store,
  useSheetV1Store,
} from "@/store";
import { isValidDatabaseGroupName, isValidDatabaseName } from "@/types";
import { ExportFormat } from "@/types/proto-es/v1/common_pb";
import type { DatabaseGroup } from "@/types/proto-es/v1/database_group_service_pb";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { Issue_Type, IssueSchema } from "@/types/proto-es/v1/issue_service_pb";
import {
  Plan_ExportDataConfigSchema,
  Plan_SpecSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";
import {
  extractDatabaseGroupName,
  extractDatabaseResourceName,
  extractIssueUID,
  extractProjectResourceName,
  generatePlanTitle,
  getDatabaseEnvironment,
  getInstanceResource,
  normalizeTitle,
  setSheetStatement,
} from "@/utils";

// ---------------------------------------------------------------------------
// Main Sheet
// ---------------------------------------------------------------------------

export interface DataExportPrepSeed {
  /** Pre-selected database resource names. */
  selectedDatabaseNames?: string[];
  /** Start directly at step 2 (Configure). */
  step?: 1 | 2;
}

export interface DataExportPrepSheetProps {
  open: boolean;
  onClose: () => void;
  projectName: string;
  seed?: DataExportPrepSeed;
}

type Step = 1 | 2;

const EXPORT_FORMATS = [
  { value: ExportFormat.CSV, label: "CSV" },
  { value: ExportFormat.JSON, label: "JSON" },
  { value: ExportFormat.SQL, label: "SQL" },
  { value: ExportFormat.XLSX, label: "XLSX" },
] as const;

export function DataExportPrepSheet({
  open,
  onClose,
  projectName,
  seed,
}: DataExportPrepSheetProps) {
  const { t } = useTranslation();
  const currentUser = useCurrentUserV1();
  const sheetStore = useSheetV1Store();
  const dbStore = useDatabaseV1Store();
  const dbGroupStore = useDBGroupStore();
  const projectStore = useProjectV1Store();
  const settingStore = useSettingV1Store();

  const project = useVueState(() => projectStore.getProjectByName(projectName));

  const [step, setStep] = useState<Step>(1);
  const [creating, setCreating] = useState(false);

  // Step 1: target selection
  const [changeSource, setChangeSource] = useState<"DATABASE" | "GROUP">(
    "DATABASE"
  );
  const [selectedDatabaseNames, setSelectedDatabaseNames] = useState<
    Set<string>
  >(new Set());
  const [selectedDatabaseGroup, setSelectedDatabaseGroup] = useState<
    string | undefined
  >();

  // Step 2: form fields
  const [title, setTitle] = useState("");
  const [titleEdited, setTitleEdited] = useState(false);
  const [description, setDescription] = useState("");
  const [labels, setLabels] = useState<string[]>([]);
  const [statement, setStatement] = useState("");
  const [format, setFormat] = useState(ExportFormat.JSON);
  const [password, setPassword] = useState("");
  const [passwordEnabled, setPasswordEnabled] = useState(false);

  const targets = useMemo(() => {
    if (changeSource === "DATABASE") {
      return [...selectedDatabaseNames];
    }
    return selectedDatabaseGroup ? [selectedDatabaseGroup] : [];
  }, [changeSource, selectedDatabaseNames, selectedDatabaseGroup]);

  const validSelectState = targets.length > 0;

  const targetTitleNames = useMemo(
    () =>
      targets.map((target) => {
        if (isValidDatabaseName(target)) {
          return extractDatabaseResourceName(target).databaseName;
        }
        if (isValidDatabaseGroupName(target)) {
          return extractDatabaseGroupName(target);
        }
        return target;
      }),
    [targets]
  );

  const canCreate = useMemo(() => {
    if (!validSelectState) return false;
    if (project?.enforceIssueTitle && !normalizeTitle(title)) return false;
    if (project?.forceIssueLabels && labels.length === 0) return false;
    if (!statement.trim()) return false;
    return true;
  }, [validSelectState, project, title, labels, statement]);

  const effectiveTitle = useMemo(() => {
    const normalized = normalizeTitle(title);
    if (normalized) return normalized;
    return generatePlanTitle("bb.plan.export-data", targetTitleNames);
  }, [title, targetTitleNames]);

  // Auto-generate title when targets change
  const targetKey = targetTitleNames.join(",");
  useEffect(() => {
    if (project?.enforceIssueTitle) return;
    if (targetTitleNames.length === 0) return;
    if (titleEdited && normalizeTitle(title)) return;
    setTitle(generatePlanTitle("bb.plan.export-data", targetTitleNames));
  }, [targetKey]);

  // Fetch database/group metadata for display
  useEffect(() => {
    for (const target of targets) {
      if (isValidDatabaseName(target)) {
        dbStore.getOrFetchDatabaseByName(target);
      } else if (isValidDatabaseGroupName(target)) {
        dbGroupStore.getOrFetchDBGroupByName(target, {
          view: DatabaseGroupView.FULL,
        });
      }
    }
  }, [targets, dbStore, dbGroupStore]);

  // Reset on open or when seed changes while open
  const seedKey = seed?.selectedDatabaseNames?.join(",") ?? "";
  const seedStep = seed?.step ?? 1;
  useEffect(() => {
    if (open) {
      setStep(seedStep);
      setCreating(false);
      setChangeSource("DATABASE");
      setSelectedDatabaseNames(new Set(seed?.selectedDatabaseNames ?? []));
      setSelectedDatabaseGroup(undefined);
      setTitle("");
      setTitleEdited(false);
      setDescription("");
      setLabels([]);
      setStatement("");
      setFormat(ExportFormat.JSON);
      setPassword("");
      setPasswordEnabled(false);
    }
  }, [open, seedKey, seedStep]);

  // Close the sheet if the project context changes while open
  useEffect(() => {
    if (open) {
      onClose();
    }
  }, [projectName]);

  // Limits
  const maximumResultSize = useVueState(() => {
    let size = settingStore.workspaceProfile.sqlResultSize;
    if (size <= 0) {
      size = BigInt(DEFAULT_MAX_RESULT_SIZE_IN_MB * 1024 * 1024);
    }
    return Number(size) / 1024 / 1024;
  });

  const handleCancel = () => {
    if (step === 2 && !seed?.step) {
      // Only go back to step 1 if the user navigated here naturally.
      // When seeded at step 2 (e.g. from database list), close instead.
      setStep(1);
      return;
    }
    onClose();
  };

  const handleCreate = async () => {
    if (creating || !canCreate || !project) return;
    setCreating(true);

    try {
      const sheet = create(SheetSchema, {});
      setSheetStatement(sheet, statement);
      const createdSheet = await sheetStore.createSheet(project.name, sheet);

      const spec = create(Plan_SpecSchema, {
        id: uuidv4(),
        config: {
          case: "exportDataConfig",
          value: create(Plan_ExportDataConfigSchema, {
            targets,
            sheet: createdSheet.name,
            format,
            password: passwordEnabled ? password : "",
          }),
        },
      });

      const planCreate = create(PlanSchema, {
        title: effectiveTitle,
        description,
        specs: [spec],
        creator: currentUser.value.name,
      });

      const issueCreate = create(IssueSchema, {
        title: effectiveTitle,
        description,
        creator: `users/${currentUser.value.email}`,
        labels,
        type: Issue_Type.DATABASE_EXPORT,
      });

      const { createdIssue } = await experimentalCreateIssueByPlan(
        project,
        issueCreate,
        planCreate,
        { skipRollout: true }
      );

      onClose();
      router.push({
        name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
        params: {
          projectId: extractProjectResourceName(createdIssue.name),
          issueId: extractIssueUID(createdIssue.name),
        },
      });
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    } finally {
      setCreating(false);
    }
  };

  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="wide">
        <SheetHeader>
          <div className="flex flex-col gap-y-3">
            <SheetTitle>
              {t("custom-approval.risk-rule.risk.namespace.data_export")}
            </SheetTitle>
            {/* Steps indicator */}
            <div className="flex items-center gap-x-4 text-sm">
              <StepIndicator
                number={1}
                label={t("plan.targets.title")}
                active={step === 1}
                completed={step > 1}
              />
              <div className="h-px w-8 bg-control-border" />
              <StepIndicator
                number={2}
                label={t("common.configure")}
                active={step === 2}
                completed={false}
              />
            </div>
          </div>
        </SheetHeader>

        <SheetBody>
          {step === 1 ? (
            <DatabaseAndGroupSelector
              projectName={projectName}
              changeSource={changeSource}
              onChangeSourceChange={setChangeSource}
              selectedDatabaseNames={selectedDatabaseNames}
              onSelectedDatabaseNamesChange={setSelectedDatabaseNames}
              selectedDatabaseGroup={selectedDatabaseGroup}
              onSelectedDatabaseGroupChange={setSelectedDatabaseGroup}
            />
          ) : (
            <div className="flex flex-col gap-y-4 pb-2">
              {/* Targets display */}
              <div className="flex flex-col gap-y-2">
                <h3 className="text-base font-medium">
                  {t("plan.targets.title")}
                </h3>
                <div className="flex flex-wrap gap-2">
                  {targets.map((target) => (
                    <TargetBadge key={target} target={target} />
                  ))}
                </div>
              </div>

              {/* Title */}
              <div className="flex flex-col gap-y-2">
                <label className="text-sm font-medium text-control">
                  {t("common.title")}
                  {project?.enforceIssueTitle && (
                    <span className="text-error"> *</span>
                  )}
                </label>
                <Input
                  value={title}
                  placeholder={t("common.title")}
                  onChange={(e) => {
                    const next = e.target.value;
                    setTitle(next);
                    // Invariant: titleEdited ⇒ title is non-empty user intent.
                    // See CreateDatabaseSheet for the detailed rationale.
                    setTitleEdited(next !== "");
                  }}
                />
              </div>

              {/* Description */}
              <div className="flex flex-col gap-y-2">
                <label className="text-sm font-medium text-control">
                  {t("common.description")}
                </label>
                <textarea
                  className="w-full border border-control-border rounded-sm px-3 py-2 text-sm focus:outline-none focus:border-accent min-h-[6rem] resize-y"
                  value={description}
                  placeholder={t("common.description")}
                  onChange={(e) => setDescription(e.target.value)}
                />
              </div>

              {/* Labels */}
              {project &&
                (project.issueLabels.length > 0 ||
                  project.forceIssueLabels) && (
                  <IssueLabelSelect
                    labels={project.issueLabels}
                    selected={labels}
                    required={project.forceIssueLabels}
                    onChange={setLabels}
                  />
                )}

              {/* SQL */}
              <div className="flex flex-col gap-y-2">
                <label className="text-sm font-medium text-control">
                  {t("common.sql")}
                  <span className="text-error"> *</span>
                </label>
                <textarea
                  className="w-full h-96 border border-control-border rounded-sm px-3 py-2 text-sm font-mono focus:outline-none focus:border-accent resize-y"
                  value={statement}
                  placeholder="SELECT ..."
                  onChange={(e) => setStatement(e.target.value)}
                />
              </div>

              {/* Export options */}
              <div className="flex flex-col gap-y-2">
                <h3 className="text-base">{t("issue.data-export.options")}</h3>
                <div className="p-3 border rounded-sm flex flex-col gap-y-3">
                  {/* Format */}
                  <div className="flex items-center gap-4">
                    <span className="text-sm">
                      {t("issue.data-export.format")}
                    </span>
                    <div className="flex items-center gap-x-3">
                      {EXPORT_FORMATS.map((f) => (
                        <label
                          key={f.value}
                          className="inline-flex items-center gap-x-1.5 text-sm cursor-pointer"
                        >
                          <input
                            type="radio"
                            name="export-format"
                            checked={format === f.value}
                            onChange={() => setFormat(f.value)}
                            className="accent-accent"
                          />
                          {f.label}
                        </label>
                      ))}
                    </div>
                  </div>

                  {/* Password */}
                  <div className="flex flex-col gap-y-2">
                    <div className="flex items-center gap-x-2">
                      <Switch
                        checked={passwordEnabled}
                        onCheckedChange={(checked) => {
                          setPasswordEnabled(checked);
                          if (!checked) setPassword("");
                        }}
                      />
                      <span className="text-sm">
                        {t("export-data.password-optional")}
                      </span>
                    </div>
                    {passwordEnabled && (
                      <Input
                        type="password"
                        value={password}
                        placeholder={t("export-data.password-optional")}
                        onChange={(e) => setPassword(e.target.value)}
                      />
                    )}
                  </div>
                </div>
              </div>

              {/* Limits */}
              <div className="w-full flex flex-col gap-y-2">
                <h3 className="text-base font-medium">
                  {t("issue.data-export.limits")}
                </h3>
                <div className="flex items-center gap-x-2">
                  <span className="text-sm">
                    {t(
                      "settings.general.workspace.maximum-sql-result.size.self"
                    )}
                  </span>
                  <span className="font-medium">{maximumResultSize} MB</span>
                </div>
              </div>
            </div>
          )}
        </SheetBody>

        <SheetFooter>
          <Button variant="outline" onClick={handleCancel}>
            {step === 1 ? t("common.cancel") : t("common.back")}
          </Button>
          {step === 1 ? (
            <Button disabled={!validSelectState} onClick={() => setStep(2)}>
              {t("common.next")}
            </Button>
          ) : (
            <Button disabled={!canCreate || creating} onClick={handleCreate}>
              {creating && <Loader2 className="size-4 mr-1 animate-spin" />}
              {t("common.create")}
            </Button>
          )}
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

// ---------------------------------------------------------------------------
// StepIndicator
// ---------------------------------------------------------------------------

function StepIndicator({
  number,
  label,
  active,
  completed,
}: {
  number: number;
  label: string;
  active: boolean;
  completed: boolean;
}) {
  return (
    <div className="flex items-center gap-x-2">
      <span
        className={cn(
          "size-6 rounded-full flex items-center justify-center text-xs font-medium",
          active
            ? "bg-accent text-accent-text"
            : completed
              ? "bg-success text-accent-text"
              : "bg-control-bg-hover text-control-light"
        )}
      >
        {completed ? (
          <svg className="size-3.5" fill="currentColor" viewBox="0 0 20 20">
            <path
              fillRule="evenodd"
              d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
              clipRule="evenodd"
            />
          </svg>
        ) : (
          number
        )}
      </span>
      <span
        className={cn(
          "text-sm",
          active ? "text-main font-medium" : "text-control-light"
        )}
      >
        {label}
      </span>
    </div>
  );
}

// ---------------------------------------------------------------------------
// TargetBadge (database or group display in step 2)
// ---------------------------------------------------------------------------

function TargetBadge({ target }: { target: string }) {
  const dbStore = useDatabaseV1Store();
  const isDatabaseTarget = isValidDatabaseName(target);
  const isGroupTarget = isValidDatabaseGroupName(target);

  // Always call useVueState unconditionally (rules of hooks)
  const db = useVueState(() =>
    isDatabaseTarget ? dbStore.getDatabaseByName(target) : undefined
  );

  if (isDatabaseTarget && db) {
    const inst = getInstanceResource(db);
    const env = getDatabaseEnvironment(db);
    const { databaseName } = extractDatabaseResourceName(target);
    return (
      <div className="inline-flex items-center gap-2 px-2 py-1.5 border rounded-sm min-w-0">
        {inst && <EngineIcon engine={inst.engine} className="size-4" />}
        {env && <EnvironmentLabel environmentName={env.name} />}
        <span className="text-sm truncate">{databaseName}</span>
      </div>
    );
  }

  if (isGroupTarget) {
    const groupName = extractDatabaseGroupName(target);
    return (
      <div className="inline-flex items-center gap-2 px-2 py-1.5 border rounded-sm min-w-0">
        <FolderTree className="size-4 shrink-0 text-control-light" />
        <span className="text-sm truncate">{groupName}</span>
      </div>
    );
  }

  return (
    <div className="inline-flex items-center gap-2 px-2 py-1.5 border rounded-sm min-w-0">
      <span className="text-sm truncate">{target}</span>
    </div>
  );
}

// ---------------------------------------------------------------------------
// DatabaseAndGroupSelector (step 1)
// ---------------------------------------------------------------------------

function DatabaseAndGroupSelector({
  projectName,
  changeSource,
  onChangeSourceChange,
  selectedDatabaseNames,
  onSelectedDatabaseNamesChange,
  selectedDatabaseGroup,
  onSelectedDatabaseGroupChange,
}: {
  projectName: string;
  changeSource: "DATABASE" | "GROUP";
  onChangeSourceChange: (source: "DATABASE" | "GROUP") => void;
  selectedDatabaseNames: Set<string>;
  onSelectedDatabaseNamesChange: (names: Set<string>) => void;
  selectedDatabaseGroup: string | undefined;
  onSelectedDatabaseGroupChange: (name: string | undefined) => void;
}) {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col gap-y-3">
      {/* Tab bar */}
      <div className="flex border-b border-control-border">
        <button
          type="button"
          className={cn(
            "px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors",
            changeSource === "DATABASE"
              ? "border-accent text-accent"
              : "border-transparent text-control-light hover:text-control"
          )}
          onClick={() => onChangeSourceChange("DATABASE")}
        >
          <span className="inline-flex items-center gap-x-1.5">
            <DatabaseIcon className="size-4" />
            {t("common.databases")}
          </span>
        </button>
        <button
          type="button"
          className={cn(
            "px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors",
            changeSource === "GROUP"
              ? "border-accent text-accent"
              : "border-transparent text-control-light hover:text-control"
          )}
          onClick={() => onChangeSourceChange("GROUP")}
        >
          <span className="inline-flex items-center gap-x-1.5">
            <FolderTree className="size-4" />
            {t("common.database-group")}
          </span>
        </button>
      </div>

      {changeSource === "DATABASE" ? (
        <DatabaseSelector
          projectName={projectName}
          selectedNames={selectedDatabaseNames}
          onSelectedNamesChange={onSelectedDatabaseNamesChange}
        />
      ) : (
        <DatabaseGroupSelector
          projectName={projectName}
          selectedGroup={selectedDatabaseGroup}
          onSelectedGroupChange={onSelectedDatabaseGroupChange}
        />
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// DatabaseSelector (table with checkboxes)
// ---------------------------------------------------------------------------

function DatabaseSelector({
  projectName,
  selectedNames,
  onSelectedNamesChange,
}: {
  projectName: string;
  selectedNames: Set<string>;
  onSelectedNamesChange: (names: Set<string>) => void;
}) {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();

  const [databases, setDatabases] = useState<Database[]>([]);
  const [loading, setLoading] = useState(true);
  const [hasMore, setHasMore] = useState(false);
  const [isFetchingMore, setIsFetchingMore] = useState(false);
  const [query, setQuery] = useState("");
  const [pageSize] = useSessionPageSize("bb.export-db-selector");
  const nextPageTokenRef = useRef("");
  const fetchIdRef = useRef(0);

  const doFetch = useCallback(
    async (isRefresh: boolean) => {
      const currentFetchId = ++fetchIdRef.current;
      if (isRefresh) {
        setLoading(true);
      } else {
        setIsFetchingMore(true);
      }
      try {
        const token = isRefresh ? "" : nextPageTokenRef.current;
        const filter = { query };
        const result = await databaseStore.fetchDatabases({
          parent: projectName,
          pageSize,
          pageToken: token || undefined,
          filter,
        });
        if (currentFetchId !== fetchIdRef.current) return;
        setDatabases((prev) =>
          isRefresh ? result.databases : [...prev, ...result.databases]
        );
        nextPageTokenRef.current = result.nextPageToken;
        setHasMore(Boolean(result.nextPageToken));
      } finally {
        if (currentFetchId === fetchIdRef.current) {
          setLoading(false);
          setIsFetchingMore(false);
        }
      }
    },
    [databaseStore, projectName, pageSize, query]
  );

  const isFirstLoad = useRef(true);
  useEffect(() => {
    if (isFirstLoad.current) {
      isFirstLoad.current = false;
      doFetch(true);
      return;
    }
    const timer = setTimeout(() => doFetch(true), 300);
    return () => clearTimeout(timer);
  }, [doFetch]);

  const toggleDatabase = (name: string) => {
    const next = new Set(selectedNames);
    if (next.has(name)) {
      next.delete(name);
    } else {
      next.add(name);
    }
    onSelectedNamesChange(next);
  };

  const toggleAll = () => {
    const allSelected = databases.every((db) => selectedNames.has(db.name));
    if (allSelected) {
      onSelectedNamesChange(new Set());
    } else {
      onSelectedNamesChange(new Set(databases.map((db) => db.name)));
    }
  };

  const allSelected =
    databases.length > 0 && databases.every((db) => selectedNames.has(db.name));
  const someSelected =
    databases.some((db) => selectedNames.has(db.name)) && !allSelected;

  return (
    <div className="flex flex-col gap-y-2">
      <SearchInput
        placeholder={t("database.filter-database")}
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />

      {loading ? (
        <div className="flex justify-center py-8 text-control-light">
          <Loader2 className="size-5 animate-spin" />
        </div>
      ) : databases.length === 0 ? (
        <div className="flex justify-center py-8 text-control-light">
          {t("common.no-data")}
        </div>
      ) : (
        <>
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left text-control-light">
                <th className="py-2 pr-2 w-8">
                  <Checkbox
                    checked={someSelected ? "indeterminate" : allSelected}
                    onCheckedChange={toggleAll}
                  />
                </th>
                <th className="py-2 pr-4 font-medium">
                  {t("common.database")}
                </th>
                <th className="py-2 pr-4 font-medium">
                  {t("common.environment")}
                </th>
              </tr>
            </thead>
            <tbody>
              {databases.map((db) => {
                const { databaseName } = extractDatabaseResourceName(db.name);
                const inst = getInstanceResource(db);
                const env = getDatabaseEnvironment(db);
                const isSelected = selectedNames.has(db.name);
                return (
                  <tr
                    key={db.name}
                    className={cn(
                      "border-b cursor-pointer hover:bg-control-bg",
                      isSelected && "bg-accent/5"
                    )}
                    onClick={() => toggleDatabase(db.name)}
                  >
                    <td className="py-2 pr-2">
                      <Checkbox checked={isSelected} />
                    </td>
                    <td className="py-2 pr-4">
                      <div className="flex items-center gap-x-1.5">
                        {inst && (
                          <EngineIcon engine={inst.engine} className="size-4" />
                        )}
                        <span>{databaseName}</span>
                      </div>
                    </td>
                    <td className="py-2 pr-4">
                      {env && <EnvironmentLabel environmentName={env.name} />}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>

          {hasMore && (
            <div className="flex justify-center">
              <Button
                variant="ghost"
                size="sm"
                disabled={isFetchingMore}
                onClick={() => doFetch(false)}
              >
                {isFetchingMore ? t("common.loading") : t("common.load-more")}
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// DatabaseGroupSelector
// ---------------------------------------------------------------------------

function DatabaseGroupSelector({
  projectName,
  selectedGroup,
  onSelectedGroupChange,
}: {
  projectName: string;
  selectedGroup: string | undefined;
  onSelectedGroupChange: (name: string | undefined) => void;
}) {
  const { t } = useTranslation();
  const dbGroupStore = useDBGroupStore();
  const [groups, setGroups] = useState<DatabaseGroup[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    dbGroupStore
      .fetchDBGroupListByProjectName(projectName, DatabaseGroupView.BASIC)
      .then((result) => {
        setGroups(result);
      })
      .finally(() => setLoading(false));
  }, [projectName, dbGroupStore]);

  if (loading) {
    return (
      <div className="flex justify-center py-8 text-control-light">
        <Loader2 className="size-5 animate-spin" />
      </div>
    );
  }

  if (groups.length === 0) {
    return (
      <div className="flex justify-center py-8 text-control-light">
        {t("common.no-data")}
      </div>
    );
  }

  return (
    <table className="w-full text-sm">
      <thead>
        <tr className="border-b text-left text-control-light">
          <th className="py-2 pr-2 w-8" />
          <th className="py-2 pr-4 font-medium">
            {t("common.database-group")}
          </th>
        </tr>
      </thead>
      <tbody>
        {groups.map((group) => {
          const isSelected = selectedGroup === group.name;
          return (
            <tr
              key={group.name}
              className={cn(
                "border-b cursor-pointer hover:bg-control-bg",
                isSelected && "bg-accent/5"
              )}
              onClick={() =>
                onSelectedGroupChange(isSelected ? undefined : group.name)
              }
            >
              <td className="py-2 pr-2">
                <input
                  type="radio"
                  checked={isSelected}
                  readOnly
                  className="accent-accent"
                />
              </td>
              <td className="py-2 pr-4">
                <div className="flex items-center gap-x-1.5">
                  <FolderTree className="size-4 text-control-light shrink-0" />
                  <span>{extractDatabaseGroupName(group.name)}</span>
                </div>
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}
