import { create } from "@bufbuild/protobuf";
import { ChevronDown, ChevronRight, Plus, X } from "lucide-react";
import { type ReactNode, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { issueServiceClientConnect } from "@/api";
import { HumanizeTs } from "@/components/HumanizeTs";
import {
  buildSourceRow,
  SOURCE_TITLE_LIMIT,
} from "@/components/release/vcsSource";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { useReleaseByName } from "@/hooks/useAppState";
import { useMediaQuery } from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";
import { extractUserEmail, pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { getTimeForPbTimestampProtoEs } from "@/types";
import type { Color } from "@/types/proto-es/google/type/color_pb";
import { ApprovalStatus, IssueStatus } from "@/types/proto-es/v1/common_pb";
import {
  Issue_Approver_Status,
  IssueSchema,
  UpdateIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  type Release,
  Release_Type,
  type Release_VCSSource,
} from "@/types/proto-es/v1/release_service_pb";
import { colorToHex, hasProjectPermissionV2 } from "@/utils";
import { PlanStages } from "../../PlanTable";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import { compareVersions } from "./changes/releaseChanges";
import { statementKindOf } from "./deploy/foldedTaskDescription";

export function PlanDetailMeta() {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const { patchState } = page;
  const project = page.project;
  const creatorEmail = useMemo(
    () => extractUserEmail(page.plan.creator),
    [page.plan.creator]
  );
  const createdTimeMs = getTimeForPbTimestampProtoEs(page.plan.createTime);
  const [detailsOpen, setDetailsOpen] = useState(true);
  const releaseName = page.plan.specs?.find(
    (spec) => spec.config?.case === "changeDatabaseConfig"
  )?.config;
  const releaseResource =
    releaseName?.case === "changeDatabaseConfig"
      ? releaseName.value.release
      : "";
  const release = useReleaseByName(releaseResource);
  const fetchRelease = useAppStore((state) => state.fetchRelease);
  useEffect(() => {
    if (releaseResource) void fetchRelease(releaseResource);
  }, [fetchRelease, releaseResource]);
  const changesSummary = useMemo(() => {
    const targets = new Set(
      (page.plan.specs ?? []).flatMap((spec) =>
        spec.config?.case === "changeDatabaseConfig"
          ? spec.config.value.targets
          : []
      )
    );
    const types = (page.planCheckRuns ?? []).flatMap((run) =>
      run.results.flatMap((result) =>
        result.report.case === "sqlSummaryReport"
          ? result.report.value.statementTypes
          : []
      )
    );
    const kind = statementKindOf(types);
    return kind
      ? t("plan.details.changes-on", { kind, count: targets.size })
      : t("plan.details.databases", { count: targets.size });
  }, [page.plan.specs, page.planCheckRuns, t]);
  const reviewSummary = useMemo(() => {
    const issue = page.issue;
    if (!issue) return "";
    switch (issue.approvalStatus) {
      case ApprovalStatus.APPROVED: {
        const approver = [...issue.approvers]
          .reverse()
          .find((a) => a.status === Issue_Approver_Status.APPROVED);
        return approver
          ? t("plan.details.approved-by", {
              user: approver.principal.replace(/^users\//, ""),
            })
          : t("plan.details.approved");
      }
      case ApprovalStatus.REJECTED:
        return t("plan.details.rejected");
      case ApprovalStatus.SKIPPED:
        return t("plan.details.no-approval");
      default:
        return t("plan.details.awaiting-approval");
    }
  }, [page.issue, t]);
  const allowChangeLabels = useMemo(() => {
    if (!project || !page.issue || page.issue.status !== IssueStatus.OPEN) {
      return false;
    }
    return hasProjectPermissionV2(project, "bb.issues.update");
  }, [page.issue, project]);

  if (page.isCreating) {
    return (
      <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-control-placeholder">
        <InlineLabels
          allowChange
          issueLabels={project?.issueLabels ?? []}
          labels={page.creationIssueLabels}
          onUpdate={page.setCreationIssueLabels}
          required={false}
        />
      </div>
    );
  }

  const handleLabelsUpdate = async (labels: string[]) => {
    if (!page.issue) return;
    const issuePatch = create(IssueSchema, page.issue);
    issuePatch.labels = labels;
    try {
      const response = await issueServiceClientConnect.updateIssue(
        create(UpdateIssueRequestSchema, {
          issue: issuePatch,
          updateMask: { paths: ["labels"] },
        })
      );
      patchState({ issue: response });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    }
  };

  const createdLine = (
    <span className="flex flex-wrap items-center gap-x-1">
      <span>{t("plan.meta.created-by", { user: creatorEmail })}</span>
      {createdTimeMs !== undefined && (
        <>
          <span aria-hidden="true">·</span>
          <HumanizeTs tsMs={createdTimeMs} />
        </>
      )}
    </span>
  );

  return (
    <div className="mt-1 flex flex-col gap-y-1 text-xs">
      <Button
        type="button"
        appearance="secondary"
        size="xs"
        aria-expanded={detailsOpen}
        className="h-auto self-start bg-transparent p-0 text-control-light hover:bg-transparent"
        data-testid="plan-details-toggle"
        onClick={() => setDetailsOpen((open) => !open)}
      >
        {detailsOpen ? (
          <ChevronDown className="size-3.5" />
        ) : (
          <ChevronRight className="size-3.5" />
        )}
        {t("plan.details.self")}
      </Button>
      {detailsOpen && (
        <dl className="grid grid-cols-[max-content_minmax(0,1fr)] gap-x-4 gap-y-1">
          {release && (
            <DetailRow label={t("plan.details.source")} row="source">
              <SourceRowView source={release.vcsSource} />
            </DetailRow>
          )}
          <DetailRow
            label={
              release ? t("plan.details.triggered") : t("plan.details.created")
            }
            row="created"
          >
            {createdLine}
          </DetailRow>
          {page.issue && !release && (
            <DetailRow label={t("plan.details.labels")} row="labels">
              <InlineLabels
                allowChange={allowChangeLabels}
                issueLabels={project?.issueLabels ?? []}
                labels={page.issue.labels || []}
                onUpdate={handleLabelsUpdate}
                required={
                  page.issue.draft && (project?.forceIssueLabels ?? false)
                }
              />
            </DetailRow>
          )}
          {release ? (
            <DetailRow label={t("plan.details.release")} row="release">
              <span data-testid="plan-detail-release">
                {release.type === Release_Type.DECLARATIVE
                  ? t("plan.details.state-based", {
                      version: release.files[0]?.version ?? "",
                    })
                  : t("plan.details.migration-based", {
                      count: release.files.length,
                      version: newestVersion(release),
                    })}
              </span>
            </DetailRow>
          ) : (
            <DetailRow label={t("plan.details.changes")} row="changes">
              {changesSummary}
            </DetailRow>
          )}
          {page.issue && !release && (
            <DetailRow label={t("plan.details.review")} row="review">
              {reviewSummary}
            </DetailRow>
          )}
          <DetailRow label={t("plan.details.progress")} row="progress">
            <PlanStages plan={page.plan} />
          </DetailRow>
        </dl>
      )}
    </div>
  );
}

function DetailRow({
  label,
  row,
  children,
}: {
  label: string;
  row: string;
  children: ReactNode;
}) {
  return (
    <div className="contents" data-testid="plan-detail-row" data-row={row}>
      <dt className="text-control-placeholder">{label}</dt>
      <dd className="min-w-0 text-control">{children}</dd>
    </div>
  );
}

const newestVersion = (release: Release) =>
  [...release.files]
    .map((file) => file.version)
    .sort(compareVersions)
    .at(-1) ?? "";

function SourceRowView({ source }: { source?: Release_VCSSource }) {
  const { t } = useTranslation();
  const narrow = useMediaQuery("(max-width: 639px)");
  const row = buildSourceRow(source);
  if (!row) return <span className="text-control-placeholder">-</span>;
  const truncated =
    row.title && row.title.length > SOURCE_TITLE_LIMIT
      ? `${row.title.slice(0, SOURCE_TITLE_LIMIT)}…`
      : row.title;
  const link = (text: string, full?: string) => (
    <a
      className="truncate text-accent hover:underline"
      data-testid="source-part"
      href={row.href}
      rel="noopener noreferrer"
      target="_blank"
      title={full}
    >
      {text}
    </a>
  );
  const rest = [
    row.title && row.author
      ? t("plan.details.by-author", { author: row.author })
      : undefined,
    row.title ? row.repoAtSha : undefined,
    row.branch,
  ].filter((part): part is string => Boolean(part));
  return (
    <span className="flex min-w-0 flex-wrap items-center gap-x-1">
      {row.title
        ? link(truncated ?? "", row.title)
        : row.repoAtSha
          ? link(row.repoAtSha)
          : null}
      {rest.length > 0 && (
        <span
          className={cn(
            "flex flex-wrap items-center gap-x-1",
            narrow && "basis-full text-control-light"
          )}
        >
          {rest.map((part) => (
            <span key={part} className="flex items-center gap-x-1">
              {!narrow && <span aria-hidden="true">·</span>}
              <span data-testid="source-part">{part}</span>
            </span>
          ))}
        </span>
      )}
    </span>
  );
}

function InlineLabels({
  allowChange,
  issueLabels,
  labels,
  onUpdate,
  required,
}: {
  allowChange: boolean;
  issueLabels: Array<{ color?: Color; value: string }>;
  labels: string[];
  onUpdate: (labels: string[]) => Promise<void> | void;
  /** The project forces issue labels, so submitting for review needs one. */
  required: boolean;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [isUpdating, setIsUpdating] = useState(false);

  const toggleLabel = async (value: string) => {
    const next = labels.includes(value)
      ? labels.filter((label) => label !== value)
      : [...labels, value];
    try {
      setIsUpdating(true);
      await onUpdate(next);
    } finally {
      setIsUpdating(false);
    }
  };

  const chips = labels.map((value) => {
    const option = issueLabels.find((item) => item.value === value);
    return (
      <span
        key={value}
        className="inline-flex items-center gap-1 rounded-xs bg-control-bg px-1.5 py-0.5 text-xs text-control"
      >
        <span
          className="size-2.5 shrink-0 rounded-sm"
          style={{
            backgroundColor: option?.color
              ? colorToHex(option.color)
              : undefined,
          }}
        />
        <span className="truncate">{value}</span>
        {allowChange && (
          <Button
            appearance="secondary"
            size="xs"
            aria-label={t("common.remove")}
            className={cn(
              "inline-flex shrink-0 items-center justify-center text-control-placeholder transition-colors",
              isUpdating
                ? "cursor-not-allowed opacity-60"
                : "hover:text-control"
            )}
            disabled={isUpdating}
            onClick={() => void toggleLabel(value)}
            type="button"
          >
            <X className="size-3" />
          </Button>
        )}
      </span>
    );
  });

  const triggerDisabled = !allowChange || isUpdating;

  return (
    <div className="flex flex-wrap items-center gap-1">
      {chips}
      <Popover open={open} onOpenChange={allowChange ? setOpen : undefined}>
        <PopoverTrigger
          render={
            <Button
              appearance="secondary"
              size="xs"
              className={cn(
                "inline-flex items-center gap-1 rounded-xs border border-dashed border-control-border px-1.5 py-0.5 text-xs text-control-placeholder transition-colors",
                allowChange &&
                  !isUpdating &&
                  "hover:border-control hover:text-control",
                triggerDisabled && "cursor-not-allowed opacity-60"
              )}
              disabled={triggerDisabled}
              type="button"
            />
          }
        >
          <Plus className="size-3" />
          <span>{t("issue.labels")}</span>
          {/* Only while unsatisfied: once a label is picked the trigger is an
              "add another" affordance, and the requirement is already met. */}
          {required && labels.length === 0 && (
            <span className="text-error">*</span>
          )}
        </PopoverTrigger>
        <PopoverContent
          side="bottom"
          align="start"
          initialFocus={false}
          finalFocus={false}
          className="w-56 overflow-hidden bg-background p-0"
        >
          <div className="max-h-60 overflow-y-auto">
            {issueLabels.length === 0 ? (
              <div className="px-3 py-6 text-sm text-control-placeholder">
                {t("common.no-data")}
              </div>
            ) : (
              issueLabels.map((option) => {
                const isSelected = labels.includes(option.value);
                return (
                  <Button
                    appearance="secondary"
                    size="md"
                    key={option.value}
                    className="flex w-full min-w-0 justify-start gap-x-2 px-3 py-2 text-left text-sm transition-colors hover:bg-control-bg"
                    disabled={isUpdating}
                    onClick={() => void toggleLabel(option.value)}
                    type="button"
                  >
                    <Checkbox checked={isSelected} />
                    <span
                      className="size-4 shrink-0 rounded-sm"
                      style={{
                        backgroundColor: option.color
                          ? colorToHex(option.color)
                          : undefined,
                      }}
                    />
                    <span
                      className="min-w-0 flex-1 truncate"
                      title={option.value}
                    >
                      {option.value}
                    </span>
                  </Button>
                );
              })
            )}
          </div>
        </PopoverContent>
      </Popover>
    </div>
  );
}
