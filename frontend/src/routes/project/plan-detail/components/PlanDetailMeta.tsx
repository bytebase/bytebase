import { create } from "@bufbuild/protobuf";
import { CircleCheck, CircleX, Clock, Plus, X } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { issueServiceClientConnect } from "@/api";
import { HumanizeTs } from "@/components/HumanizeTs";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { useReleaseByName } from "@/hooks/useAppState";
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
import { extractReleaseUID } from "@/utils/v1/release";
import { PlanStageParts } from "../../PlanTable";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import { isDatabaseGroupName } from "../utils/phaseSummary";
import {
  statementKindLabel,
  statementKindOf,
} from "./deploy/foldedTaskDescription";
import { useUserDisplayName } from "./useUserDisplayName";

// The lines under the plan title. Every part starts with the same left padding, which
// holds its separator; a line's negative margin moves a row's first padding
// into the clipped strip left of the box, so a separator or an arrow shows
// only between two parts on one row. Rows wrapped or not share one height and
// one gap.
const LINES =
  "flex flex-col gap-y-1 overflow-x-clip [overflow-clip-margin:2px]";
const LINE = "-ml-4 flex flex-wrap items-center gap-y-1";
const PART =
  "relative flex min-h-6 min-w-0 max-w-full items-center gap-x-1 pl-4";
const SEPARATED = `${PART} before:absolute before:left-1.5 before:text-control-light before:content-['·']`;
const ARROW = "absolute left-0.5";

// The end a reader identifies a branch by.
const BRANCH_END_LENGTH = 16;

type Approval = {
  tone: "done" | "rejected" | "waiting" | "none";
  text: string;
};

