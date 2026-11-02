import { create } from "@bufbuild/protobuf";
import { Plus, X } from "lucide-react";
import { type ReactNode, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { issueServiceClientConnect } from "@/api";
import { HumanizeTs } from "@/components/HumanizeTs";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Collapsible,
  CollapsiblePanel,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { useReleaseByName } from "@/hooks/useAppState";
import { useMediaQuery } from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";
import {
  buildSourceRow,
  SOURCE_TITLE_LIMIT,
} from "@/routes/project/plan-detail/components/release/vcsSource";
import { pushNotification } from "@/stores";
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
import { displayRoleTitle } from "@/utils/role";
import { PlanStages } from "../../PlanTable";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import { isDatabaseGroupName } from "../utils/phaseSummary";
import { compareVersions } from "./changes/releaseChanges";
import {
  statementKindLabel,
  statementKindOf,
} from "./deploy/foldedTaskDescription";
import { useUserDisplayName } from "./useUserDisplayName";

export function PlanDetailMeta() {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const { patchState } = page;
  const project = page.project;
  const creatorName = useUserDisplayName(page.plan.creator || undefined);
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
    if (releaseResource) {
      fetchRelease(releaseResource, true).catch(() => {});
    }
  }, [fetchRelease, releaseResource]);
  const changesSummary = useMemo(() => {
    const targets = [
      ...new Set(
        (page.plan.specs ?? []).flatMap((spec) =>
          spec.config?.case === "changeDatabaseConfig"
            ? spec.config.value.targets
            : []
        )
      ),
    ];
    const groupCount = targets.filter(isDatabaseGroupName).length;
    const databaseCount = targets.length - groupCount;
    const types = (page.planCheckRuns ?? []).flatMap((run) =>
      run.results.flatMap((result) =>
        result.report.case === "sqlSummaryReport"
          ? result.report.value.statementTypes
          : []
      )
    );
    const kind = statementKindOf(types);
    const databases = kind
      ? t("plan.details.changes-on", {
          kind: statementKindLabel(kind, t),
          count: databaseCount,
        })
      : t("plan.details.databases", { count: databaseCount });
    return groupCount > 0
      ? `${databases} · ${t("plan.details.database-groups", { count: groupCount })}`
      : databases;
  }, [page.plan.specs, page.planCheckRuns, t]);
  // Approvers are recorded in the order of the approval flow's roles.
  const flowRoles = page.issue?.approvalTemplate?.flow?.roles ?? [];
  const approvedSteps = (page.issue?.approvers ?? []).filter(
    (approver) => approver.status === Issue_Approver_Status.APPROVED
  );
  const lastApprover = approvedSteps.at(-1);
  const lastApproverName = useUserDisplayName(lastApprover?.principal);
  const reviewSummary = useMemo(() => {
    const issue = page.issue;
    if (!issue) return "";
    switch (issue.approvalStatus) {
      case ApprovalStatus.APPROVED: {
        const role = flowRoles[approvedSteps.length - 1];
        return lastApprover
          ? t("plan.details.approved-by", {
              user: role
                ? `${lastApproverName} (${displayRoleTitle(role)})`
                : lastApproverName,
            })
          : t("plan.details.approved");
      }
      case ApprovalStatus.REJECTED:
        return t("plan.details.rejected");
      case ApprovalStatus.SKIPPED:
        return t("plan.details.no-approval");
      default: {
        const next = flowRoles[approvedSteps.length];
        return next
          ? t("plan.details.waiting-for", { role: displayRoleTitle(next) })
          : t("plan.details.awaiting-approval");
      }
    }
  }, [page.issue, t, flowRoles, approvedSteps, lastApprover, lastApproverName]);
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
    <span
      className="flex flex-wrap items-center gap-x-1"
      data-testid="plan-detail-created"
    >
      <span>{creatorName}</span>
      {createdTimeMs !== undefined && (
        <>
          <span aria-hidden="true">·</span>
          <HumanizeTs mode="datetime" tsMs={createdTimeMs} />
          <span aria-hidden="true">·</span>
          <HumanizeTs tsMs={createdTimeMs} />
        </>
      )}
    </span>
  );

  // Folded, the block keeps one line: the source title on a release plan,
  // then the progress. The fold lives only while the page is open (D10).
  const toggle = (
    <CollapsibleTrigger
      className="w-auto shrink-0 text-accent hover:underline"
      data-testid="plan-details-toggle"
    >
      {detailsOpen
        ? t("plan.phase.hide-details")
        : t("plan.phase.show-details")}
    </CollapsibleTrigger>
  );
  return (
    <Collapsible
      open={detailsOpen}
      onOpenChange={setDetailsOpen}
      className="mt-1 flex flex-col gap-y-1 text-sm"
      data-testid="plan-details"
    >
      {!detailsOpen && (
        <div className="flex min-w-0 items-center gap-x-3">
          {release && <SourceRowView source={release.vcsSource} titleOnly />}
          <PlanStages plan={page.plan} />
          <span className="flex-1" />
          {toggle}
        </div>
      )}
      <CollapsiblePanel className="flex items-start gap-x-3">
        <dl className="grid min-w-0 flex-1 grid-cols-[max-content_minmax(0,1fr)] gap-x-4 gap-y-1">
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
        {toggle}
      </CollapsiblePanel>
    </Collapsible>
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

function SourceRowView({
  source,
  titleOnly = false,
}: {
  source?: Release_VCSSource;
  // The folded block keeps only the title.
  titleOnly?: boolean;
}) {
  const { t } = useTranslation();
  const narrow = useMediaQuery("(max-width: 639px)");
  const row = buildSourceRow(source);
  if (!row) return <span className="text-control-placeholder">-</span>;
  const text = row.title ?? row.repoAtSha ?? row.href;
  const truncated =
    text && text.length > SOURCE_TITLE_LIMIT
      ? `${text.slice(0, SOURCE_TITLE_LIMIT)}…`
      : text;
  const head = truncated ? (
    row.href ? (
      <a
        className="truncate text-accent hover:underline"
        data-testid="source-part"
        href={row.href}
        rel="noopener noreferrer"
        target="_blank"
        title={text}
      >
        {truncated}
      </a>
    ) : (
      <span className="truncate" data-testid="source-part" title={text}>
        {truncated}
      </span>
    )
  ) : null;
  if (titleOnly) return head;
  const rest = [
    row.author
      ? t("plan.details.by-author", { author: row.author })
      : undefined,
    row.title ? row.repoAtSha : undefined,
    row.branch,
  ].filter((part): part is string => Boolean(part));
  return (
    <span className="flex min-w-0 flex-wrap items-center gap-x-1">
      {head}
      {rest.length > 0 && (
        <span
          className={cn(
            "flex flex-wrap items-center gap-x-1",
            narrow && "basis-full text-control-light"
          )}
        >
          {rest.map((part, index) => (
            <span key={part} className="flex items-center gap-x-1">
              {(!narrow || index > 0) && head !== null && (
                <span aria-hidden="true">·</span>
              )}
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
