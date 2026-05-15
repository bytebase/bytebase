import { clone, create } from "@bufbuild/protobuf";
import {
  CheckCircle,
  ChevronRight,
  DatabaseIcon,
  EllipsisVertical,
  ExternalLink,
  FolderTree,
  Loader2,
  XCircle,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import {
  instanceRoleServiceClientConnect,
  planServiceClientConnect,
} from "@/connect";
import { EngineIcon } from "@/react/components/EngineIcon";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { Alert } from "@/react/components/ui/alert";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/react/components/ui/alert-dialog";
import { Badge } from "@/react/components/ui/badge";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
import { SearchInput } from "@/react/components/ui/search-input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/react/components/ui/select";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { Switch } from "@/react/components/ui/switch";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useSessionPageSize } from "@/react/hooks/useSessionPageSize";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import {
  PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
} from "@/router/dashboard/projectV1";
import {
  getProjectNameAndDatabaseGroupName,
  pushNotification,
  useCurrentUserV1,
  useDatabaseV1Store,
  useDBGroupStore,
  useEnvironmentV1Store,
  useProjectV1Store,
  useSheetV1Store,
} from "@/store";
import {
  isValidDatabaseGroupName,
  isValidDatabaseName,
  isValidReleaseName,
} from "@/types";
import { Engine, State } from "@/types/proto-es/v1/common_pb";
import type { DatabaseGroup } from "@/types/proto-es/v1/database_group_service_pb";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { SyncStatus } from "@/types/proto-es/v1/database_service_pb";
import { ListInstanceRolesRequestSchema } from "@/types/proto-es/v1/instance_role_service_pb";
import {
  Plan_ChangeDatabaseConfigSchema,
  type Plan_Spec,
  Plan_SpecSchema,
  type PlanCheckRun,
  PlanSchema,
  UpdatePlanRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import type { CheckReleaseResponse_CheckResult } from "@/types/proto-es/v1/release_service_pb";
import {
  extractDatabaseResourceName,
  getDatabaseEnvironment,
  getDefaultTransactionMode,
  getInstanceResource,
  hasProjectPermissionV2,
} from "@/utils";
import { getStatementSize } from "@/utils/sheet";
import { extractDatabaseGroupName } from "@/utils/v1/databaseGroup";
import { sheetNameOfSpec } from "@/utils/v1/issue/plan";
import {
  extractSheetUID,
  getSheetStatement,
  setSheetStatement as setLocalSheetStatement,
} from "@/utils/v1/sheet";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import {
  getDefaultGhostConfig,
  getGhostConfig as getGhostConfigFromStatement,
  updateGhostConfig,
  updateIsolationLevel,
  updateRoleSetter,
  updateTransactionMode,
} from "../utils/directiveUtils";
import { getLocalSheetByName, getNextLocalSheetUID } from "../utils/localSheet";
import {
  allowGhostForDatabase,
  getPlanOptionVisibility,
} from "../utils/options";
import {
  planCheckRunListForSpec,
  transformReleaseCheckResultsToPlanCheckRuns,
} from "../utils/planCheck";
import { getSelectedSpec, getSpecTitle } from "../utils/spec";
import { updateSpecSheetWithStatement } from "../utils/specMutation";
import {
  filterPlanTargets,
  getDatabaseGroupRouteParams,
  splitInlineDatabases,
} from "../utils/targets";
import { PlanDetailChecks } from "./PlanDetailChecks";
import { PlanDetailDraftChecks } from "./PlanDetailDraftChecks";
import { PlanDetailStatementSection } from "./PlanDetailStatementSection";
import { PlanDetailTabItem, PlanDetailTabStrip } from "./PlanDetailTabStrip";

const DEFAULT_VISIBLE_TARGETS = 20;
const EMPTY_SELECT_VALUE = "__empty__";

const pushSpecDetailRoute = (
  projectId: string,
  planId: string,
  specId: string
) =>
  router.push({
    name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
    params: { planId, projectId, specId },
  });

type IsolationLevel =
  | "READ_UNCOMMITTED"
  | "READ_COMMITTED"
  | "REPEATABLE_READ"
  | "SERIALIZABLE";

const BACKUP_AVAILABLE_ENGINES = [
  Engine.MYSQL,
  Engine.MARIADB,
  Engine.TIDB,
  Engine.MSSQL,
  Engine.ORACLE,
  Engine.POSTGRES,
];

export function PlanDetailChangesBranch({
  selectedSpecId,
  onSelectedSpecIdChange,
}: {
  selectedSpecId: string;
  onSelectedSpecIdChange: (specId: string) => void;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const { patchState } = page;
  const currentUser = useCurrentUserV1().value;
  const projectStore = useProjectV1Store();
  const project = useVueState(() =>
    projectStore.getProjectByName(`projects/${page.projectId}`)
  );
  const [showAddSpecSheet, setShowAddSpecSheet] = useState(false);
  const [showTargetSelectorSheet, setShowTargetSelectorSheet] = useState(false);
  const [specPendingDelete, setSpecPendingDelete] = useState<Plan_Spec | null>(
    null
  );
  // Locally-held new spec for an already-created plan. We don't persist a new
  // spec to the backend at the "pick targets" step — that would create a spec
  // with an empty SQL statement. The spec lives here until the user fills in
  // a statement and saves; that single save commits the sheet and the spec
  // together via PlanDetailStatementSection's existing save flow.
  const [pendingNewSpec, setPendingNewSpec] = useState<Plan_Spec | null>(null);
  // Whether the draft tab is the active selection. Tracked separately from
  // selectedSpecId (which mirrors the URL) so a draft can be selected even
  // when the URL still points at a real spec.
  const [isPendingSelected, setIsPendingSelected] = useState(false);
  const [draftCheckRunsBySpecId, setDraftCheckRunsBySpecId] = useState<
    Record<string, PlanCheckRun[]>
  >({});
  const [draftCheckResultsBySpecId, setDraftCheckResultsBySpecId] = useState<
    Record<string, CheckReleaseResponse_CheckResult[] | undefined>
  >({});
  const specs = page.plan.specs ?? [];
  // Specs visible in the tab strip — real specs plus any pending draft.
  const visibleSpecs = useMemo(
    () => (pendingNewSpec ? [...specs, pendingNewSpec] : specs),
    [specs, pendingNewSpec]
  );
  const selectedSpec = useMemo(() => {
    if (pendingNewSpec && isPendingSelected) {
      return pendingNewSpec;
    }
    return getSelectedSpec({ selectedSpecId, specs });
  }, [pendingNewSpec, isPendingSelected, selectedSpecId, specs]);
  const canModifySpecs = useMemo(() => {
    if (page.plan.state === State.DELETED) return false;
    if (page.readonly) return false;
    if (page.isCreating) return true;
    return (
      !page.plan.hasRollout &&
      (page.plan.creator === currentUser.name ||
        hasProjectPermissionV2(project, "bb.plans.update"))
    );
  }, [
    currentUser.name,
    page.isCreating,
    page.plan.creator,
    page.plan.hasRollout,
    page.plan.state,
    page.readonly,
    project,
  ]);

  const commitSpecs = useCallback(
    async (nextSpecs: Plan_Spec[]) => {
      const planPatch = clone(PlanSchema, page.plan);
      planPatch.specs = nextSpecs;

      if (page.isCreating) {
        patchState({ plan: planPatch });
        return planPatch;
      }

      const response = await planServiceClientConnect.updatePlan(
        create(UpdatePlanRequestSchema, {
          plan: planPatch,
          updateMask: { paths: ["specs"] },
        })
      );
      patchState({ plan: response });
      return response;
    },
    [page.isCreating, page.plan, patchState]
  );

  const selectSpec = useCallback(
    (specId: string) => {
      if (page.isCreating) {
        // No URL drives the selection during plan creation.
        onSelectedSpecIdChange(specId);
        return;
      }
      // Let the URL change drive selectedSpecId via the sync effect in the
      // parent. Updating local state optimistically would bypass the leave
      // confirm dialog when the navigation is cancelled.
      void pushSpecDetailRoute(page.projectId, page.planId, specId);
    },
    [onSelectedSpecIdChange, page.isCreating, page.planId, page.projectId]
  );

  const handleSpecCreate = async (targets: string[]) => {
    try {
      const sheetUID = getNextLocalSheetUID();
      const localSheet = getLocalSheetByName(
        `${project.name}/sheets/${sheetUID}`
      );
      const spec = create(Plan_SpecSchema, {
        id: uuidv4(),
        config: {
          case: "changeDatabaseConfig",
          value: create(Plan_ChangeDatabaseConfigSchema, {
            sheet: localSheet.name,
            targets,
          }),
        },
      });

      if (page.isCreating) {
        // The whole plan is local during creation; just append to plan.specs.
        await commitSpecs([...page.plan.specs, spec]);
        selectSpec(spec.id);
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.updated"),
        });
        setShowAddSpecSheet(false);
        return;
      }

      // Already-created plan: hold the spec as a draft. The first statement
      // save commits the sheet and the spec together, so we never persist an
      // empty-statement spec to the backend.
      setPendingNewSpec(spec);
      setIsPendingSelected(true);
      setShowAddSpecSheet(false);
      return;
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: String(error),
      });
      return;
    }
  };

  const handleDeleteSpec = async () => {
    const targetSpec = specPendingDelete;
    if (!targetSpec || page.plan.specs.length <= 1) {
      setSpecPendingDelete(null);
      return;
    }
    const nextSpecs = page.plan.specs.filter(
      (spec) => spec.id !== targetSpec.id
    );
    const fallbackSpec = nextSpecs[0];
    try {
      await commitSpecs(nextSpecs);
      if (fallbackSpec) {
        selectSpec(fallbackSpec.id);
      }
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: String(error),
      });
    } finally {
      setSpecPendingDelete(null);
    }
  };

  const handleTargetsUpdate = async (targets: string[]) => {
    if (!selectedSpec) return;
    // Pending draft is local only — update in place without hitting the API.
    if (pendingNewSpec && selectedSpec.id === pendingNewSpec.id) {
      const patched = clone(Plan_SpecSchema, pendingNewSpec);
      if (
        patched.config.case === "changeDatabaseConfig" ||
        patched.config.case === "exportDataConfig"
      ) {
        patched.config.value.targets = targets;
      }
      setPendingNewSpec(patched);
      setShowTargetSelectorSheet(false);
      return;
    }
    const nextSpecs = page.plan.specs.map((spec) => {
      if (spec.id !== selectedSpec.id) return spec;
      const patched = clone(Plan_SpecSchema, spec);
      if (
        patched.config.case === "changeDatabaseConfig" ||
        patched.config.case === "exportDataConfig"
      ) {
        patched.config.value.targets = targets;
      }
      return patched;
    });
    try {
      await commitSpecs(nextSpecs);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
      setShowTargetSelectorSheet(false);
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: String(error),
      });
    }
  };

  useEffect(() => {
    // Skip when the draft is the active selection — the parent's
    // selectedSpecId mirrors the URL, and the draft has no URL yet.
    if (
      selectedSpec &&
      selectedSpec.id !== selectedSpecId &&
      selectedSpec.id !== pendingNewSpec?.id
    ) {
      onSelectedSpecIdChange(selectedSpec.id);
    }
  }, [onSelectedSpecIdChange, pendingNewSpec, selectedSpec, selectedSpecId]);

  // Once the URL (and therefore selectedSpecId) lands on a real spec, the
  // draft is no longer the active selection. We don't clear pending eagerly
  // on tab click so the leave-confirm dialog can intercept and keep the
  // draft visible until the user discards.
  useEffect(() => {
    setIsPendingSelected(false);
  }, [selectedSpecId]);

  // Once the draft spec has been committed (the statement save adds it to
  // page.plan.specs), drop the local pending state so it's no longer rendered
  // as a separate tab. Selection follows the freshly-persisted spec.
  useEffect(() => {
    if (
      pendingNewSpec &&
      page.plan.specs.some((spec) => spec.id === pendingNewSpec.id)
    ) {
      const committedId = pendingNewSpec.id;
      setPendingNewSpec(null);
      setIsPendingSelected(false);
      // Push the URL now that the spec is persisted, so refresh keeps state.
      if (!page.isCreating) {
        void pushSpecDetailRoute(page.projectId, page.planId, committedId);
      }
    }
  }, [
    page.isCreating,
    page.plan.specs,
    page.planId,
    page.projectId,
    pendingNewSpec,
  ]);

  const selectedSpecIdForDraftChecks = selectedSpec?.id;
  const handleDraftCheckResultsChange = useCallback(
    (results: CheckReleaseResponse_CheckResult[] | undefined) => {
      if (!selectedSpecIdForDraftChecks) return;
      setDraftCheckResultsBySpecId((prev) => {
        if (prev[selectedSpecIdForDraftChecks] === results) return prev;
        return {
          ...prev,
          [selectedSpecIdForDraftChecks]: results,
        };
      });
      setDraftCheckRunsBySpecId((prev) => ({
        ...prev,
        [selectedSpecIdForDraftChecks]:
          transformReleaseCheckResultsToPlanCheckRuns(results ?? []),
      }));
    },
    [selectedSpecIdForDraftChecks]
  );

  if (!selectedSpec) {
    return (
      <div className="rounded-md border bg-white px-4 py-3 text-sm text-control-light">
        {t("common.no-data")}
      </div>
    );
  }

  const specHasRelease =
    selectedSpec.config.case === "changeDatabaseConfig" &&
    isValidReleaseName(selectedSpec.config.value.release);
  const currentTargets =
    selectedSpec.config.case === "changeDatabaseConfig"
      ? (selectedSpec.config.value.targets ?? [])
      : selectedSpec.config.case === "exportDataConfig"
        ? (selectedSpec.config.value.targets ?? [])
        : [];

  return (
    <div className="flex w-full flex-1 flex-col">
      <PlanDetailTabStrip
        action={
          canModifySpecs ? (
            <Tooltip
              content={
                pendingNewSpec ? t("plan.add-spec-pending-draft") : undefined
              }
            >
              <Button
                disabled={Boolean(pendingNewSpec)}
                onClick={() => setShowAddSpecSheet(true)}
                size="xs"
                variant="outline"
              >
                {t("plan.add-spec")}
              </Button>
            </Tooltip>
          ) : undefined
        }
      >
        {visibleSpecs.map((spec, index) => {
          const isSelected = selectedSpec.id === spec.id;
          const isPending = pendingNewSpec?.id === spec.id;
          return (
            <PlanDetailTabItem
              key={spec.id}
              action={
                canModifySpecs && visibleSpecs.length > 1 ? (
                  <DropdownMenu>
                    <DropdownMenuTrigger
                      className={cn(
                        "mr-2 inline-flex h-6 w-6 shrink-0 cursor-pointer items-center justify-center rounded-xs text-control-light outline-hidden transition-colors hover:bg-control-bg hover:text-control focus-visible:ring-2 focus-visible:ring-accent"
                      )}
                    >
                      <EllipsisVertical className="size-3.5" />
                    </DropdownMenuTrigger>
                    <DropdownMenuContent>
                      <DropdownMenuItem
                        className="text-error"
                        onClick={() => {
                          if (isPending) {
                            // Drafts only live in local state; clear and
                            // fall back to the first real spec.
                            setPendingNewSpec(null);
                            setIsPendingSelected(false);
                            return;
                          }
                          setSpecPendingDelete(spec);
                        }}
                      >
                        {t("common.delete")}
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                ) : undefined
              }
              onSelect={() => {
                if (isPending) {
                  // Draft has no backend URL — only update local selection.
                  setIsPendingSelected(true);
                  return;
                }
                // Don't clear isPendingSelected here — if a leave-confirm
                // dialog intercepts the navigation, we want the draft to
                // stay visible behind the dialog. The URL-sync effect
                // below clears it once selectedSpecId actually changes.
                selectSpec(spec.id);
              }}
              selected={isSelected}
            >
              <span
                className={cn(
                  "flex items-center gap-1 text-sm font-medium transition-colors",
                  isSelected ? "" : "text-control-light hover:text-control"
                )}
              >
                <span className="opacity-80">{index + 1}.</span>
                <span>{getSpecTitle(spec, t)}</span>
              </span>
            </PlanDetailTabItem>
          );
        })}
      </PlanDetailTabStrip>

      <div className="flex flex-1 flex-col overflow-y-auto px-4 py-4">
        <div className="flex flex-col gap-y-4">
          <TargetsSection
            allowEdit={canModifySpecs}
            onEdit={() => setShowTargetSelectorSheet(true)}
            selectedSpec={selectedSpec}
          />
          <PlanDetailStatementSection
            planCheckRuns={
              page.isCreating
                ? (draftCheckRunsBySpecId[selectedSpec.id] ?? [])
                : planCheckRunListForSpec(page.planCheckRuns, selectedSpec)
            }
            spec={selectedSpec}
          />
          {!specHasRelease &&
            (page.isCreating ? (
              selectedSpec.config.case === "changeDatabaseConfig" ? (
                <PlanDetailDraftChecks
                  key={selectedSpec.id}
                  checkResults={draftCheckResultsBySpecId[selectedSpec.id]}
                  onCheckResultsChange={handleDraftCheckResultsChange}
                  selectedSpec={selectedSpec}
                />
              ) : null
            ) : (
              <PlanDetailChecks selectedSpec={selectedSpec} />
            ))}
          {!specHasRelease && <OptionsSection selectedSpec={selectedSpec} />}
        </div>
      </div>

      <TargetSelectorSheet
        currentTargets={currentTargets}
        onConfirm={handleTargetsUpdate}
        onOpenChange={setShowTargetSelectorSheet}
        open={showTargetSelectorSheet}
        projectName={project.name}
      />

      <TargetSelectorSheet
        currentTargets={[]}
        onConfirm={handleSpecCreate}
        onOpenChange={setShowAddSpecSheet}
        open={showAddSpecSheet}
        projectName={project.name}
        title={t("plan.add-spec")}
      />

      <AlertDialog
        open={specPendingDelete !== null}
        onOpenChange={(open) => {
          if (!open) setSpecPendingDelete(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogTitle>
            {t("bbkit.confirm-button.sure-to-delete")}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t("bbkit.confirm-button.cannot-undo")}
          </AlertDialogDescription>
          <AlertDialogFooter>
            <Button
              variant="outline"
              onClick={() => setSpecPendingDelete(null)}
            >
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              onClick={() => void handleDeleteSpec()}
            >
              {t("common.delete")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

function OptionsSection({ selectedSpec }: { selectedSpec: Plan_Spec }) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const { patchState, refreshState } = page;
  const currentUser = useCurrentUserV1().value;
  const projectStore = useProjectV1Store();
  const project = useVueState(() =>
    projectStore.getProjectByName(`projects/${page.projectId}`)
  );
  const databaseStore = useDatabaseV1Store();
  const dbGroupStore = useDBGroupStore();
  const sheetStore = useSheetV1Store();
  const [sheetStatement, setSheetStatementValue] = useState("");
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
      .flatMap((target) => {
        if (isValidDatabaseName(target)) {
          return [databaseStore.getDatabaseByName(target)];
        }
        if (isValidDatabaseGroupName(target)) {
          const dbGroup = dbGroupStore.getDBGroupByName(
            target,
            DatabaseGroupView.FULL
          );
          return (dbGroup.matchedDatabases ?? []).map((database) =>
            databaseStore.getDatabaseByName(database.name)
          );
        }
        return [];
      })
      .filter((database) => isValidDatabaseName(database.name))
  );
  const firstDatabaseName = databases[0]?.name ?? "";
  const instanceName = firstDatabaseName
    ? extractDatabaseResourceName(firstDatabaseName).instance
    : "";

  const parsed = useMemo(
    () => parseStatement(sheetStatement),
    [sheetStatement]
  );
  const selectedRole = parsed.role ?? "";
  const selectedIsolation = parsed.isolationLevel ?? "";
  const transactionModeChecked =
    parsed.transactionMode !== undefined
      ? parsed.transactionMode === "on"
      : getDefaultTransactionMode();
  const ghostEnabled = parsed.ghostEnabled;
  const preBackupEnabled =
    selectedSpec.config?.case === "changeDatabaseConfig"
      ? Boolean(selectedSpec.config.value.enablePriorBackup)
      : false;
  const isSheetBasedDatabaseChange =
    selectedSpec.config?.case === "changeDatabaseConfig" &&
    !selectedSpec.config.value.release;

  const {
    shouldShow,
    showGhost,
    showInstanceRole,
    showIsolationLevel,
    showPreBackup,
    showTransactionMode,
  } = useMemo(
    () =>
      getPlanOptionVisibility({
        databases,
        isChangeDatabaseConfig:
          selectedSpec.config?.case === "changeDatabaseConfig",
        isSheetBasedDatabaseChange,
      }),
    [databases, isSheetBasedDatabaseChange, selectedSpec.config?.case]
  );
  const allowChange = useMemo(() => {
    if (page.readonly) return false;
    if (page.plan.hasRollout) return false;
    if (page.isCreating) return true;
    return (
      page.plan.creator === currentUser.name ||
      hasProjectPermissionV2(project, "bb.plans.update")
    );
  }, [
    currentUser.name,
    page.isCreating,
    page.plan.creator,
    page.plan.hasRollout,
    page.readonly,
    project,
  ]);
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

  useEffect(() => {
    if (targets.length === 0) {
      return;
    }
    void fetchTargets(targets, dbGroupStore, databaseStore);
  }, [databaseStore, dbGroupStore, targets]);

  useEffect(() => {
    let canceled = false;
    const sheetName = sheetNameOfSpec(selectedSpec);
    if (!sheetName) {
      setSheetStatementValue("");
      setIsSheetOversize(false);
      return;
    }
    const load = async () => {
      const uid = extractSheetUID(sheetName);
      const sheet = uid.startsWith("-")
        ? getLocalSheetByName(sheetName)
        : await sheetStore.getOrFetchSheetByName(sheetName);
      if (!sheet || canceled) return;
      const statement = getSheetStatement(sheet);
      setSheetStatementValue(statement);
      setIsSheetOversize(getStatementSize(statement) < sheet.contentSize);
    };
    void load();
    return () => {
      canceled = true;
    };
  }, [selectedSpec, sheetStore]);

  useEffect(() => {
    let canceled = false;
    if (!showInstanceRole || !instanceName) {
      setInstanceRoles((prev) => (prev.length === 0 ? prev : []));
      return;
    }
    const load = async () => {
      try {
        const response =
          await instanceRoleServiceClientConnect.listInstanceRoles(
            create(ListInstanceRolesRequestSchema, { parent: instanceName })
          );
        if (!canceled) {
          setInstanceRoles(response.roles.map((role) => role.roleName));
        }
      } catch {
        if (!canceled) setInstanceRoles([]);
      }
    };
    void load();
    return () => {
      canceled = true;
    };
  }, [instanceName, showInstanceRole]);

  const persistStatement = useCallback(
    async (nextStatement: string) => {
      const sheetName = sheetNameOfSpec(selectedSpec);
      if (!sheetName) return;

      if (page.isCreating) {
        const sheet = getLocalSheetByName(sheetName);
        setLocalSheetStatement(sheet, nextStatement);
        setSheetStatementState(nextStatement);
        return;
      }

      await updateSpecSheetWithStatement(
        page.plan,
        selectedSpec,
        nextStatement
      );
      setSheetStatementState(nextStatement);
      await refreshState();
    },
    [page.isCreating, page.plan, refreshState, selectedSpec]
  );

  const setSheetStatementState = (statement: string) => {
    setSheetStatementValue(statement);
    setIsSheetOversize(false);
  };

  const updatePreBackup = async (enabled: boolean) => {
    if (selectedSpec.config.case !== "changeDatabaseConfig") return;
    const planPatch = clone(PlanSchema, page.plan);
    const spec = planPatch.specs.find((item) => item.id === selectedSpec.id);
    if (!spec || spec.config.case !== "changeDatabaseConfig") return;
    spec.config.value.enablePriorBackup = enabled;

    if (page.isCreating) {
      patchState({ plan: planPatch });
      return;
    }

    const response = await planServiceClientConnect.updatePlan(
      create(UpdatePlanRequestSchema, {
        plan: planPatch,
        updateMask: { paths: ["specs"] },
      })
    );
    patchState({ plan: response });
  };

  const currentGhostConfig = getGhostConfigFromStatement(sheetStatement);
  const ghostIssueDatabases = useMemo(() => {
    return databases.filter((db) => {
      const instance = getInstanceResource(db);
      return (
        !allowGhostForDatabase(instance.engine, instance.engineVersion) ||
        !db.backupAvailable
      );
    });
  }, [databases]);
  const preBackupIssueDatabases = useMemo(() => {
    return databases.filter((db) => {
      const instance = getInstanceResource(db);
      return (
        !BACKUP_AVAILABLE_ENGINES.includes(instance.engine) ||
        !db.backupAvailable
      );
    });
  }, [databases]);
  const ghostTooltip = !allowChange
    ? t("common.read-only")
    : isSheetOversize
      ? t("issue.options-disabled-due-to-oversize")
      : !ghostEnabled && ghostIssueDatabases.length > 0
        ? t("plan.ghost.requirements-not-met", {
            databases: ghostIssueDatabases
              .map((db) => extractDatabaseResourceName(db.name).databaseName)
              .join(", "),
          })
        : undefined;
  const preBackupTooltip = !allowChange
    ? t("common.read-only")
    : !preBackupEnabled && preBackupIssueDatabases.length > 0
      ? t("plan.pre-backup.requirements-not-met", {
          databases: preBackupIssueDatabases
            .map((db) => extractDatabaseResourceName(db.name).databaseName)
            .join(", "),
        })
      : undefined;

  return (
    <div className="flex flex-col gap-y-2">
      <h3 className="textlabel uppercase">{t("plan.options.self")}</h3>
      {shouldShow ? (
        <div className="flex flex-wrap items-center gap-x-6 gap-y-2">
          <div
            className={cn(
              "flex min-h-8 items-center gap-2",
              !showInstanceRole && "hidden"
            )}
          >
            <span className="inline-flex h-8 shrink-0 items-center whitespace-nowrap text-sm text-control-placeholder">
              {t("common.role.self")}
            </span>
            <Tooltip
              content={
                (!allowChange ? t("common.read-only") : oversizeTooltip) ?? ""
              }
            >
              <div>
                <Select
                  disabled={!allowChange || isSheetOversize}
                  onValueChange={(value) => {
                    void persistStatement(
                      updateRoleSetter(sheetStatement, value || undefined)
                    );
                  }}
                  value={selectedRole}
                >
                  <SelectTrigger className="w-44" size="sm">
                    <SelectValue>
                      {instanceRoleOptions.find(
                        (option) => option.value === selectedRole
                      )?.label || t("instance.select-database-user")}
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    {instanceRoleOptions.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </Tooltip>
          </div>
          <div
            className={cn(
              "flex min-h-8 items-center gap-2",
              !showTransactionMode && "hidden"
            )}
          >
            <span className="inline-flex h-8 shrink-0 items-center whitespace-nowrap text-sm text-control-placeholder">
              {t("issue.transaction-mode.label")}
            </span>
            <Tooltip content={oversizeTooltip}>
              <div>
                <Switch
                  checked={transactionModeChecked}
                  disabled={!allowChange || isSheetOversize}
                  onCheckedChange={(checked) => {
                    void persistStatement(
                      updateTransactionMode(
                        sheetStatement,
                        checked ? "on" : "off"
                      )
                    );
                  }}
                  size="sm"
                />
              </div>
            </Tooltip>
          </div>
          <div
            className={cn(
              "flex min-h-8 items-center gap-2",
              !showIsolationLevel && "hidden"
            )}
          >
            <span className="inline-flex h-8 shrink-0 items-center whitespace-nowrap text-sm text-control-placeholder">
              {t("plan.isolation-level.self")}
            </span>
            <Tooltip
              content={
                (!allowChange ? t("common.read-only") : oversizeTooltip) ?? ""
              }
            >
              <div>
                <Select
                  disabled={!allowChange || isSheetOversize}
                  onValueChange={(value) => {
                    void persistStatement(
                      updateIsolationLevel(
                        sheetStatement,
                        value === EMPTY_SELECT_VALUE
                          ? undefined
                          : (value as IsolationLevel) || undefined
                      )
                    );
                  }}
                  value={selectedIsolation || EMPTY_SELECT_VALUE}
                >
                  <SelectTrigger className="w-44" size="sm">
                    <SelectValue>
                      {[
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
                      ].find((option) => option.value === selectedIsolation)
                        ?.label || t("plan.select-isolation-level")}
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={EMPTY_SELECT_VALUE}>
                      {t("plan.select-isolation-level")}
                    </SelectItem>
                    <SelectItem value="READ_UNCOMMITTED">
                      {t("plan.isolation-level.read-uncommitted")}
                    </SelectItem>
                    <SelectItem value="READ_COMMITTED">
                      {t("plan.isolation-level.read-committed")}
                    </SelectItem>
                    <SelectItem value="REPEATABLE_READ">
                      {t("plan.isolation-level.repeatable-read")}
                    </SelectItem>
                    <SelectItem value="SERIALIZABLE">
                      {t("plan.isolation-level.serializable")}
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </Tooltip>
          </div>
          <div
            className={cn(
              "flex min-h-8 items-center gap-2",
              !showPreBackup && "hidden"
            )}
          >
            <span className="inline-flex h-8 shrink-0 items-center whitespace-nowrap text-sm text-control-placeholder">
              {t("task.prior-backup")}
            </span>
            <Tooltip content={preBackupTooltip ?? ""}>
              <div>
                <Switch
                  checked={preBackupEnabled}
                  disabled={
                    !allowChange ||
                    (!preBackupEnabled && preBackupIssueDatabases.length > 0)
                  }
                  onCheckedChange={(checked) => {
                    void updatePreBackup(checked);
                  }}
                  size="sm"
                />
              </div>
            </Tooltip>
          </div>
          <div
            className={cn(
              "flex min-h-8 items-center gap-2",
              !showGhost && "hidden"
            )}
          >
            <span className="inline-flex h-8 shrink-0 items-center whitespace-nowrap text-sm text-control-placeholder">
              {t("task.online-migration.self")}
            </span>
            <Tooltip content={ghostTooltip ?? ""}>
              <div>
                <Switch
                  checked={ghostEnabled}
                  disabled={
                    !allowChange ||
                    isSheetOversize ||
                    (!ghostEnabled && ghostIssueDatabases.length > 0)
                  }
                  onCheckedChange={(checked) => {
                    void persistStatement(
                      updateGhostConfig(
                        sheetStatement,
                        checked
                          ? (currentGhostConfig ?? getDefaultGhostConfig())
                          : undefined
                      )
                    );
                  }}
                  size="sm"
                />
              </div>
            </Tooltip>
          </div>
        </div>
      ) : (
        <div className="text-sm text-control-light">
          {t("plan.options.no-options-available")}
        </div>
      )}
    </div>
  );
}

function TargetsSection({
  allowEdit,
  onEdit,
  selectedSpec,
}: {
  allowEdit: boolean;
  onEdit: () => void;
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
  const visibleTargets = useMemo(
    () => targets.slice(0, Math.min(DEFAULT_VISIBLE_TARGETS, targets.length)),
    [targets]
  );
  const nonEnvDatabaseNames = useMemo(() => {
    if (isLoadingTargets) return [];
    return targets
      .flatMap((target) => {
        if (!isValidDatabaseGroupName(target)) return [target];
        return (
          dbGroupStore
            .getDBGroupByName(target)
            .matchedDatabases?.map((database) => database.name) ?? []
        );
      })
      .filter(
        (name) => !databaseStore.getDatabaseByName(name).effectiveEnvironment
      );
  }, [databaseStore, dbGroupStore, isLoadingTargets, targets]);
  const filteredTargets = useMemo(
    () =>
      filterPlanTargets({
        getDatabaseDisplayName: (target) => {
          if (!isValidDatabaseName(target)) return target;
          const database = databaseStore.getDatabaseByName(target);
          return extractDatabaseResourceName(database.name).databaseName;
        },
        query: searchText,
        targets,
      }),
    [databaseStore, searchText, targets]
  );
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
    const load = async () => {
      if (targets.length === 0) {
        setIsLoadingTargets(false);
        return;
      }
      setIsLoadingTargets(true);
      try {
        await fetchTargets(visibleTargets, dbGroupStore, databaseStore);
      } finally {
        if (!canceled) setIsLoadingTargets(false);
      }
    };
    void load();
    return () => {
      canceled = true;
    };
  }, [databaseStore, dbGroupStore, targets, visibleTargets]);

  useEffect(() => {
    if (!showAllTargetsDialog) return;
    let canceled = false;
    setSearchText("");
    const load = async () => {
      setIsLoadingAllTargets(true);
      try {
        await fetchTargets(targets, dbGroupStore, databaseStore);
      } finally {
        if (!canceled) setIsLoadingAllTargets(false);
      }
    };
    void load();
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
          {allowEdit && (
            <Button onClick={onEdit} size="xs" variant="outline">
              {t("common.edit")}
            </Button>
          )}
        </div>
        {!isLoadingTargets && nonEnvDatabaseNames.length > 0 && (
          <Alert
            className="px-3 py-2"
            variant="warning"
            description={
              <>
                <div>{nonEnvWarning}</div>
                <div className="mt-1 flex flex-col gap-1 text-sm">
                  {nonEnvDatabaseNames.map((name) => (
                    <div key={name} className="flex items-center gap-2">
                      <span className="h-1 w-1 shrink-0 rounded-full bg-current" />
                      <DatabaseTarget target={name} />
                    </div>
                  ))}
                </div>
              </>
            }
          />
        )}
        {isLoadingTargets ? (
          <div className="flex items-center justify-center py-2">
            <div className="h-5 w-5 animate-spin rounded-full border-2 border-control-border border-t-accent" />
          </div>
        ) : targets.length > 0 ? (
          <div className="flex flex-wrap items-center gap-2">
            {visibleTargets.map((target) =>
              isValidDatabaseName(target) ? (
                <div
                  key={target}
                  className="inline-flex cursor-default items-center gap-x-1 rounded-lg border px-2 py-1"
                >
                  <DatabaseTarget showEnvironment target={target} />
                </div>
              ) : isValidDatabaseGroupName(target) ? (
                <div key={target} className="rounded-lg border px-2 py-1">
                  <DatabaseGroupTarget className="py-1" target={target} />
                </div>
              ) : (
                <div
                  key={target}
                  className="inline-flex cursor-default items-center gap-x-1 rounded-lg border px-2 py-1"
                >
                  <span className="text-sm text-control-placeholder">
                    {target}
                  </span>
                </div>
              )
            )}
            {targets.length > DEFAULT_VISIBLE_TARGETS && (
              <Button
                onClick={() => setShowAllTargetsDialog(true)}
                size="sm"
                type="button"
                variant="ghost"
              >
                {t("plan.targets.view-all", { count: targets.length })}
              </Button>
            )}
          </div>
        ) : (
          <div className="py-1 text-sm text-control-light">
            {t("plan.targets.no-targets-found")}
          </div>
        )}
      </div>

      <Sheet open={showAllTargetsDialog} onOpenChange={setShowAllTargetsDialog}>
        <SheetContent width="wide">
          <SheetHeader>
            <SheetTitle>
              {t("plan.targets.title")} ({targets.length})
            </SheetTitle>
          </SheetHeader>
          <SheetBody className="gap-y-4 px-4 pb-4">
            <SearchInput
              placeholder={t("common.search")}
              value={searchText}
              onChange={(event) => setSearchText(event.target.value)}
            />
            <div className="flex-1 overflow-hidden">
              {isLoadingAllTargets ? (
                <div className="flex h-full items-center justify-center">
                  <div className="h-5 w-5 animate-spin rounded-full border-2 border-control-border border-t-accent" />
                </div>
              ) : filteredTargets.length > 0 ? (
                <div className="h-full overflow-y-auto">
                  <div className="flex flex-wrap gap-2">
                    {filteredTargets.map((target) =>
                      isValidDatabaseName(target) ? (
                        <div
                          key={target}
                          className="inline-flex cursor-default items-center gap-x-1 rounded-lg border px-2 py-1 transition-all"
                        >
                          <DatabaseTarget showEnvironment target={target} />
                        </div>
                      ) : isValidDatabaseGroupName(target) ? (
                        <div
                          key={target}
                          className="rounded-lg border px-2 py-1 transition-all"
                        >
                          <DatabaseGroupTarget target={target} />
                        </div>
                      ) : (
                        <div
                          key={target}
                          className="inline-flex cursor-default items-center gap-x-1 rounded-lg border px-2 py-1 transition-all"
                        >
                          <span className="text-sm">{target}</span>
                        </div>
                      )
                    )}
                  </div>
                </div>
              ) : (
                <div className="flex h-full items-center justify-center text-control-light">
                  {t("common.no-data")}
                </div>
              )}
            </div>
          </SheetBody>
        </SheetContent>
      </Sheet>
    </>
  );
}

function TargetSelectorSheet({
  currentTargets,
  onConfirm,
  onOpenChange,
  open,
  projectName,
  title,
}: {
  currentTargets: string[];
  onConfirm: (targets: string[]) => void | Promise<void>;
  onOpenChange: (open: boolean) => void;
  open: boolean;
  projectName: string;
  title?: string;
}) {
  const { t } = useTranslation();
  const [changeSource, setChangeSource] = useState<"DATABASE" | "GROUP">(
    "DATABASE"
  );
  const [selectedDatabaseNames, setSelectedDatabaseNames] = useState<
    Set<string>
  >(new Set());
  const [selectedDatabaseGroup, setSelectedDatabaseGroup] = useState<
    string | undefined
  >();
  const currentTargetsKey = currentTargets.join("\0");

  useEffect(() => {
    if (!open) {
      return;
    }
    const targets = currentTargetsKey ? currentTargetsKey.split("\0") : [];
    const firstTarget = targets[0];
    if (firstTarget && isValidDatabaseGroupName(firstTarget)) {
      setChangeSource("GROUP");
      setSelectedDatabaseGroup(firstTarget);
      setSelectedDatabaseNames(new Set());
      return;
    }
    setChangeSource("DATABASE");
    setSelectedDatabaseNames(new Set(targets));
    setSelectedDatabaseGroup(undefined);
  }, [currentTargetsKey, open]);

  const canSubmit =
    changeSource === "DATABASE"
      ? selectedDatabaseNames.size > 0
      : Boolean(selectedDatabaseGroup);

  return (
    <Sheet onOpenChange={onOpenChange} open={open}>
      <SheetContent className="w-screen sm:w-[80vw]" width="wide">
        <SheetHeader>
          <SheetTitle>{title ?? t("plan.select-targets")}</SheetTitle>
        </SheetHeader>
        <SheetBody>
          <DatabaseAndGroupSelector
            changeSource={changeSource}
            onChangeSourceChange={setChangeSource}
            onSelectedDatabaseGroupChange={setSelectedDatabaseGroup}
            onSelectedDatabaseNamesChange={setSelectedDatabaseNames}
            projectName={projectName}
            selectedDatabaseGroup={selectedDatabaseGroup}
            selectedDatabaseNames={selectedDatabaseNames}
          />
        </SheetBody>
        <SheetFooter>
          <Button onClick={() => onOpenChange(false)} variant="ghost">
            {t("common.cancel")}
          </Button>
          <Button
            disabled={!canSubmit}
            onClick={() =>
              void onConfirm(
                changeSource === "DATABASE"
                  ? [...selectedDatabaseNames]
                  : selectedDatabaseGroup
                    ? [selectedDatabaseGroup]
                    : []
              )
            }
          >
            {t("common.confirm")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

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
      <div className="flex border-b border-control-border">
        <button
          type="button"
          className={cn(
            "border-b-2 -mb-px px-4 py-2 text-sm font-medium transition-colors",
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
            "border-b-2 -mb-px px-4 py-2 text-sm font-medium transition-colors",
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
          onSelectedNamesChange={onSelectedDatabaseNamesChange}
          projectName={projectName}
          selectedNames={selectedDatabaseNames}
        />
      ) : (
        <DatabaseGroupSelector
          onSelectedGroupChange={onSelectedDatabaseGroupChange}
          projectName={projectName}
          selectedGroup={selectedDatabaseGroup}
        />
      )}
    </div>
  );
}

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
  const [pageSize] = useSessionPageSize("bb.plan-detail-db-selector");
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
        const result = await databaseStore.fetchDatabases({
          parent: projectName,
          pageSize,
          pageToken: token || undefined,
          filter: { query },
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
    [databaseStore, pageSize, projectName, query]
  );

  useEffect(() => {
    const timer = setTimeout(() => void doFetch(true), 300);
    return () => clearTimeout(timer);
  }, [doFetch]);

  const toggleDatabase = (name: string) => {
    const next = new Set(selectedNames);
    if (next.has(name)) next.delete(name);
    else next.add(name);
    onSelectedNamesChange(next);
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
        onChange={(event) => setQuery(event.target.value)}
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
                <th className="w-8 py-2 pr-2">
                  <Checkbox
                    checked={someSelected ? "indeterminate" : allSelected}
                    onCheckedChange={() =>
                      onSelectedNamesChange(
                        allSelected
                          ? new Set()
                          : new Set(databases.map((db) => db.name))
                      )
                    }
                  />
                </th>
                <th className="py-2 pr-4 font-medium">
                  {t("common.database")}
                </th>
                <th className="py-2 pr-4 font-medium">
                  {t("common.instance")}
                </th>
                <th className="py-2 pr-4 font-medium">
                  {t("common.environment")}
                </th>
                <th className="py-2 pr-4 font-medium whitespace-nowrap">
                  {t("common.status")}
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
                      "cursor-pointer border-b hover:bg-control-bg",
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
                    <td className="py-2 pr-4">{inst?.title}</td>
                    <td className="py-2 pr-4">
                      {env && <EnvironmentLabel environmentName={env.name} />}
                    </td>
                    <td className="py-2 pr-4">
                      {db.syncStatus === SyncStatus.FAILED ? (
                        <XCircle className="size-4 text-error" />
                      ) : (
                        <CheckCircle className="size-4 text-success" />
                      )}
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
                onClick={() => void doFetch(false)}
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
      .then((result) => setGroups(result))
      .finally(() => setLoading(false));
  }, [dbGroupStore, projectName]);

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
          <th className="w-8 py-2 pr-2" />
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
                "cursor-pointer border-b hover:bg-control-bg",
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
                  <FolderTree className="size-4 shrink-0 text-control-light" />
                  <span>{group.title}</span>
                </div>
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}

export function DatabaseTarget({
  showEnvironment = false,
  target,
}: {
  showEnvironment?: boolean;
  target: string;
}) {
  const environmentStore = useEnvironmentV1Store();
  const databaseStore = useDatabaseV1Store();
  const database = databaseStore.getDatabaseByName(target);
  const environment = database.effectiveEnvironment
    ? environmentStore.getEnvironmentByName(database.effectiveEnvironment)
    : undefined;
  const instance = getInstanceResource(database);
  const databaseName = extractDatabaseResourceName(database.name).databaseName;
  const instanceName = instance.title;

  return (
    <div className="flex min-w-0 items-center truncate text-sm">
      <EngineIcon engine={instance.engine} className="mr-1 h-4 w-4" />
      {showEnvironment && environment?.title && (
        <span className="mr-1 truncate text-control-placeholder">
          {environment.title}
        </span>
      )}
      <span className="truncate text-control-light">{instanceName}</span>
      <ChevronRight className="h-4 w-4 shrink-0 text-control-light/80" />
      <span className="truncate text-control">{databaseName}</span>
    </div>
  );
}

export function DatabaseGroupTarget({
  className,
  target,
}: {
  className?: string;
  target: string;
}) {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();
  const dbGroupStore = useDBGroupStore();
  const dbGroup = dbGroupStore.getDBGroupByName(target, DatabaseGroupView.FULL);
  const matchedDatabases = dbGroup.matchedDatabases ?? [];
  const { extraDatabases, inlineDatabases } =
    splitInlineDatabases(matchedDatabases);
  const groupName = extractDatabaseGroupName(target);

  useEffect(() => {
    const load = async () => {
      const group = await dbGroupStore.getOrFetchDBGroupByName(target, {
        view: DatabaseGroupView.FULL,
        silent: true,
      });
      const databaseNames =
        group.matchedDatabases?.map((database) => database.name) ?? [];
      if (databaseNames.length > 0) {
        await databaseStore.batchGetOrFetchDatabases(databaseNames);
      }
    };
    void load();
  }, [databaseStore, dbGroupStore, target]);

  const route = {
    name: PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
    params: getDatabaseGroupRouteParams({
      databaseGroupName: getProjectNameAndDatabaseGroupName(target)[1],
      projectName: getProjectNameAndDatabaseGroupName(target)[0],
    }),
  };
  const showExternalLink = isValidDatabaseGroupName(dbGroup.name);

  return (
    <div className={cn("flex w-full flex-col gap-2", className)}>
      <div className="flex items-center gap-x-2">
        <FolderTree className="h-5 w-5 shrink-0 text-control" />
        <Badge className="px-2 py-0 text-xs font-medium" variant="default">
          {t("common.database-group")}
        </Badge>
        <span className="min-w-0 truncate text-sm text-control">
          {groupName}
        </span>
        {showExternalLink && (
          <a
            className="flex items-center opacity-60 hover:opacity-100"
            href={router.resolve(route).href}
            rel="noreferrer"
            target="_blank"
          >
            <ExternalLink className="h-4 w-4" />
          </a>
        )}
      </div>
      {matchedDatabases.length > 0 && (
        <div className="flex flex-wrap items-center gap-2 pl-7">
          {inlineDatabases.map((database) => (
            <div
              key={database.name}
              className="inline-flex cursor-default items-center gap-x-1 rounded-lg border bg-gray-50 px-2 py-1 transition-all"
            >
              <DatabaseTarget showEnvironment target={database.name} />
            </div>
          ))}
          {extraDatabases.length > 0 && (
            <Tooltip
              content={
                <div className="flex max-h-64 flex-col gap-y-1 overflow-y-auto py-1">
                  {extraDatabases.map((database) => (
                    <div key={database.name} className="py-1">
                      <DatabaseTarget showEnvironment target={database.name} />
                    </div>
                  ))}
                </div>
              }
            >
              <span className="cursor-pointer text-xs text-accent">
                {t("common.n-more", { n: extraDatabases.length })}
              </span>
            </Tooltip>
          )}
        </div>
      )}
    </div>
  );
}

function parseStatement(statement: string): {
  transactionMode?: "on" | "off";
  isolationLevel?: IsolationLevel;
  ghostEnabled: boolean;
  role?: string;
} {
  const transactionMode = statement
    .match(/^\s*--\s*txn-mode\s*=\s*(on|off)\s*$/im)?.[1]
    ?.toLowerCase() as "on" | "off" | undefined;
  const isolationLevel = statement
    .match(
      /^\s*--\s*txn-isolation\s*=\s*(READ\s+UNCOMMITTED|READ\s+COMMITTED|REPEATABLE\s+READ|SERIALIZABLE)\s*$/im
    )?.[1]
    ?.toUpperCase()
    .replace(/\s+/g, "_") as IsolationLevel | undefined;
  const ghostEnabled =
    /^\s*--\s*gh-ost\s*=\s*(\{[^}]*\})\s*(?:\/\*.*\*\/)?\s*$/im.test(statement);
  const role = statement.match(
    /\/\*\s*=== Bytebase Role Setter\. DO NOT EDIT\. === \*\/\s*SET ROLE ([a-zA-Z_][a-zA-Z0-9_]{0,62});/
  )?.[1];
  return {
    ghostEnabled,
    isolationLevel,
    role,
    transactionMode,
  };
}

async function fetchTargets(
  targets: string[],
  dbGroupStore: ReturnType<typeof useDBGroupStore>,
  databaseStore: ReturnType<typeof useDatabaseV1Store>
) {
  await Promise.all(
    targets.map(async (target) => {
      if (isValidDatabaseName(target)) {
        await databaseStore.getOrFetchDatabaseByName(target, true);
        return;
      }
      if (isValidDatabaseGroupName(target)) {
        const dbGroup = await dbGroupStore.getOrFetchDBGroupByName(target, {
          silent: true,
          view: DatabaseGroupView.FULL,
        });
        const databaseNames =
          dbGroup.matchedDatabases?.map((database) => database.name) ?? [];
        if (databaseNames.length > 0) {
          await databaseStore.batchGetOrFetchDatabases(databaseNames);
        }
      }
    })
  );
}
