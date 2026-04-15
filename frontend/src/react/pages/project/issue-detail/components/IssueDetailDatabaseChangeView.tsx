import { create } from "@bufbuild/protobuf";
import { ChevronRight, ExternalLink, FolderTree, X } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { instanceRoleServiceClientConnect } from "@/connect";
import { EngineIconPath } from "@/react/components/instance/constants";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { SearchInput } from "@/react/components/ui/search-input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/react/components/ui/select";
import { Switch } from "@/react/components/ui/switch";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import {
  PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
} from "@/router/dashboard/projectV1";
import { buildPlanDeployRouteFromPlanName } from "@/router/dashboard/projectV1RouteHelpers";
import {
  getProjectNameAndDatabaseGroupName,
  useDatabaseV1Store,
  useDBGroupStore,
  useEnvironmentV1Store,
  useSheetV1Store,
} from "@/store";
import { isValidDatabaseGroupName, isValidDatabaseName } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import { ListInstanceRolesRequestSchema } from "@/types/proto-es/v1/instance_role_service_pb";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import {
  extractDatabaseResourceName,
  extractInstanceResourceName,
  extractPlanUID,
  extractProjectResourceName,
  getDefaultTransactionMode,
  getInstanceResource,
} from "@/utils";
import { getStatementSize } from "@/utils/sheet";
import { extractDatabaseGroupName } from "@/utils/v1/databaseGroup";
import { instanceV1SupportsTransactionMode } from "@/utils/v1/instance";
import { sheetNameOfSpec } from "@/utils/v1/issue/plan";
import { extractSheetUID, getSheetStatement } from "@/utils/v1/sheet";
import { useIssueDetailContext } from "../context/IssueDetailContext";
import { useIssueDetailSpecValidation } from "../hooks/useIssueDetailSpecValidation";
import {
  allowGhostForDatabase,
  BACKUP_AVAILABLE_ENGINES,
  getGhostConfig,
  type IsolationLevel,
  isDatabaseChangeSpec,
  parseStatement,
} from "../utils/databaseChange";
import { getLocalSheetByName } from "../utils/localSheet";
import { IssueDetailStatementSection } from "./IssueDetailStatementSection";

const DEFAULT_VISIBLE_TARGETS = 20;
const MAX_INLINE_DATABASES = 5;
const EMPTY_SELECT_VALUE = "__empty__";
const EMPTY_SPECS: Plan_Spec[] = [];

