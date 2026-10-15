import { create } from "@bufbuild/protobuf";
import {
  Check,
  RotateCcw,
  ShieldAlert,
  ShieldCheck,
  User,
  X,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { issueServiceClientConnect } from "@/api";
import { AccountTypeBadge } from "@/components/AccountTypeBadge";
import { FeatureBadge } from "@/components/FeatureBadge";
import { getAvatarColor, getInitials } from "@/components/UserAvatar";
import { Button } from "@/components/ui/button";
import { Tooltip } from "@/components/ui/tooltip";
import { useCurrentUser } from "@/hooks/useAppState";
import { principalLabel, usePrincipal } from "@/hooks/usePrincipal";
import { useProjectByName } from "@/hooks/useProjectByName";
import { displayRoleTitleFromList } from "@/lib/role";
import { cn } from "@/lib/utils";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { projectNamePrefix, userNamePrefix } from "@/stores/modules/v1/common";
import { ApprovalStatus, RiskLevel } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import {
  Issue_Approver_Status,
  RequestIssueRequestSchema,
  RetryIssueApprovalRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  type ApprovalCandidate,
  useApprovalCandidates,
} from "../../useApprovalCandidates";
import { useIssueDetailContext } from "../context/IssueDetailContext";

type ApprovalStepStatus = "approved" | "rejected" | "current" | "pending";

type PotentialApprover = Pick<ApprovalCandidate, "ineligibilities" | "user">;

export function IssueDetailApprovalFlow() {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const loadProjectIamPolicy = useAppStore(
    (state) => state.loadProjectIamPolicy
  );
  const projectName = `${projectNamePrefix}${page.projectId}`;
  const issue = page.issue;

  useEffect(() => {
    void useAppStore
      .getState()
      .getOrFetchProjectByName(projectName)
      .catch(() => undefined);
    void loadProjectIamPolicy(projectName).catch(() => undefined);
  }, [loadProjectIamPolicy, projectName]);

  if (!issue) {
    return null;
  }

  const approvalSteps = issue.approvalTemplate?.flow?.roles ?? [];
  const hasRollout = page.plan?.hasRollout ?? false;
  const statusTag = getStatusTag(issue, approvalSteps.length, t);

  return (
    <div>
      <div className="flex w-full flex-row flex-wrap items-center gap-2">
        <h3 className="textinfolabel">{t("issue.approval-flow.self")}</h3>
        <FeatureBadge feature={PlanFeature.FEATURE_APPROVAL_WORKFLOW} />
        <ApprovalRiskLevelIcon
          riskLevel={issue.riskLevel}
          title={issue.approvalTemplate?.title?.trim()}
        />
        <div className="grow" />
        {statusTag && (
          <span
            className={cn(
              "inline-flex items-center rounded-xs border border-transparent px-1.5 py-0.5 text-xs",
              statusTag.className
            )}
          >
            {statusTag.label}
          </span>
        )}
      </div>

      <div className="mt-2">
        {issue.approvalStatus === ApprovalStatus.CHECKING ? (
          <CheckingState issue={issue} />
        ) : approvalSteps.length > 0 ? (
          <div className="mt-1 flex flex-col gap-y-4 pl-1">
            {approvalSteps.map((step, index) => (
              <ApprovalStepItem
                key={`${step}-${index}`}
                issue={issue}
                readonly={page.readonly || hasRollout}
                step={step}
                stepIndex={index}
                stepNumber={index + 1}
                totalSteps={approvalSteps.length}
              />
            ))}
          </div>
        ) : (
          <div className="flex items-center gap-x-1 text-sm text-control-placeholder">
            {t("custom-approval.approval-flow.skip")}
          </div>
        )}
      </div>
    </div>
  );
}

/**
 * "Generating approval flow…" placeholder shown while the backend is
 * still computing an approval template. Includes a Retry button that
 * calls the backend's `RetryIssueApproval` RPC — the synchronous
 * post-create finding path swallows errors (e.g. CEL evaluation against
 * a malformed workspace approval rule) and there is no event-driven
 * retry for non-DATABASE_CHANGE issue types, so without this button a
 * stuck issue would stay in CHECKING indefinitely.
 */
function CheckingState({ issue }: { issue: Issue }) {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const [retrying, setRetrying] = useState(false);

  const handleRetry = async () => {
    if (retrying) return;
    setRetrying(true);
    try {
      const response = await issueServiceClientConnect.retryIssueApproval(
        create(RetryIssueApprovalRequestSchema, { name: issue.name })
      );
      page.patchState({ issue: response });
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: error instanceof Error ? error.message : String(error),
      });
    } finally {
      setRetrying(false);
    }
  };

  return (
    <div className="flex items-center gap-x-2 text-sm text-control-placeholder">
      <div className="h-4 w-4 animate-spin rounded-full border-2 border-control-border border-t-accent" />
      <span>{t("custom-approval.issue-review.generating-approval-flow")}</span>
      <Button
        className="gap-x-1.5"
        disabled={retrying}
        onClick={() => {
          void handleRetry();
        }}
        size="xs"
        appearance="outline"
      >
        <RotateCcw className="h-3 w-3" />
        {t("custom-approval.issue-review.retry-approval-finding")}
      </Button>
    </div>
  );
}

