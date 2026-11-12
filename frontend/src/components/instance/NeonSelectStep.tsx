import {
  Ban,
  ChevronDown,
  ChevronRight,
  CircleCheck,
  CircleDashed,
  Info,
  LoaderCircle,
  RotateCw,
  TriangleAlert,
} from "lucide-react";
import { useId, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { SearchInput } from "@/components/ui/search-input";
import { SegmentedControl } from "@/components/ui/segmented-control";
import {
  Table,
  TableBody,
  TableCell,
  TableEmptyView,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import type { NeonBranch } from "@/types/proto-es/v1/instance_service_pb";
import { NeonWarning } from "./NeonWarning";
import {
  branchAvailable,
  branchKey,
  formatClock,
  type NeonImportDraft,
  type NeonRequestFailure,
  neonReasonText,
  statusReason,
} from "./neon-import";

export type NeonProjectFailure = {
  projectId: string;
  reason: string;
  message: string;
  retrying: boolean;
};

export type NeonDiscoveryStatus = {
  // False until a first page has been applied to the list.
  started: boolean;
  loading: boolean;
  complete: boolean;
  pageFailure?: NeonRequestFailure;
  retryInSeconds: number;
  // When Neon allows the next page request, while that is still ahead.
  retryAt?: number;
};

type ProjectGroup = {
  projectId: string;
  projectTitle: string;
  rows: NeonImportDraft[];
};

export function groupByNeonProject(
  rows: readonly NeonImportDraft[]
): ProjectGroup[] {
  const groups = new Map<string, ProjectGroup>();
  for (const row of rows) {
    const { projectId, projectTitle } = row.branch;
    let group = groups.get(projectId);
    if (!group) {
      group = { projectId, projectTitle: projectTitle || projectId, rows: [] };
      groups.set(projectId, group);
    }
    group.rows.push(row);
  }
  return [...groups.values()];
}

const checkedState = (selected: number, total: number) =>
  selected === 0 ? false : selected === total ? true : "indeterminate";

const COLUMN_COUNT = 5;

// Widens a table checkbox's tap target to about 44px without moving it.
export const CHECKBOX_HIT_AREA =
  "relative after:absolute after:-inset-x-3.5 after:-inset-y-3 after:content-['']";
export const CHECKBOX_HIT_AREA_DENSE =
  "relative after:absolute after:-inset-x-3.5 after:-inset-y-2 after:content-['']";

export function NeonSelectStep({
  drafts,
  status,
  failures,
  capacity,
  onSelect,
  onLoadMore,
  onLoadAll,
  onRetryProject,
  onRestart,
}: {
  drafts: readonly NeonImportDraft[];
  status: NeonDiscoveryStatus;
  failures: readonly NeonProjectFailure[];
  // The number of instances the workspace can still add; undefined is no limit.
  capacity?: number;
  onSelect: (keys: readonly string[], selected: boolean) => void;
  onLoadMore: () => void;
  onLoadAll: () => void;
  onRetryProject: (projectId: string) => void;
  onRestart: () => void;
}) {
  const { t } = useTranslation();
  const id = useId();
  const [view, setView] = useState<"loaded" | "selected">("loaded");
  const [filter, setFilter] = useState("");
  const [collapsed, setCollapsed] = useState<ReadonlySet<string>>(new Set());

  const selectedCount = drafts.filter((row) => row.selected).length;
  const projectCount = useMemo(
    () => new Set(drafts.map((row) => row.branch.projectId)).size,
    [drafts]
  );
  // Filtering before every page is loaded would hide branches Neon has not
  // returned yet behind what looks like a complete search.
  const query = status.complete ? filter.trim().toLowerCase() : "";
  const shown = drafts.filter(
    (row) =>
      (view === "loaded" || row.selected) &&
      (!query ||
        [row.branch.projectTitle, row.branch.branchTitle, row.branch.branchId]
          .join(" ")
          .toLowerCase()
          .includes(query))
  );
  // Per Neon project, over every loaded row, so a filter or the selected view
  // cannot make a project look smaller than it is.
  const projectTotals = useMemo(() => {
    const totals = new Map<string, { loaded: number; selected: number }>();
    for (const row of drafts) {
      const total = totals.get(row.branch.projectId) ?? {
        loaded: 0,
        selected: 0,
      };
      total.loaded++;
      if (row.selected) total.selected++;
      totals.set(row.branch.projectId, total);
    }
    return totals;
  }, [drafts]);
  const narrowed = view === "selected" || !!query;
  const shownAvailable = shown.filter((row) => branchAvailable(row.branch));
  const shownSelected = shownAvailable.filter((row) => row.selected).length;
  const allSelected =
    shownAvailable.length > 0 && shownSelected === shownAvailable.length;
  const overLimit =
    capacity === undefined ? 0 : Math.max(0, selectedCount - capacity);
  const neonProjects = (count: number) =>
    t("instance.neon.count.neon-projects", { count });
  const branches = (count: number) =>
    t("instance.neon.count.branches", { count });

  const selectAllLabel = !status.started
    ? t("instance.neon.select.select-all-label")
    : view === "selected"
      ? t("instance.neon.select.selected-shown", {
          count: shownAvailable.length,
        })
      : query
        ? allSelected
          ? t("instance.neon.select.all-selected-matching", {
              count: shownAvailable.length,
            })
          : t("instance.neon.select.select-all-matching", {
              count: shownAvailable.length,
            })
        : allSelected
          ? t("instance.neon.select.all-selected", {
              count: shownAvailable.length,
            })
          : t("instance.neon.select.select-all", {
              count: shownAvailable.length,
            });

  const toggleCollapsed = (projectId: string) =>
    setCollapsed((previous) => {
      const next = new Set(previous);
      if (!next.delete(projectId)) next.add(projectId);
      return next;
    });

  const unavailableReason = (branch: NeonBranch) =>
    branch.discoveryError
      ? neonReasonText(
          t,
          statusReason(branch.discoveryError),
          t("instance.neon.reason.inspection-failed")
        )
      : t("instance.neon.reason.no-databases");

  const pageFailureText = (failure: NeonRequestFailure) => {
    switch (failure.kind) {
      case "rate-limited":
        return [
          t("instance.neon.select.page-error-rate"),
          status.retryAt !== undefined
            ? t("instance.neon.select.page-error-retry-in", {
                time: formatClock(new Date(status.retryAt)),
              })
            : t("instance.neon.select.page-error-retry-now"),
        ].join(" ");
      case "unavailable":
        return t("instance.neon.select.page-error-unavailable");
      case "key-invalid":
        return t("instance.neon.select.page-error-key");
      case "token-invalid":
        return t("instance.neon.select.page-error-token");
      default:
        return failure.message || t("instance.neon.connect.error.other-body");
    }
  };

  return (
    <div className="flex w-full flex-col gap-4">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
        <SegmentedControl
          size="sm"
          appearance="soft"
          ariaLabel={t("instance.neon.select.view-label")}
          value={view}
          onValueChange={setView}
          options={[
            {
              value: "loaded",
              label: (
                <span className="px-2 text-xs font-medium">
                  {t("instance.neon.select.view-loaded", {
                    count: drafts.length,
                  })}
                </span>
              ),
            },
            {
              value: "selected",
              label: (
                <span className="px-2 text-xs font-medium">
                  {t("instance.neon.select.view-selected", {
                    count: selectedCount,
                  })}
                </span>
              ),
            },
          ]}
        />
        <SearchInput
          size="sm"
          wrapperClassName="w-56 max-w-full flex-none"
          aria-label={t("instance.neon.select.filter-label")}
          placeholder={t("instance.neon.select.filter-placeholder")}
          disabled={!status.complete}
          value={filter}
          onChange={(event) => setFilter(event.target.value)}
        />
        {!status.complete && (
          <span className="text-xs leading-4 text-control-light">
            {t("instance.neon.select.filter-locked")}
          </span>
        )}
        <Tooltip
          content={t("instance.neon.select.restart-help")}
          render={<span className="ml-auto inline-flex" />}
        >
          <Button
            appearance="secondary"
            size="sm"
            disabled={status.loading}
            onClick={onRestart}
          >
            <RotateCw className="size-3.5" />
            {t("instance.neon.select.restart")}
          </Button>
        </Tooltip>
      </div>

      <div
        role="status"
        className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs leading-4 text-control-light"
      >
        {!status.started ? (
          <span className="inline-flex items-center gap-1.5">
            <LoaderCircle className="size-4 animate-spin" />
            {t("instance.neon.select.status.listing")}
          </span>
        ) : status.complete ? (
          <span className="inline-flex items-center gap-1.5 text-main">
            {failures.length === 0 && (
              <CircleCheck className="size-4 text-success" />
            )}
            {t("instance.neon.select.status.complete", {
              count: drafts.length,
              projects: neonProjects(projectCount),
            })}
          </span>
        ) : (
          <>
            <span className="inline-flex items-center gap-1.5 text-main">
              <CircleDashed className="size-4 text-control-light" />
              {t("instance.neon.select.status.partial", {
                count: drafts.length,
                projects: neonProjects(projectCount),
              })}
            </span>
            <span>{t("instance.neon.select.status.total-note")}</span>
          </>
        )}
        {failures.length > 0 && (
          <span className="text-warning-hover">
            {t("instance.neon.select.status.failed-projects", {
              count: failures.length,
            })}
          </span>
        )}
      </div>

      {capacity === 0 ? (
        <Alert
          variant="error"
          title={t("instance.neon.limit-reached.title")}
          description={t("instance.neon.limit-reached.body")}
        />
      ) : (
        overLimit > 0 &&
        capacity !== undefined && (
          <Alert
            variant="error"
            title={t("instance.neon.select.over-limit-title", {
              branches: branches(selectedCount),
              instances: t("instance.neon.count.more-instances", {
                count: capacity,
              }),
            })}
            description={t("instance.neon.select.over-limit-body", {
              branches: branches(overLimit),
            })}
          />
        )
      )}

      <div className="overflow-x-auto rounded-sm border border-block-border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-10">
                {status.started && (
                  <Checkbox
                    className={CHECKBOX_HIT_AREA_DENSE}
                    aria-label={
                      view === "selected"
                        ? t("instance.neon.select.deselect-shown")
                        : selectAllLabel
                    }
                    checked={checkedState(shownSelected, shownAvailable.length)}
                    disabled={shownAvailable.length === 0}
                    onCheckedChange={(checked) =>
                      onSelect(
                        shownAvailable.map((row) => branchKey(row.branch)),
                        checked
                      )
                    }
                  />
                )}
              </TableHead>
              <TableHead>
                {status.started
                  ? selectAllLabel
                  : t("instance.neon.select.col.branch")}
              </TableHead>
              <TableHead className="hidden md:table-cell">
                {t("instance.neon.select.col.databases")}
              </TableHead>
              <TableHead className="hidden lg:table-cell">
                {t("instance.neon.select.col.roles")}
              </TableHead>
              <TableHead className="hidden sm:table-cell">
                {t("common.status")}
              </TableHead>
            </TableRow>
          </TableHeader>

          {!status.started && (
            <TableBody striped={false} aria-hidden="true">
              {Array.from({ length: 6 }, (_, index) => (
                <TableRow key={index}>
                  <TableCell>
                    <div className="size-4 rounded-sm bg-control-bg" />
                  </TableCell>
                  <TableCell>
                    <div className="h-4 w-40 max-w-full rounded-xs bg-control-bg" />
                    <div className="mt-1.5 h-3 w-56 max-w-full rounded-xs bg-control-bg" />
                  </TableCell>
                  <TableCell className="hidden md:table-cell">
                    <div className="h-4 w-32 rounded-xs bg-control-bg" />
                  </TableCell>
                  <TableCell className="hidden lg:table-cell">
                    <div className="h-4 w-32 rounded-xs bg-control-bg" />
                  </TableCell>
                  <TableCell className="hidden sm:table-cell">
                    <div className="h-4 w-16 rounded-xs bg-control-bg" />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          )}

          {failures.length > 0 && (
            <TableBody striped={false}>
              {failures.map((failure) => (
                <TableRow
                  key={failure.projectId}
                  className="bg-error/5 hover:bg-error/5"
                >
                  <TableCell className="align-top">
                    <TriangleAlert className="size-4 text-error" />
                  </TableCell>
                  <TableCell colSpan={COLUMN_COUNT - 1}>
                    <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-2">
                      <div className="min-w-0">
                        <div className="font-medium text-main break-words">
                          {t("instance.neon.select.project-failed", {
                            id: failure.projectId,
                          })}
                        </div>
                        <p className="max-w-2xl text-xs leading-4 text-control">
                          {neonReasonText(t, failure.reason, failure.message)}{" "}
                          {t("instance.neon.select.project-failed-help")}
                        </p>
                      </div>
                      <Button
                        appearance="outline"
                        size="sm"
                        disabled={failure.retrying}
                        aria-label={t("instance.neon.select.retry-project", {
                          id: failure.projectId,
                        })}
                        onClick={() => onRetryProject(failure.projectId)}
                      >
                        {failure.retrying ? (
                          <LoaderCircle className="size-3.5 animate-spin" />
                        ) : (
                          <RotateCw className="size-3.5" />
                        )}
                        {failure.retrying
                          ? t("instance.neon.select.retrying")
                          : t("instance.neon.select.retry")}
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          )}

          {groupByNeonProject(shown).map((group) => {
            const available = group.rows.filter((row) =>
              branchAvailable(row.branch)
            );
            const selected = available.filter((row) => row.selected).length;
            const unavailable = group.rows.length - available.length;
            const closed = collapsed.has(group.projectId);
            const totals = projectTotals.get(group.projectId);
            return (
              <TableBody key={group.projectId} striped={false}>
                <TableRow className="bg-control-bg/70 hover:bg-control-bg/70">
                  <TableCell className="w-10 py-2">
                    <Checkbox
                      className={CHECKBOX_HIT_AREA_DENSE}
                      aria-label={
                        narrowed
                          ? t("instance.neon.select.group-select-shown-label", {
                              project: group.projectTitle,
                            })
                          : t("instance.neon.select.group-select-label", {
                              project: group.projectTitle,
                            })
                      }
                      checked={checkedState(selected, available.length)}
                      disabled={available.length === 0}
                      onCheckedChange={(checked) =>
                        onSelect(
                          available.map((row) => branchKey(row.branch)),
                          checked
                        )
                      }
                    />
                  </TableCell>
                  <TableCell
                    colSpan={COLUMN_COUNT - 1}
                    className="max-w-0 py-2"
                  >
                    <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
                      <Button
                        appearance="secondary"
                        size="sm"
                        className="h-auto min-w-0 items-start px-0 text-left text-sm whitespace-normal hover:bg-transparent"
                        aria-expanded={!closed}
                        onClick={() => toggleCollapsed(group.projectId)}
                      >
                        {closed ? (
                          <ChevronRight className="mt-0.5 size-4 shrink-0 text-control-light" />
                        ) : (
                          <ChevronDown className="mt-0.5 size-4 shrink-0 text-control-light" />
                        )}
                        <span className="mt-0.5 shrink-0 text-xs font-normal text-control-light">
                          {t("instance.neon.neon-project")}
                        </span>
                        <span className="min-w-0 font-medium wrap-anywhere text-main">
                          {group.projectTitle}
                        </span>
                      </Button>
                      <span className="hidden font-mono text-xs text-control-light sm:inline">
                        {group.projectId}
                      </span>
                      <span className="ml-auto text-xs text-control-light">
                        {t("instance.neon.select.group-summary", {
                          selected: totals?.selected ?? selected,
                          count: totals?.loaded ?? group.rows.length,
                        })}
                        {narrowed &&
                          ` · ${t("instance.neon.select.group-shown", { count: group.rows.length })}`}
                        {unavailable > 0 &&
                          ` · ${t("instance.neon.select.group-unavailable", { count: unavailable })}`}
                      </span>
                    </div>
                  </TableCell>
                </TableRow>
                {!closed &&
                  group.rows.map((row) => {
                    const { branch } = row;
                    const available = branchAvailable(branch);
                    const checkboxId = `${id}-${branchKey(branch)}`;
                    const owners = [
                      ...new Set(branch.databases.map((db) => db.owner)),
                    ];
                    const inspected = !branch.discoveryError || owners.length;
                    return (
                      <TableRow
                        key={branchKey(branch)}
                        data-state={row.selected ? "selected" : undefined}
                      >
                        <TableCell className="align-top">
                          <Checkbox
                            id={checkboxId}
                            className={CHECKBOX_HIT_AREA}
                            aria-label={t(
                              "instance.neon.select.row-select-label",
                              { name: row.title }
                            )}
                            checked={row.selected}
                            disabled={!available}
                            onCheckedChange={(checked) =>
                              onSelect([branchKey(branch)], checked)
                            }
                          />
                        </TableCell>
                        <TableCell className="w-full max-w-0 align-top">
                          <label
                            htmlFor={checkboxId}
                            className={cn(
                              "block",
                              available && "cursor-pointer"
                            )}
                          >
                            <div
                              className={cn(
                                "truncate font-medium",
                                available ? "text-main" : "text-control-light"
                              )}
                            >
                              {branch.branchTitle || branch.branchId}
                            </div>
                            <div className="truncate font-mono text-xs leading-4 text-control-light">
                              {branch.branchId}
                            </div>
                          </label>
                          {branch.databases.length > 0 && (
                            <div className="mt-1 truncate text-xs leading-4 md:hidden">
                              {branch.databases.map((db) => db.name).join(", ")}
                            </div>
                          )}
                          {!available && (
                            <div className="mt-1 text-xs leading-4 text-control sm:hidden">
                              <span className="inline-flex items-center gap-1 font-medium">
                                <Ban className="size-3.5" />
                                {t("instance.neon.select.cannot-import")}
                              </span>
                              {" · "}
                              {unavailableReason(branch)}
                            </div>
                          )}
                        </TableCell>
                        <TableCell className="hidden align-top whitespace-nowrap md:table-cell">
                          {!inspected || branch.databases.length === 0 ? (
                            <span className="text-control-light">
                              {t("instance.neon.select.not-available")}
                            </span>
                          ) : owners.length === 1 ? (
                            <>
                              <div className="text-main">
                                {branch.databases
                                  .map((db) => db.name)
                                  .join(", ")}
                              </div>
                              <div className="text-xs leading-4 text-control-light">
                                {t("instance.neon.select.owner", {
                                  owner: owners[0],
                                })}
                              </div>
                            </>
                          ) : (
                            <>
                              <div className="text-main">
                                {branch.databases
                                  .map((db) => `${db.name} (${db.owner})`)
                                  .join(", ")}
                              </div>
                              <div className="text-xs leading-4 text-control-light">
                                {t("instance.neon.select.owners-differ")}
                              </div>
                            </>
                          )}
                        </TableCell>
                        <TableCell className="hidden align-top text-xs whitespace-nowrap lg:table-cell">
                          {branch.roles.length > 0 ? (
                            branch.roles.join(", ")
                          ) : (
                            <span className="text-control-light">
                              {t("instance.neon.select.not-available")}
                            </span>
                          )}
                        </TableCell>
                        <TableCell className="hidden align-top sm:table-cell">
                          {available ? (
                            <span className="text-control-light">
                              {t("instance.neon.select.available")}
                            </span>
                          ) : (
                            <>
                              <div className="inline-flex items-center gap-1 font-medium whitespace-nowrap text-control">
                                <Ban className="size-3.5" />
                                {t("instance.neon.select.cannot-import")}
                              </div>
                              <div className="w-40 text-xs leading-4 text-control-light">
                                {unavailableReason(branch)}
                              </div>
                            </>
                          )}
                        </TableCell>
                      </TableRow>
                    );
                  })}
              </TableBody>
            );
          })}

          {status.started &&
            shown.length === 0 &&
            (drafts.length > 0 ||
              failures.length === 0 ||
              !status.complete) && (
              <TableBody striped={false}>
                <TableEmptyView
                  colSpan={COLUMN_COUNT}
                  contentClassName="min-h-24 flex-col gap-2"
                >
                  {query ? (
                    <>
                      {t("instance.neon.select.empty-filter", {
                        query: filter,
                      })}
                      <Button
                        appearance="link"
                        size="sm"
                        className="h-auto p-0"
                        onClick={() => setFilter("")}
                      >
                        {t("instance.neon.select.clear-filter")}
                      </Button>
                    </>
                  ) : view === "selected" ? (
                    t("instance.neon.select.empty-selected")
                  ) : status.complete ? (
                    t("instance.neon.select.empty")
                  ) : (
                    t("instance.neon.select.empty-more")
                  )}
                </TableEmptyView>
              </TableBody>
            )}
        </Table>

        {status.started && !status.complete && !status.pageFailure && (
          <div className="flex flex-wrap items-center justify-center gap-2 border-t border-block-border px-4 py-3">
            <Button
              appearance="outline"
              size="sm"
              disabled={status.loading}
              onClick={onLoadMore}
            >
              {status.loading && (
                <LoaderCircle className="size-3.5 animate-spin" />
              )}
              {status.loading
                ? t("instance.neon.select.loading")
                : t("instance.neon.select.load-more")}
            </Button>
            <Button
              appearance="secondary"
              size="sm"
              disabled={status.loading}
              onClick={onLoadAll}
            >
              {t("instance.neon.select.load-all")}
            </Button>
          </div>
        )}
        {status.pageFailure && (
          <div className="border-t border-block-border p-3">
            <NeonWarning
              title={t("instance.neon.select.page-error-title")}
              description={[
                pageFailureText(status.pageFailure),
                t("instance.neon.select.page-error-kept", {
                  branches: branches(drafts.length),
                  selections: t("instance.neon.count.selections", {
                    count: selectedCount,
                  }),
                }),
              ].join(" ")}
            >
              <Button
                appearance="outline"
                size="sm"
                className="mt-2"
                disabled={status.loading || status.retryInSeconds > 0}
                onClick={onLoadMore}
              >
                <RotateCw className="size-3.5" />
                {t("instance.neon.select.retry-page")}
              </Button>
            </NeonWarning>
          </div>
        )}
      </div>

      <p className="flex items-start gap-1.5 text-xs leading-4 text-control-light">
        <Info className="mt-px size-3.5 shrink-0" />
        <span>{t("instance.neon.select.live-note")}</span>
      </p>
    </div>
  );
}