export function IssueDetailDatabaseChangeView({
  onSelectedSpecIdChange,
  selectedSpecId,
}: {
  onSelectedSpecIdChange: (specId: string) => void;
  selectedSpecId: string;
}) {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const specs = page.plan?.specs ?? EMPTY_SPECS;
  const { emptySpecIdSet } = useIssueDetailSpecValidation(specs);
  const selectedSpec = useMemo(() => {
    return specs.find((spec) => spec.id === selectedSpecId) ?? specs[0];
  }, [selectedSpecId, specs]);
  const planHref = useMemo(() => {
    if (!page.plan) {
      return "";
    }
    const route = page.plan.hasRollout
      ? buildPlanDeployRouteFromPlanName(page.plan.name)
      : {
          name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
          params: {
            projectId: extractProjectResourceName(page.plan.name),
            planId: extractPlanUID(page.plan.name),
          },
        };
    return router.resolve(route).href;
  }, [page.plan]);

  useEffect(() => {
    if (selectedSpec && selectedSpec.id !== selectedSpecId) {
      onSelectedSpecIdChange(selectedSpec.id);
    }
  }, [onSelectedSpecIdChange, selectedSpec, selectedSpecId]);

  return (
    <div className="flex w-full flex-col">
      <div>
        <div className="flex items-center justify-between border-b border-control-border">
          <div className="flex items-center">
            {specs.map((spec, index) => {
              const isSelected = selectedSpec?.id === spec.id;
              return (
                <button
                  key={spec.id}
                  className={cn(
                    "relative -mb-px flex cursor-pointer items-center gap-1 rounded-t-md border px-3 py-1.5 text-sm transition-colors",
                    isSelected
                      ? "border-control-border border-b-white bg-white font-medium text-main"
                      : "border-transparent bg-transparent text-control-light hover:text-control"
                  )}
                  onClick={() => onSelectedSpecIdChange(spec.id)}
                  type="button"
                >
                  <span className="opacity-60">#{index + 1}</span>
                  <span>{t("plan.spec.type.database-change")}</span>
                  {emptySpecIdSet.has(spec.id) && (
                    <Tooltip content={t("plan.navigator.statement-empty")}>
                      <span className="text-error">*</span>
                    </Tooltip>
                  )}
                </button>
              );
            })}
          </div>

          {page.plan && (
            <a
              className="px-3 text-sm text-accent hover:underline"
              href={planHref}
            >
              {t("plan.go-to-plan-page")} →
            </a>
          )}
        </div>

        <div className="rounded-b-lg border-x border-b border-control-border bg-white px-3 py-2">
          {selectedSpec && (
            <div className="flex flex-col gap-2">
              <IssueDetailDatabaseChangeTargets selectedSpec={selectedSpec} />
              <div className="flex flex-col">
                <IssueDetailStatementSection
                  forceReadonly
                  spec={selectedSpec}
                />
              </div>
              <IssueDetailDatabaseChangeOptions selectedSpec={selectedSpec} />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function IssueDetailDatabaseChangeOptions({
  selectedSpec,
}: {
  selectedSpec: Plan_Spec;
}) {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();
  const sheetStore = useSheetV1Store();
  const [sheetStatement, setSheetStatement] = useState("");
  const [isSheetOversize, setIsSheetOversize] = useState(false);
  const [instanceRoles, setInstanceRoles] = useState<string[]>([]);

  const targets = useMemo(() => {
    if (selectedSpec.config?.case === "changeDatabaseConfig") {
      return selectedSpec.config.value.targets ?? [];
    }
    return [];
  }, [selectedSpec]);
  const databases = useVueState(() =>
    targets
      .map((target) => databaseStore.getDatabaseByName(target))
      .filter((database) => isValidDatabaseName(database.name))
  );
  const parsedStatement = useMemo(
    () => parseStatement(sheetStatement),
    [sheetStatement]
  );
  const selectedRole = useMemo(() => {
    if (!parsedStatement.roleSetterBlock) {
      return "";
    }
    const match = parsedStatement.roleSetterBlock.match(
      /SET ROLE ([a-zA-Z_][a-zA-Z0-9_]{0,62});/
    );
    return match?.[1] ?? "";
  }, [parsedStatement.roleSetterBlock]);
  const selectedIsolation = parsedStatement.isolationLevel ?? "";
  const transactionModeChecked =
    parsedStatement.transactionMode !== undefined
      ? parsedStatement.transactionMode === "on"
      : getDefaultTransactionMode();
  const ghostEnabled = getGhostConfig(sheetStatement) !== undefined;
  const preBackupEnabled =
    selectedSpec.config?.case === "changeDatabaseConfig"
      ? Boolean(selectedSpec.config.value.enablePriorBackup)
      : false;
  const isSheetBasedDatabaseChange =
    selectedSpec.config?.case === "changeDatabaseConfig" &&
    !selectedSpec.config.value.release;
  const showTransactionMode = useMemo(() => {
    if (!isSheetBasedDatabaseChange) {
      return false;
    }
    return databases.every((database) =>
      instanceV1SupportsTransactionMode(getInstanceResource(database).engine)
    );
  }, [databases, isSheetBasedDatabaseChange]);
  const showInstanceRole = useMemo(() => {
    if (!isSheetBasedDatabaseChange) {
      return false;
    }
    return databases.every(
      (database) => getInstanceResource(database).engine === Engine.POSTGRES
    );
  }, [databases, isSheetBasedDatabaseChange]);
  const instanceName = useMemo(() => {
    const database = databases[0];
    if (!database) {
      return "";
    }
    return extractDatabaseResourceName(database.name).instance;
  }, [databases]);
  const showIsolationLevel = useMemo(() => {
    if (!isSheetBasedDatabaseChange) {
      return false;
    }
    return databases.every((database) =>
      [Engine.MYSQL, Engine.MARIADB, Engine.TIDB].includes(
        getInstanceResource(database).engine
      )
    );
  }, [databases, isSheetBasedDatabaseChange]);
  const showPreBackup = useMemo(() => {
    if (selectedSpec.config?.case !== "changeDatabaseConfig") {
      return false;
    }
    if (isDatabaseChangeSpec(selectedSpec)) {
      return databases.every((database) =>
        BACKUP_AVAILABLE_ENGINES.includes(getInstanceResource(database).engine)
      );
    }
    return true;
  }, [databases, selectedSpec]);
  const showGhost = useMemo(() => {
    if (!isSheetBasedDatabaseChange) {
      return false;
    }
    return databases.every((database) => allowGhostForDatabase(database));
  }, [databases, isSheetBasedDatabaseChange]);
  const shouldShow =
    showTransactionMode ||
    showInstanceRole ||
    showIsolationLevel ||
    showPreBackup ||
    showGhost;
  const instanceRoleOptions = useMemo(() => {
    const roles = !selectedRole
      ? instanceRoles
      : instanceRoles.includes(selectedRole)
        ? instanceRoles
        : [selectedRole, ...instanceRoles];
    return roles.map((role) => ({ label: role, value: role }));
  }, [instanceRoles, selectedRole]);
  const oversizeTooltip = isSheetOversize
    ? t("issue.options-disabled-due-to-oversize")
    : undefined;
  const isolationLevelOptions = useMemo(() => {
    return [
      {
        label: t("plan.isolation-level.read-uncommitted"),
        value: "READ_UNCOMMITTED",
      },
      {
        label: t("plan.isolation-level.read-committed"),
        value: "READ_COMMITTED",
      },
      {
        label: t("plan.isolation-level.repeatable-read"),
        value: "REPEATABLE_READ",
      },
      {
        label: t("plan.isolation-level.serializable"),
        value: "SERIALIZABLE",
      },
    ] satisfies Array<{ label: string; value: IsolationLevel }>;
  }, [t]);

  useEffect(() => {
    let canceled = false;
    const sheetName = sheetNameOfSpec(selectedSpec);
    if (!sheetName) {
      setSheetStatement("");
      setIsSheetOversize(false);
      return;
    }

    const loadSheet = async () => {
      const uid = extractSheetUID(sheetName);
      const sheet = uid.startsWith("-")
        ? getLocalSheetByName(sheetName)
        : await sheetStore.getOrFetchSheetByName(sheetName);
      if (!sheet || canceled) {
        return;
      }
      const statement = getSheetStatement(sheet);
      setSheetStatement(statement);
      setIsSheetOversize(getStatementSize(statement) < sheet.contentSize);
    };

    void loadSheet();

    return () => {
      canceled = true;
    };
  }, [selectedSpec, sheetStore]);

  useEffect(() => {
    let canceled = false;
    if (!showInstanceRole || !instanceName) {
      setInstanceRoles((current) => (current.length === 0 ? current : []));
      return;
    }

    const loadInstanceRoles = async () => {
      try {
        const request = create(ListInstanceRolesRequestSchema, {
          parent: instanceName,
        });
        const response =
          await instanceRoleServiceClientConnect.listInstanceRoles(request);
        if (!canceled) {
          setInstanceRoles(response.roles.map((role) => role.roleName));
        }
      } catch {
        if (!canceled) {
          setInstanceRoles([]);
        }
      }
    };

    void loadInstanceRoles();

    return () => {
      canceled = true;
    };
  }, [instanceName, showInstanceRole]);

  return (
    <div className={cn("flex flex-col gap-1", !shouldShow && "hidden")}>
      <span className="text-base">{t("plan.options.self")}</span>
      <div className="flex flex-wrap items-center gap-x-4 gap-y-1 sm:gap-x-6">
        <div
          className={cn(
            "flex items-center gap-1",
            !showInstanceRole && "hidden"
          )}
        >
          <span className="text-sm text-control-light">
            {t("common.role.self")}
          </span>
          <IssueDetailReadonlySelect
            disabledReason={oversizeTooltip}
            options={instanceRoleOptions}
            placeholder={t("instance.select-database-user")}
            value={selectedRole}
            widthClassName="w-36"
          />
        </div>
        <div
          className={cn(
            "flex items-center gap-1",
            !showTransactionMode && "hidden"
          )}
        >
          <span className="text-sm text-control-light">
            {t("issue.transaction-mode.label")}
          </span>
          <Tooltip content={oversizeTooltip}>
            <div>
              <Switch
                checked={transactionModeChecked}
                disabled
                onCheckedChange={() => {}}
                size="small"
              />
            </div>
          </Tooltip>
        </div>
        <div
          className={cn(
            "flex items-center gap-1",
            !showIsolationLevel && "hidden"
          )}
        >
          <span className="text-sm text-control-light">
            {t("plan.isolation-level.self")}
          </span>
          <IssueDetailReadonlySelect
            disabledReason={oversizeTooltip}
            options={isolationLevelOptions}
            placeholder={t("plan.select-isolation-level")}
            value={selectedIsolation}
            widthClassName="w-36"
          />
        </div>
        <div
          className={cn("flex items-center gap-1", !showPreBackup && "hidden")}
        >
          <span className="text-sm text-control-light">
            {t("task.prior-backup")}
          </span>
          <div>
            <Switch
              checked={preBackupEnabled}
              disabled
              onCheckedChange={() => {}}
              size="small"
            />
          </div>
        </div>
        <div className={cn("flex items-center gap-1", !showGhost && "hidden")}>
          <span className="text-sm text-control-light">
            {t("task.online-migration.self")}
          </span>
          <Tooltip content={oversizeTooltip}>
            <div>
              <Switch
                checked={ghostEnabled}
                disabled
                onCheckedChange={() => {}}
                size="small"
              />
            </div>
          </Tooltip>
        </div>
      </div>
    </div>
  );
}

function IssueDetailReadonlySelect({
  disabledReason,
  options,
  placeholder,
  value,
  widthClassName,
}: {
  disabledReason?: string;
  options: Array<{ label: string; value: string }>;
  placeholder: string;
  value: string;
  widthClassName?: string;
}) {
  return (
    <Tooltip content={disabledReason ?? ""}>
      <div>
        <Select
          disabled
          onValueChange={() => {}}
          value={value || EMPTY_SELECT_VALUE}
        >
          <SelectTrigger className={cn("h-6 text-xs", widthClassName)}>
            <SelectValue>
              {options.find((option) => option.value === value)?.label ||
                placeholder}
            </SelectValue>
          </SelectTrigger>
          <SelectContent>
            {!value && (
              <SelectItem value={EMPTY_SELECT_VALUE}>{placeholder}</SelectItem>
            )}
            {options.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </Tooltip>
  );
}

function IssueDetailDatabaseChangeTargets({
  selectedSpec,
}: {
  selectedSpec: Plan_Spec;
}) {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();
  const dbGroupStore = useDBGroupStore();
  const [showAllTargetsDialog, setShowAllTargetsDialog] = useState(false);
  const [searchText, setSearchText] = useState("");
  const [isLoadingTargets, setIsLoadingTargets] = useState(false);
  const [isLoadingAllTargets, setIsLoadingAllTargets] = useState(false);

  const targets = useMemo(() => {
    if (selectedSpec.config?.case === "changeDatabaseConfig") {
      return selectedSpec.config.value.targets ?? [];
    }
    return [];
  }, [selectedSpec]);
  const visibleTargets = useMemo(() => {
    return targets.slice(0, Math.min(DEFAULT_VISIBLE_TARGETS, targets.length));
  }, [targets]);
  const nonEnvDatabaseNames = useVueState(() => {
    if (isLoadingTargets) {
      return [];
    }
    return targets
      .flatMap((target) => {
        if (!isValidDatabaseGroupName(target)) {
          return [target];
        }
        return (
          dbGroupStore
            .getDBGroupByName(target)
            .matchedDatabases?.map((database) => database.name) ?? []
        );
      })
      .filter(
        (name) => !databaseStore.getDatabaseByName(name).effectiveEnvironment
      );
  });
  const filteredTargets = useVueState(() => {
    if (!searchText) {
      return targets;
    }

    const normalizedSearchText = searchText.toLowerCase();
    return targets.filter((target) => {
      if (isValidDatabaseName(target)) {
        const database = databaseStore.getDatabaseByName(target);
        return extractDatabaseResourceName(database.name)
          .databaseName.toLowerCase()
          .includes(normalizedSearchText);
      }
      return String(target).toLowerCase().includes(normalizedSearchText);
    });
  });
  const nonEnvWarning =
    nonEnvDatabaseNames.length === 1
      ? t("plan.targets.non-env-warning.one", {
          count: nonEnvDatabaseNames.length,
        })
      : t("plan.targets.non-env-warning.other", {
          count: nonEnvDatabaseNames.length,
        });

  useEffect(() => {
    let canceled = false;

    const loadVisibleTargets = async () => {
      if (targets.length === 0) {
        setIsLoadingTargets(false);
        return;
      }

      setIsLoadingTargets(true);
      try {
        await fetchTargets(visibleTargets, dbGroupStore, databaseStore);
      } catch {
        // Ignore target loading failures to match the current Vue behavior.
      } finally {
        if (!canceled) {
          setIsLoadingTargets(false);
        }
      }
    };

    void loadVisibleTargets();

    return () => {
      canceled = true;
    };
  }, [databaseStore, dbGroupStore, targets, visibleTargets]);

  useEffect(() => {
    if (!showAllTargetsDialog) {
      return;
    }

    let canceled = false;
    setSearchText("");

    const loadAllTargets = async () => {
      setIsLoadingAllTargets(true);
      try {
        await fetchTargets(targets, dbGroupStore, databaseStore);
      } catch {
        // Ignore target loading failures to match the current Vue behavior.
      } finally {
        if (!canceled) {
          setIsLoadingAllTargets(false);
        }
      }
    };

    void loadAllTargets();

    return () => {
      canceled = true;
    };
  }, [databaseStore, dbGroupStore, showAllTargetsDialog, targets]);

  return (
    <>
      <div className="flex flex-col gap-y-1">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1">
            <span className="textlabel uppercase">
              {t("plan.targets.title")}
            </span>
            {targets.length > 1 && (
              <span className="textlabel text-control-light">
                ({targets.length})
              </span>
            )}
          </div>
        </div>

        {!isLoadingTargets && nonEnvDatabaseNames.length > 0 && (
          <div className="rounded-sm border border-yellow-200 bg-yellow-50 px-3 py-2 text-sm text-yellow-900">
            <div>{nonEnvWarning}</div>
            <div className="mt-1 flex flex-col gap-1 text-sm">
              {nonEnvDatabaseNames.map((name) => (
                <div key={name} className="flex items-center gap-2">
                  <span className="h-1 w-1 shrink-0 rounded-full bg-current" />
                  <IssueDetailDatabaseTarget target={name} />
                </div>
              ))}
            </div>
          </div>
        )}

        {isLoadingTargets ? (
          <div className="flex items-center justify-center py-2">
            <div className="h-5 w-5 animate-spin rounded-full border-2 border-control-border border-t-accent" />
          </div>
        ) : targets.length > 0 ? (
          <div className="flex flex-wrap items-center gap-2">
            {visibleTargets.map((target) => (
              <div
                key={target}
                className="inline-flex cursor-default items-center gap-x-1 rounded-lg border px-2 py-1"
              >
                {isValidDatabaseName(target) ? (
                  <IssueDetailDatabaseTarget showEnvironment target={target} />
                ) : isValidDatabaseGroupName(target) ? (
                  <IssueDetailDatabaseGroupTarget
                    className="py-1"
                    target={target}
                  />
                ) : (
                  <span className="text-sm text-control-placeholder">
                    {target}
                  </span>
                )}
              </div>
            ))}
            {targets.length > DEFAULT_VISIBLE_TARGETS && (
              <button
                className="h-7 cursor-pointer rounded-sm px-2 text-xs text-control transition-colors hover:bg-control-bg"
                onClick={() => setShowAllTargetsDialog(true)}
                type="button"
              >
                {t("plan.targets.view-all", { count: targets.length })}
              </button>
            )}
          </div>
        ) : (
          <div className="py-1 text-sm text-control-light">
            {t("plan.targets.no-targets-found")}
          </div>
        )}
      </div>

      <Dialog
        open={showAllTargetsDialog}
        onOpenChange={(open) => setShowAllTargetsDialog(open)}
      >
        <DialogContent className="relative max-w-[100vw] p-0 md:w-[50rem]">
          <div className="flex h-full max-h-[calc(100vh-10rem)] flex-col gap-y-4">
            <div className="flex items-center justify-between border-b px-4 py-4">
              <DialogTitle>
                {t("plan.targets.title")} ({targets.length})
              </DialogTitle>
              <DialogClose
                render={
                  <button
                    className="inline-flex h-8 w-8 cursor-pointer items-center justify-center rounded-sm text-control transition-colors hover:bg-control-bg"
                    type="button"
                  />
                }
              >
                <X className="h-4 w-4" />
              </DialogClose>
            </div>

            <div className="px-4">
              <SearchInput
                placeholder={t("common.search")}
                value={searchText}
                onChange={(event) => setSearchText(event.target.value)}
              />
            </div>

            <div className="flex-1 overflow-hidden px-4 pb-4">
              {isLoadingAllTargets ? (
                <div className="flex h-full items-center justify-center">
                  <div className="h-5 w-5 animate-spin rounded-full border-2 border-control-border border-t-accent" />
                </div>
              ) : filteredTargets.length > 0 ? (
                <div className="h-full overflow-y-auto">
                  <div className="flex flex-wrap gap-2">
                    {filteredTargets.map((target) => (
                      <div
                        key={target}
                        className="inline-flex cursor-default items-center gap-x-1 rounded-lg border px-2 py-1 transition-all"
                      >
                        {isValidDatabaseName(target) ? (
                          <IssueDetailDatabaseTarget
                            showEnvironment
                            target={target}
                          />
                        ) : isValidDatabaseGroupName(target) ? (
                          <IssueDetailDatabaseGroupTarget target={target} />
                        ) : (
                          <span className="text-sm">{target}</span>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              ) : (
                <div className="flex h-full items-center justify-center text-control-light">
                  {t("common.no-data")}
                </div>
              )}
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}

function IssueDetailDatabaseTarget({
  showEnvironment = false,
  target,
}: {
  showEnvironment?: boolean;
  target: string;
}) {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();
  const environmentStore = useEnvironmentV1Store();
  const database = useVueState(() => databaseStore.getDatabaseByName(target));
  const environment = useVueState(() =>
    environmentStore.getEnvironmentByName(
      database.effectiveEnvironment ??
        database.instanceResource?.environment ??
        ""
    )
  );
  const instance = database.instanceResource;
  const engineIcon = instance ? EngineIconPath[instance.engine] : "";
  const { databaseName } = extractDatabaseResourceName(target);
  const instanceTitle =
    instance?.title ||
    extractInstanceResourceName(target) ||
    t("common.unknown");

  return (
    <div className="flex min-w-0 items-center truncate text-sm">
      {engineIcon && (
        <img alt="" className="mr-1 inline-block h-4 w-4" src={engineIcon} />
      )}
      {showEnvironment && (
        <span className="mr-1 truncate text-gray-400">{environment.title}</span>
      )}
      <span className="truncate text-gray-600">{instanceTitle}</span>
      <ChevronRight className="h-4 w-4 shrink-0 text-gray-500 opacity-60" />
      <span className="truncate text-gray-800">{databaseName}</span>
    </div>
  );
}

function IssueDetailDatabaseGroupTarget({
  className,
  target,
}: {
  className?: string;
  target: string;
}) {
  const { t } = useTranslation();
  const dbGroupStore = useDBGroupStore();
  const databaseStore = useDatabaseV1Store();
  const databaseGroup = useVueState(() =>
    dbGroupStore.getDBGroupByName(target)
  );
  const databases = useMemo(
    () => databaseGroup.matchedDatabases?.map((db) => db.name) ?? [],
    [databaseGroup.matchedDatabases]
  );
  const extraDatabases = databases.slice(MAX_INLINE_DATABASES);

  useEffect(() => {
    if (!isValidDatabaseGroupName(target)) {
      return;
    }
    void dbGroupStore.getOrFetchDBGroupByName(target, {
      silent: true,
      view: DatabaseGroupView.FULL,
    });
  }, [dbGroupStore, target]);

  useEffect(() => {
    if (databases.length > 0) {
      void databaseStore.batchGetOrFetchDatabases(databases);
    }
  }, [databaseStore, databases]);

  const gotoDatabaseGroupDetailPage = () => {
    const [projectId, databaseGroupName] =
      getProjectNameAndDatabaseGroupName(target);
    const url = router.resolve({
      name: PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
      params: {
        databaseGroupName,
        projectId,
      },
    }).fullPath;
    window.open(url, "_blank");
  };

  return (
    <div className={cn("flex w-full flex-col gap-2", className)}>
      <div className="flex items-center gap-x-2">
        <FolderTree className="h-5 w-5 shrink-0 text-control" />
        <span className="inline-flex items-center rounded-full border px-2 py-0.5 text-xs">
          {t("common.database-group")}
        </span>
        <span className="truncate text-sm text-gray-800">
          {extractDatabaseGroupName(databaseGroup.name || target)}
        </span>
        {isValidDatabaseGroupName(databaseGroup.name) && (
          <button
            className="flex cursor-pointer items-center opacity-60 hover:opacity-100"
            onClick={gotoDatabaseGroupDetailPage}
            type="button"
          >
            <ExternalLink className="h-4 w-auto" />
          </button>
        )}
      </div>

      {databases.length > 0 && (
        <div className="flex flex-wrap items-center gap-2 pl-7">
          {databases.slice(0, MAX_INLINE_DATABASES).map((database) => (
            <div
              key={database}
              className="inline-flex cursor-default items-center gap-x-1 rounded-lg border bg-gray-50 px-2 py-1 transition-all"
            >
              <IssueDetailDatabaseTarget showEnvironment target={database} />
            </div>
          ))}
          {extraDatabases.length > 0 && (
            <Tooltip
              content={
                <div className="flex flex-col gap-y-1">
                  {extraDatabases.map((database) => (
                    <span key={database}>{database}</span>
                  ))}
                </div>
              }
              side="bottom"
            >
              <span className="cursor-pointer text-xs text-accent">
                {t("common.n-more", {
                  n: databases.length - MAX_INLINE_DATABASES,
                })}
              </span>
            </Tooltip>
          )}
        </div>
      )}
    </div>
  );
}

const fetchTargets = async (
  targets: string[],
  dbGroupStore: ReturnType<typeof useDBGroupStore>,
  databaseStore: ReturnType<typeof useDatabaseV1Store>
) => {
  const databaseTargets = new Set<string>();

  for (const target of targets) {
    if (isValidDatabaseGroupName(target)) {
      try {
        const databaseGroup = await dbGroupStore.getOrFetchDBGroupByName(
          target,
          {
            silent: true,
            view: DatabaseGroupView.FULL,
          }
        );
        for (const database of databaseGroup.matchedDatabases ?? []) {
          databaseTargets.add(database.name);
        }
      } catch {
        // Ignore target loading failures to match the current Vue behavior.
      }
      continue;
    }
    if (isValidDatabaseName(target)) {
      databaseTargets.add(target);
    }
  }

  if (databaseTargets.size > 0) {
    await databaseStore.batchGetOrFetchDatabases([...databaseTargets]);
  }

  return [...databaseTargets];
};
