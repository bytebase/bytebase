import { create as createProto } from "@bufbuild/protobuf";
import {
  CheckCircle,
  Clock4,
  Database as DatabaseIcon,
  EllipsisVertical,
  FolderTree,
  Link2,
  Loader2,
  XCircle,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import { EngineIconPath } from "@/components/InstanceForm/constants";
import {
  planServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/connect";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { HumanizeTs } from "@/react/components/HumanizeTs";
import { ReleaseFileDetailPanel } from "@/react/components/release/ReleaseFileDetailPanel";
import { ReleaseFileTable } from "@/react/components/release/ReleaseFileTable";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/react/components/ui/alert-dialog";
import { Button } from "@/react/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
import { SearchInput } from "@/react/components/ui/search-input";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useSessionPageSize } from "@/react/hooks/useSessionPageSize";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import { buildPlanDeployRoute } from "@/router/dashboard/projectV1RouteHelpers";
import {
  pushNotification,
  useDatabaseV1Store,
  useDBGroupStore,
  useProjectV1Store,
  useReleaseStore,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { getTimeForPbTimestampProtoEs } from "@/types";
import { State, VCSType } from "@/types/proto-es/v1/common_pb";
import type { DatabaseGroup } from "@/types/proto-es/v1/database_group_service_pb";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import {
  type Database,
  SyncStatus,
} from "@/types/proto-es/v1/database_service_pb";
import {
  Plan_ChangeDatabaseConfigSchema,
  Plan_SpecSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import type { Release_File } from "@/types/proto-es/v1/release_service_pb";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractDatabaseGroupName,
  extractDatabaseResourceName,
  getDatabaseEnvironment,
  getInstanceResource,
  hasPermissionToCreateChangeDatabaseIssueInProject,
  setDocumentTitle,
} from "@/utils";

export function ProjectReleaseDetailPage({
  projectId,
  releaseId,
}: {
  projectId: string;
  releaseId: string;
}) {
  const { t } = useTranslation();
  const releaseStore = useReleaseStore();
  const projectV1Store = useProjectV1Store();

  const projectName = `${projectNamePrefix}${projectId}`;
  const releaseName = `${projectName}/releases/${releaseId}`;

  const release = useVueState(() => releaseStore.getReleaseByName(releaseName));
  const project = useVueState(() =>
    projectV1Store.getProjectByName(projectName)
  );

  useEffect(() => {
    let cancelled = false;
    void projectV1Store.getOrFetchProjectByName(projectName).catch((error) => {
      if (!cancelled) console.error("Failed to fetch project", error);
    });
    void releaseStore.fetchReleaseByName(releaseName).catch((error) => {
      if (!cancelled) console.error("Failed to fetch release", error);
    });
    return () => {
      cancelled = true;
    };
  }, [projectV1Store, releaseStore, projectName, releaseName]);

  useEffect(() => {
    if (project?.title) {
      setDocumentTitle(t("release.releases"), project.title);
    }
  }, [project?.title, t]);

  const [selectedReleaseFile, setSelectedReleaseFile] = useState<
    Release_File | undefined
  >();
  const [applyOpen, setApplyOpen] = useState(false);
  const [abandonOpen, setAbandonOpen] = useState(false);

  const releaseDisplayName = useMemo(() => {
    const parts = release.name.split("/");
    return parts[parts.length - 1] || release.name;
  }, [release.name]);

  const allowApply = useMemo(
    () => hasPermissionToCreateChangeDatabaseIssueInProject(project),
    [project]
  );

  const isActive = release.state === State.ACTIVE;
  const isDeleted = release.state === State.DELETED;

  const handleAbandon = async () => {
    try {
      await releaseStore.deleteRelease(release.name);
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: error instanceof Error ? error.message : String(error),
      });
    } finally {
      setAbandonOpen(false);
    }
  };

  const handleRestore = async () => {
    try {
      await releaseStore.undeleteRelease(release.name);
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: error instanceof Error ? error.message : String(error),
      });
    }
  };

  return (
    <div className="flex flex-col items-start gap-y-4 p-4 relative">
      {isDeleted && (
        <div className="h-8 w-full text-base font-medium bg-gray-700 text-white flex justify-center items-center">
          {t("common.archived")}
        </div>
      )}

      <div className="w-full flex flex-row items-center justify-between gap-x-4">
        <div className="flex-1 p-0.5 overflow-hidden">
          <h1 className="text-xl font-medium truncate">{releaseDisplayName}</h1>
        </div>
        <div className="flex items-center justify-end gap-x-2">
          {isActive && (
            <Button disabled={!allowApply} onClick={() => setApplyOpen(true)}>
              {t("common.apply-to-database")}
            </Button>
          )}
          {(isActive || isDeleted) && (
            <DropdownMenu>
              <DropdownMenuTrigger
                className="inline-flex items-center justify-center rounded-xs p-1 text-control hover:bg-control-bg focus:outline-hidden"
                aria-label="More actions"
              >
                <EllipsisVertical className="size-4" />
              </DropdownMenuTrigger>
              <DropdownMenuContent>
                {isActive && (
                  <DropdownMenuItem onClick={() => setAbandonOpen(true)}>
                    {t("common.abandon")}
                  </DropdownMenuItem>
                )}
                {isDeleted && (
                  <DropdownMenuItem onClick={handleRestore}>
                    {t("common.restore")}
                  </DropdownMenuItem>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>
      </div>

      <ReleaseBasicInfo
        createTime={
          release.createTime
            ? getTimeForPbTimestampProtoEs(release.createTime) / 1000
            : undefined
        }
        vcsType={release.vcsSource?.vcsType}
        vcsUrl={release.vcsSource?.url}
      />

      <ReleaseFileTable
        files={release.files}
        releaseType={release.type}
        showSelection={false}
        onRowClick={(file) => setSelectedReleaseFile(file)}
      />

      <ApplyToDatabaseSheet
        open={applyOpen}
        onClose={() => setApplyOpen(false)}
        projectId={projectId}
        projectName={projectName}
        releaseName={release.name}
        releaseDisplayName={releaseDisplayName}
      />

      <Sheet
        open={!!selectedReleaseFile}
        onOpenChange={(next) => !next && setSelectedReleaseFile(undefined)}
      >
        <SheetContent width="wide">
          <SheetHeader>
            <SheetTitle>{t("release.file")}</SheetTitle>
          </SheetHeader>
          <SheetBody>
            {selectedReleaseFile && (
              <ReleaseFileDetailPanel releaseFile={selectedReleaseFile} />
            )}
          </SheetBody>
        </SheetContent>
      </Sheet>

      <AlertDialog
        open={abandonOpen}
        onOpenChange={(next) => setAbandonOpen(next)}
      >
        <AlertDialogContent>
          <AlertDialogTitle>
            {t("bbkit.confirm-button.sure-to-abandon")}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t("bbkit.confirm-button.can-undo")}
          </AlertDialogDescription>
          <AlertDialogFooter>
            <Button variant="outline" onClick={() => setAbandonOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button onClick={handleAbandon}>{t("common.confirm")}</Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

// ---------------------------------------------------------------------------
// ReleaseBasicInfo
// ---------------------------------------------------------------------------

function beautifyUrl(url: string): string {
  try {
    const parsed = new URL(url);
    return parsed.pathname.length > 0
      ? parsed.pathname.substring(1)
      : parsed.pathname;
  } catch {
    return url;
  }
}

function ReleaseBasicInfo({
  createTime,
  vcsType,
  vcsUrl,
}: {
  createTime: number | undefined;
  vcsType: VCSType | undefined;
  vcsUrl: string | undefined;
}) {
  const showVcs =
    vcsType !== undefined && vcsType !== VCSType.VCS_TYPE_UNSPECIFIED && vcsUrl;

  return (
    <div className="flex flex-row items-center pl-4 gap-4">
      <div className="flex items-center gap-1">
        <Clock4 className="size-4 text-control-light" />
        {createTime !== undefined && (
          <HumanizeTs ts={createTime} className="text-sm text-control" />
        )}
      </div>
      {showVcs && (
        <div className="flex flex-row items-center gap-1">
          <Link2 className="size-4 text-control-light" />
          <a
            href={vcsUrl}
            target="_blank"
            rel="noreferrer"
            className="text-sm text-accent hover:underline truncate"
          >
            {beautifyUrl(vcsUrl)}
          </a>
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// ApplyToDatabaseSheet
// ---------------------------------------------------------------------------

function ApplyToDatabaseSheet({
  open,
  onClose,
  projectId,
  projectName,
  releaseName,
  releaseDisplayName,
}: {
  open: boolean;
  onClose: () => void;
  projectId: string;
  projectName: string;
  releaseName: string;
  releaseDisplayName: string;
}) {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();

  const [creating, setCreating] = useState(false);
  const [changeSource, setChangeSource] = useState<"DATABASE" | "GROUP">(
    "DATABASE"
  );
  const [selectedDatabaseNames, setSelectedDatabaseNames] = useState<
    Set<string>
  >(new Set());
  const [selectedDatabaseGroup, setSelectedDatabaseGroup] = useState<
    string | undefined
  >();

  useEffect(() => {
    if (open) {
      setCreating(false);
      setChangeSource("DATABASE");
      setSelectedDatabaseNames(new Set());
      setSelectedDatabaseGroup(undefined);
    }
  }, [open]);

  const targets = useMemo(() => {
    if (changeSource === "DATABASE") {
      return [...selectedDatabaseNames];
    }
    return selectedDatabaseGroup ? [selectedDatabaseGroup] : [];
  }, [changeSource, selectedDatabaseNames, selectedDatabaseGroup]);

  const canSubmit = targets.length > 0;

  const handleCreate = async () => {
    if (!canSubmit || creating) return;
    setCreating(true);
    try {
      if (changeSource === "DATABASE" && selectedDatabaseNames.size > 0) {
        await databaseStore.batchGetOrFetchDatabases([
          ...selectedDatabaseNames,
        ]);
      }

      const newPlan = createProto(PlanSchema, {
        title: `Release "${releaseDisplayName}"`,
        description: `Apply release "${releaseDisplayName}" to selected databases.`,
        specs: [
          createProto(Plan_SpecSchema, {
            id: uuidv4(),
            config: {
              case: "changeDatabaseConfig",
              value: createProto(Plan_ChangeDatabaseConfigSchema, {
                targets,
                release: releaseName,
              }),
            },
          }),
        ],
      });

      const createdPlan = await planServiceClientConnect.createPlan({
        parent: projectName,
        plan: newPlan,
      });

      await rolloutServiceClientConnect.createRollout(
        createProto(CreateRolloutRequestSchema, {
          parent: createdPlan.name,
        })
      );

      const planId = createdPlan.name.split("/").pop() || "_";

      onClose();
      router.push(buildPlanDeployRoute({ projectId, planId }));
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: error instanceof Error ? error.message : String(error),
      });
    } finally {
      setCreating(false);
    }
  };

  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="wide">
        <SheetHeader>
          <SheetTitle>{t("common.apply-to-database")}</SheetTitle>
        </SheetHeader>
        <SheetBody>
          <DatabaseAndGroupSelector
            projectName={projectName}
            changeSource={changeSource}
            onChangeSourceChange={setChangeSource}
            selectedDatabaseNames={selectedDatabaseNames}
            onSelectedDatabaseNamesChange={setSelectedDatabaseNames}
            selectedDatabaseGroup={selectedDatabaseGroup}
            onSelectedDatabaseGroupChange={setSelectedDatabaseGroup}
          />
        </SheetBody>
        <SheetFooter>
          <div className="flex-1 text-sm text-control-light">
            {changeSource === "DATABASE" &&
              selectedDatabaseNames.size > 0 &&
              t("database.selected-n-databases", {
                n: selectedDatabaseNames.size,
              })}
          </div>
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button disabled={!canSubmit || creating} onClick={handleCreate}>
            {creating && <Loader2 className="size-4 mr-1 animate-spin" />}
            {t("common.create")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

// ---------------------------------------------------------------------------
// DatabaseAndGroupSelector (inline, ported from ProjectPlanDashboardPage)
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
  const [pageSize] = useSessionPageSize("bb.release-apply-db-selector");
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
                  <input
                    type="checkbox"
                    checked={allSelected}
                    ref={(el) => {
                      if (el) el.indeterminate = someSelected;
                    }}
                    onChange={toggleAll}
                    className="accent-accent"
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
                      "border-b cursor-pointer hover:bg-control-bg",
                      isSelected && "bg-accent/5"
                    )}
                    onClick={() => toggleDatabase(db.name)}
                  >
                    <td className="py-2 pr-2">
                      <input
                        type="checkbox"
                        checked={isSelected}
                        readOnly
                        className="accent-accent"
                      />
                    </td>
                    <td className="py-2 pr-4">
                      <div className="flex items-center gap-x-1.5">
                        {inst && EngineIconPath[inst.engine] && (
                          <img
                            className="size-4 shrink-0"
                            src={EngineIconPath[inst.engine]}
                            alt=""
                          />
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
                        <Tooltip
                          content={
                            db.syncError || t("database.sync-status-failed")
                          }
                        >
                          <XCircle className="size-4 text-error" />
                        </Tooltip>
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