export function PlanDetailMeta() {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const { patchState } = page;
  const project = page.project;
  const creatorName = useUserDisplayName(page.plan.creator || undefined);
  const createdTimeMs = getTimeForPbTimestampProtoEs(page.plan.createTime);
  const releaseConfig = page.plan.specs?.find(
    (spec) => spec.config?.case === "changeDatabaseConfig"
  )?.config;
  const releaseResource =
    releaseConfig?.case === "changeDatabaseConfig"
      ? releaseConfig.value.release
      : "";
  const release = useReleaseByName(releaseResource);
  // The spec names the release, so the lines are a release plan's from the
  // first render, before the release is read or when it cannot be.
  const isReleasePlan = Boolean(releaseResource);
  const fetchRelease = useAppStore((state) => state.fetchRelease);
  const [releaseUnreadable, setReleaseUnreadable] = useState(false);
  useEffect(() => {
    setReleaseUnreadable(false);
    if (releaseResource) {
      fetchRelease(releaseResource, true).catch(() =>
        setReleaseUnreadable(true)
      );
    }
  }, [fetchRelease, releaseResource]);
  const changesParts = useMemo(() => {
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
    // A kind of target the plan does not have is left out.
    const parts = [
      databaseCount > 0 || groupCount === 0
        ? t("plan.details.databases", { count: databaseCount })
        : "",
      groupCount > 0
        ? t("plan.details.database-groups", { count: groupCount })
        : "",
    ].filter(Boolean);
    if (!kind) return parts;
    const [first, ...rest] = parts;
    return [
      t("plan.details.changes-on-targets", {
        kind: statementKindLabel(kind, t),
        targets: first,
      }),
      ...rest,
    ];
  }, [page.plan.specs, page.planCheckRuns, t]);
  // Approvers are recorded in the order of the approval flow's roles.
  const flowRoles = page.issue?.approvalTemplate?.flow?.roles ?? [];
  const approvedSteps = (page.issue?.approvers ?? []).filter(
    (approver) => approver.status === Issue_Approver_Status.APPROVED
  );
  const lastApprover = approvedSteps.at(-1);
  const lastApproverName = useUserDisplayName(lastApprover?.principal);
  const approval = useMemo((): Approval | undefined => {
    const issue = page.issue;
    // A draft has no state line: the title row's action says where it stands.
    if (!issue || issue.draft) return undefined;
    switch (issue.approvalStatus) {
      case ApprovalStatus.APPROVED: {
        const role = flowRoles[approvedSteps.length - 1];
        return {
          tone: "done",
          text: lastApprover
            ? t("plan.details.approved-by", {
                user: role
                  ? `${lastApproverName} (${displayRoleTitle(role)})`
                  : lastApproverName,
              })
            : t("plan.details.approved"),
        };
      }
      case ApprovalStatus.REJECTED:
        return { tone: "rejected", text: t("plan.details.rejected") };
      case ApprovalStatus.SKIPPED:
        return { tone: "none", text: t("plan.details.no-approval") };
      default: {
        const next = flowRoles[approvedSteps.length];
        return {
          tone: "waiting",
          text: next
            ? t("plan.details.waiting-for", { role: displayRoleTitle(next) })
            : t("plan.details.awaiting-approval"),
        };
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

  // Until the release is read the source line holds its row; it goes when
  // the release cannot be read or carries no source.
  const showSourceLine =
    isReleasePlan &&
    (release ? Boolean(buildSourceRow(release.vcsSource)) : !releaseUnreadable);
  const releaseParts = !isReleasePlan
    ? []
    : !release
      ? [extractReleaseUID(releaseResource)]
      : release.type === Release_Type.DECLARATIVE
        ? [t("plan.details.state-based"), release.files[0]?.version ?? ""]
        : [
            t("plan.details.migration-based"),
            t("plan.details.release-files", { count: release.files.length }),
            t("plan.details.update-to-version", {
              version: newestVersion(release),
            }),
          ];
  const labelsShown =
    page.issue && (allowChangeLabels || (page.issue.labels ?? []).length > 0);
  // The verb tells the CI account apart from the source line's author.
  const creatorText = isReleasePlan
    ? t("plan.details.triggered-by", { user: creatorName })
    : creatorName;
  const hasStages = page.plan.rolloutStageSummaries.length > 0;
  const showStateLine = Boolean(approval) || hasStages;

  return (
    <div
      className={cn("mt-1 text-sm text-control", LINES)}
      data-testid="plan-details"
    >
      {showSourceLine && (
        <div className={LINE} data-testid="plan-detail-line" data-line="source">
          <span className="sr-only">{t("plan.details.source")}</span>
          {release ? (
            <SourceParts source={release.vcsSource} />
          ) : (
            <span className={PART} />
          )}
        </div>
      )}
      <div className={LINE} data-testid="plan-detail-line" data-line="facts">
        <span className={SEPARATED} data-part="creator">
          {!isReleasePlan && (
            <span className="sr-only">{t("plan.details.created")}</span>
          )}
          <span className="truncate" title={creatorText}>
            {creatorText}
          </span>
        </span>
        {createdTimeMs !== undefined && (
          <span
            className={cn(SEPARATED, "text-control-light")}
            data-part="time"
          >
            <HumanizeTs tsMs={createdTimeMs} />
          </span>
        )}
        {(isReleasePlan ? releaseParts : changesParts)
          .filter(Boolean)
          .map((part, index) => (
            <span
              key={`${index}-${part}`}
              className={SEPARATED}
              data-part="changes"
            >
              {index === 0 && (
                <span className="sr-only">
                  {isReleasePlan
                    ? t("plan.details.release")
                    : t("plan.details.changes")}
                </span>
              )}
              <span className="truncate" title={part}>
                {part}
              </span>
            </span>
          ))}
        {labelsShown && page.issue && (
          <>
            <span className="sr-only">{t("plan.details.labels")}</span>
            <InlineLabels
              allowChange={allowChangeLabels}
              issueLabels={project?.issueLabels ?? []}
              labels={page.issue.labels || []}
              onUpdate={handleLabelsUpdate}
              required={
                page.issue.draft && (project?.forceIssueLabels ?? false)
              }
              inLine
            />
          </>
        )}
      </div>
      {showStateLine && (
        <div
          className={cn(LINE, "font-medium text-main")}
          data-testid="plan-detail-line"
          data-line="state"
        >
          {approval && (
            <span className={SEPARATED} data-part="approval">
              <span className="sr-only">{t("plan.details.review")}</span>
              <ApprovalIcon tone={approval.tone} />
              <span className="truncate" title={approval.text}>
                {approval.text}
              </span>
            </span>
          )}
          {hasStages && (
            <>
              <span className="sr-only">{t("plan.details.progress")}</span>
              <PlanStageParts
                plan={page.plan}
                partClassName={(index) => (index === 0 ? SEPARATED : PART)}
                arrowClassName={ARROW}
              />
            </>
          )}
        </div>
      )}
    </div>
  );
}

function ApprovalIcon({ tone }: { tone: Approval["tone"] }) {
  switch (tone) {
    case "done":
      return (
        <CircleCheck
          aria-hidden="true"
          className="size-4 shrink-0 text-success"
        />
      );
    case "rejected":
      return (
        <CircleX aria-hidden="true" className="size-4 shrink-0 text-error" />
      );
    case "waiting":
      return (
        <Clock
          aria-hidden="true"
          className="size-4 shrink-0 text-control-light"
        />
      );
    default:
      return (
        <CircleCheck
          aria-hidden="true"
          className="size-4 shrink-0 text-control-light"
        />
      );
  }
}

// The store keeps the files in the executor's order, newest last.
const newestVersion = (release: Release) => release.files.at(-1)?.version ?? "";

// The 72-character cap and the row's width cut the same place: before a
// PR's "#N", so the number stays.
function SourceTitle({ text, href }: { text: string; href?: string }) {
  const match = / #\d+$/.exec(text);
  const tail = match ? match[0] : "";
  const room = SOURCE_TITLE_LIMIT - tail.length;
  let head = tail ? text.slice(0, -tail.length) : text;
  if (head.length > room) head = `${head.slice(0, room)}…`;
  const content = (
    <>
      <span className="truncate">{head}</span>
      {tail && <span className="shrink-0 whitespace-pre">{tail}</span>}
    </>
  );
  return href ? (
    <a
      className="flex min-w-0 font-medium text-accent hover:underline"
      data-testid="source-part"
      href={href}
      rel="noopener noreferrer"
      target="_blank"
      title={text}
    >
      {content}
    </a>
  ) : (
    <span className="flex min-w-0" data-testid="source-part" title={text}>
      {content}
    </span>
  );
}

// A branch shortens before its end, the part a reader tells branches apart by.
function BranchPart({ branch }: { branch: string }) {
  if (branch.length <= BRANCH_END_LENGTH) {
    return (
      <span className="truncate" data-testid="source-part" title={branch}>
        {branch}
      </span>
    );
  }
  return (
    <span className="flex min-w-0" data-testid="source-part" title={branch}>
      <span className="truncate">{branch.slice(0, -BRANCH_END_LENGTH)}</span>
      <span className="shrink-0">{branch.slice(-BRANCH_END_LENGTH)}</span>
    </span>
  );
}

// A source part that shortens at its end keeps its full text on hover.
function TruncatedText({ text }: { text: string }) {
  return (
    <span className="truncate" data-testid="source-part" title={text}>
      {text}
    </span>
  );
}

function SourceParts({ source }: { source?: Release_VCSSource }) {
  const { t } = useTranslation();
  const row = buildSourceRow(source);
  if (!row) return null;
  const text = row.title ?? row.repoAtSha ?? row.href;
  return (
    <>
      {text && (
        <span className={SEPARATED}>
          <SourceTitle text={text} href={row.href} />
        </span>
      )}
      {row.author && (
        <span className={cn(SEPARATED, "text-control-light")}>
          <TruncatedText
            text={t("plan.details.by-author", { author: row.author })}
          />
        </span>
      )}
      {row.title && row.repoAtSha && (
        <span className={cn(SEPARATED, "text-control-light")}>
          <TruncatedText text={row.repoAtSha} />
        </span>
      )}
      {row.branch && (
        <span className={cn(SEPARATED, "text-control-light")}>
          <BranchPart branch={row.branch} />
        </span>
      )}
    </>
  );
}

function InlineLabels({
  allowChange,
  issueLabels,
  labels,
  onUpdate,
  required,
  inLine = false,
}: {
  allowChange: boolean;
  issueLabels: Array<{ color?: Color; value: string }>;
  labels: string[];
  onUpdate: (labels: string[]) => Promise<void> | void;
  /** The project forces issue labels, so submitting for review needs one. */
  required: boolean;
  /** Each chip and the add control are parts of the fact line. */
  inLine?: boolean;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [isUpdating, setIsUpdating] = useState(false);
  // A chip or the add control is a part of the line, with no separator.
  const partClass = inLine ? PART : undefined;

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
        className={partClass}
        data-part={inLine ? "label" : undefined}
      >
        <span className="inline-flex h-6 items-center gap-1 rounded-xs bg-control-bg px-1.5 text-xs text-control">
          <span
            className="size-2.5 shrink-0 rounded-sm"
            style={{
              backgroundColor: option?.color
                ? colorToHex(option.color)
                : undefined,
            }}
          />
          <span className="truncate" title={value}>
            {value}
          </span>
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
      </span>
    );
  });

  const triggerDisabled = !allowChange || isUpdating;

  const picker = (
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
  );

  return (
    <div className={inLine ? "contents" : "flex flex-wrap items-center gap-1"}>
      {chips}
      {(allowChange || !inLine) && (
        <span
          className={partClass}
          data-part={inLine ? "label-add" : undefined}
        >
          {picker}
        </span>
      )}
    </div>
  );
}