function ApprovalStepItem({
  issue,
  readonly,
  step,
  stepIndex,
  stepNumber,
  totalSteps,
}: {
  issue: Issue;
  readonly: boolean;
  step: string;
  stepIndex: number;
  stepNumber: number;
  totalSteps: number;
}) {
  const { t } = useTranslation();
  const {
    canReRequest,
    handleReRequestReview,
    potentialApprovers,
    reRequesting,
    roleName,
    showSelfApprovalTip,
    status,
    stepApprover,
  } = useApprovalStep(issue, step, stepIndex);

  return (
    <div className="relative pl-9">
      {stepIndex < totalSteps - 1 && (
        <div className="absolute left-[13px] top-7 bottom-[-16px] w-px bg-control-border" />
      )}

      <div
        className={cn(
          "absolute left-0 top-0 z-10 flex h-7 w-7 items-center justify-center rounded-full",
          status === "approved"
            ? "bg-success"
            : status === "rejected"
              ? "bg-warning"
              : status === "current"
                ? "bg-accent"
                : "bg-control-bg"
        )}
      >
        {status === "approved" ? (
          <Check className="h-3.5 w-3.5 text-accent-text" />
        ) : status === "rejected" ? (
          <X className="h-3.5 w-3.5 text-accent-text" />
        ) : status === "current" ? (
          <User className="h-3.5 w-3.5 text-accent-text" />
        ) : (
          <span className="text-xs font-medium text-control">{stepNumber}</span>
        )}
      </div>

      <div>
        <div className="text-sm font-medium text-main">{roleName}</div>
        <div className="mt-1 text-sm text-control-light">
          {status === "approved" && (
            <div className="flex flex-row flex-wrap items-start gap-x-1">
              <span className="shrink-0 text-xs leading-5">
                {t("custom-approval.issue-review.approved-by")}
              </span>
              {stepApprover?.principal && (
                <ApprovalUserText candidate={stepApprover.principal} />
              )}
            </div>
          )}

          {status === "rejected" && (
            <div className="flex flex-col gap-1">
              <div className="flex flex-row flex-wrap items-start gap-x-1">
                <span className="shrink-0 text-xs leading-5">
                  {t("custom-approval.issue-review.rejected-by")}
                </span>
                {stepApprover?.principal && (
                  <ApprovalUserText candidate={stepApprover.principal} />
                )}
              </div>
              {canReRequest && !readonly && (
                <div className="mt-1">
                  <Button
                    className="gap-x-1.5"
                    disabled={reRequesting}
                    onClick={() => {
                      void handleReRequestReview();
                    }}
                    size="xs"
                    appearance="outline"
                  >
                    <RotateCcw className="h-3 w-3" />
                    {t("custom-approval.issue-review.re-request-review")}
                  </Button>
                </div>
              )}
            </div>
          )}

          {status === "current" && !readonly && (
            <div className="flex flex-col gap-1">
              <PotentialApprovers users={potentialApprovers} />
              {showSelfApprovalTip && (
                <div className="rounded-sm border border-warning bg-warning/10 px-1 py-0.5 text-xs text-warning">
                  {t("custom-approval.issue-review.self-approval-not-allowed")}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function ApprovalUserText({ candidate }: { candidate: string }) {
  const principal = usePrincipal(candidate);
  return (
    <ApproverIdentity
      email={principal?.email}
      name={principalLabel(principal)}
    />
  );
}

// Avatar and name, with the account-type mark beneath the name: the sidebar is
// too narrow to hold both on one line.
function ApproverIdentity({
  email,
  name,
}: {
  email: string | undefined;
  name: string;
}) {
  return (
    <span className="inline-flex min-w-0 max-w-full items-start gap-1.5">
      <span
        className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full text-xs font-medium text-accent-text"
        style={{ backgroundColor: getAvatarColor(name) }}
      >
        {getInitials(name)}
      </span>
      <span className="flex min-w-0 flex-col items-start">
        <span className="text-xs leading-5 text-control">{name}</span>
        <AccountTypeBadge compact email={email} />
      </span>
    </span>
  );
}

function PotentialApprovers({ users }: { users: PotentialApprover[] }) {
  const { t } = useTranslation();

  if (users.length === 0) {
    return null;
  }

  if (users.length <= 3) {
    return (
      <div className="flex flex-col items-start gap-1">
        {users.map((user) => (
          <ApprovalCandidateRow key={user.user.name} candidate={user} />
        ))}
      </div>
    );
  }

  const visibleUsers = users.slice(0, 3);
  const names = visibleUsers
    .map(
      (candidate) => candidate.user.title || candidate.user.email.split("@")[0]
    )
    .join(", ");
  const remainingCount = users.length - visibleUsers.length;
  const triggerText = t("custom-approval.issue-review.and-n-other-users", {
    count: remainingCount,
    names,
  });

  return (
    <Tooltip
      content={
        <div className="flex max-w-xs flex-col gap-y-1">
          {users.map((user) => (
            <ApprovalCandidateRow
              candidate={user}
              key={user.user.name}
              showEmail
            />
          ))}
        </div>
      }
      side="bottom"
    >
      <span className="cursor-pointer text-xs text-control-light hover:text-accent">
        {triggerText}
      </span>
    </Tooltip>
  );
}

function ApprovalCandidateRow({
  candidate,
  showEmail = false,
}: {
  candidate: PotentialApprover;
  showEmail?: boolean;
}) {
  const { t } = useTranslation();
  const currentUser = useCurrentUser();
  const { user } = candidate;
  const isCurrentUser = currentUser?.name === user.name;

  return (
    <div className="inline-flex items-start gap-1.5">
      <ApproverIdentity email={user.email} name={principalLabel(user)} />
      {isCurrentUser && (
        <span className="rounded-xs border border-transparent bg-success/10 px-1.5 py-0.5 text-xs text-success">
          {t("common.you")}
        </span>
      )}
      {showEmail && (
        <span className="text-xs leading-5 text-control-light">
          {user.email}
        </span>
      )}
      {candidate.ineligibilities.includes("last-plan-editor") && (
        <span className="text-xs leading-5 text-warning">
          {t("plan.review.candidate-last-changed-plan")}
        </span>
      )}
    </div>
  );
}

function ApprovalRiskLevelIcon({
  riskLevel,
  title,
}: {
  riskLevel: RiskLevel;
  title?: string;
}) {
  const { t } = useTranslation();

  if (riskLevel === RiskLevel.RISK_LEVEL_UNSPECIFIED) {
    return null;
  }

  const riskLevelText =
    riskLevel === RiskLevel.LOW
      ? t("issue.risk-level.low")
      : riskLevel === RiskLevel.MODERATE
        ? t("issue.risk-level.moderate")
        : t("issue.risk-level.high");

  return (
    <Tooltip
      content={
        <div className="flex flex-col gap-1">
          <div>
            <span>{riskLevelText}</span>
            <span className="ml-1 opacity-60">
              ({t("issue.risk-level.self")})
            </span>
          </div>
          {title && <div className="text-sm opacity-80">{title}</div>}
        </div>
      }
    >
      {riskLevel === RiskLevel.LOW ? (
        <ShieldCheck className="h-4 w-4 text-success" />
      ) : riskLevel === RiskLevel.MODERATE ? (
        <ShieldAlert className="h-4 w-4 text-warning" />
      ) : (
        <ShieldAlert className="h-4 w-4 text-error" />
      )}
    </Tooltip>
  );
}

function getStatusTag(
  issue: Issue,
  approvalStepCount: number,
  t: ReturnType<typeof useTranslation>["t"]
) {
  if (approvalStepCount === 0) {
    return undefined;
  }
  if (issue.approvalStatus === ApprovalStatus.APPROVED) {
    return {
      className: "bg-success/10 text-success",
      label: t("issue.table.approved"),
    };
  }
  if (issue.approvalStatus === ApprovalStatus.REJECTED) {
    return {
      className: "bg-warning/10 text-warning",
      label: t("common.rejected"),
    };
  }
  if (issue.approvalStatus === ApprovalStatus.PENDING) {
    return {
      className: "bg-accent/10 text-accent",
      label: t("common.under-review"),
    };
  }
  return undefined;
}

function useApprovalStep(issue: Issue, step: string, stepIndex: number) {
  const { t } = useTranslation();
  const roleList = useAppStore((state) => state.roleList);
  const page = useIssueDetailContext();
  const currentUser = useCurrentUser();
  // subscribe to re-render on project cache change
  const projectsByName = useAppStore((s) => s.projectsByName);
  const [reRequesting, setReRequesting] = useState(false);
  const projectName = `${projectNamePrefix}${page.projectId}`;
  const currentUserEmail = currentUser?.email ?? "";
  const project = useProjectByName(projectName);
  void projectsByName;
  const stepApprover = issue.approvers[stepIndex];

  const status = useMemo<ApprovalStepStatus>(() => {
    if (stepApprover?.status === Issue_Approver_Status.APPROVED) {
      return "approved";
    }
    if (stepApprover?.status === Issue_Approver_Status.REJECTED) {
      return "rejected";
    }
    for (let i = 0; i < stepIndex; i++) {
      const previousApprover = issue.approvers[i];
      if (
        !previousApprover ||
        (previousApprover.status !== Issue_Approver_Status.APPROVED &&
          previousApprover.status !== Issue_Approver_Status.REJECTED)
      ) {
        return "pending";
      }
      if (previousApprover.status === Issue_Approver_Status.REJECTED) {
        return "pending";
      }
    }
    return "current";
  }, [issue.approvers, stepApprover, stepIndex]);

  const roleName = useMemo(() => {
    return step
      ? displayRoleTitleFromList(step, roleList)
      : t("custom-approval.approval-flow.node.approver");
  }, [roleList, step, t]);

  const canReRequest = useMemo(() => {
    return (
      issue.creator === `${userNamePrefix}${currentUserEmail}` &&
      stepApprover?.status === Issue_Approver_Status.REJECTED
    );
  }, [currentUserEmail, issue.creator, stepApprover?.status]);

  const { candidates: potentialApprovers, isCurrentUserRoleMember } =
    useApprovalCandidates(
      issue,
      page.projectId,
      status === "current" ? step : "",
      page.plan?.lastPlanEditor ?? ""
    );

  const showSelfApprovalTip =
    status === "current" &&
    !project.allowSelfApproval &&
    issue.creator === `${userNamePrefix}${currentUserEmail}` &&
    isCurrentUserRoleMember;

  const handleReRequestReview = async () => {
    if (reRequesting) {
      return;
    }

    try {
      setReRequesting(true);
      const response = await issueServiceClientConnect.requestIssue(
        create(RequestIssueRequestSchema, {
          name: issue.name,
        })
      );
      page.patchState({ issue: response });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("custom-approval.issue-review.re-request-review-success"),
      });
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    } finally {
      setReRequesting(false);
    }
  };

  return {
    canReRequest,
    handleReRequestReview,
    potentialApprovers,
    reRequesting,
    roleName,
    showSelfApprovalTip,
    status,
    stepApprover,
  };
}
