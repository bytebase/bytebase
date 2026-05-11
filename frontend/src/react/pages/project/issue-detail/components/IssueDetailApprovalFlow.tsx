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
import { issueServiceClientConnect } from "@/connect";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { getAvatarColor, getInitials } from "@/react/components/UserAvatar";
import { Button } from "@/react/components/ui/button";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import {
  pushNotification,
  useCurrentUserV1,
  useGroupStore,
  useProjectIamPolicyStore,
  useProjectV1Store,
  useUserStore,
} from "@/store";
import { projectNamePrefix, userNamePrefix } from "@/store/modules/v1/common";
import {
  ApprovalStatus,
  RiskLevel,
  State,
} from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import {
  Issue_Approver_Status,
  RequestIssueRequestSchema,
  RetryIssueApprovalRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { User as UserMessage } from "@/types/proto-es/v1/user_service_pb";
import {
  AccountType,
  getAccountTypeByEmail,
  groupBindingPrefix,
} from "@/types/v1/user";
import {
  displayRoleTitle,
  ensureUserFullName,
  isBindingPolicyExpired,
  memberMapToRolesInProjectIAM,
} from "@/utils";
import { useIssueDetailContext } from "../context/IssueDetailContext";

type ApprovalStepStatus = "approved" | "rejected" | "current" | "pending";

export function IssueDetailApprovalFlow() {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const projectStore = useProjectV1Store();
  const projectIamPolicyStore = useProjectIamPolicyStore();
  const projectName = `${projectNamePrefix}${page.projectId}`;
  const issue = page.issue;

  useEffect(() => {
    void projectStore
      .getOrFetchProjectByName(projectName)
      .catch(() => undefined);
    void projectIamPolicyStore
      .getOrFetchProjectIamPolicy(projectName)
      .catch(() => undefined);
  }, [projectIamPolicyStore, projectName, projectStore]);

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
              "inline-flex items-center rounded-full px-2 py-0.5 text-xs",
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
        variant="outline"
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
          <Check className="h-3.5 w-3.5 text-white" />
        ) : status === "rejected" ? (
          <X className="h-3.5 w-3.5 text-white" />
        ) : status === "current" ? (
          <User className="h-3.5 w-3.5 text-white" />
        ) : (
          <span className="text-xs font-medium text-control">{stepNumber}</span>
        )}
      </div>

      <div>
        <div className="text-sm font-medium text-main">{roleName}</div>
        <div className="mt-1 text-sm text-control-light">
          {status === "approved" && (
            <div className="flex flex-row items-center gap-1">
              <span className="text-xs">
                {t("custom-approval.issue-review.approved-by")}
              </span>
              {stepApprover?.principal && (
                <ApprovalUserText candidate={stepApprover.principal} />
              )}
            </div>
          )}

          {status === "rejected" && (
            <div className="flex flex-col gap-1">
              <div className="flex flex-row items-center gap-1">
                <span className="text-xs">
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
                    variant="outline"
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
  const userStore = useUserStore();
  const [user, setUser] = useState<UserMessage>();

  useEffect(() => {
    let canceled = false;

    const load = async () => {
      const next = await userStore.getOrFetchUserByIdentifier({
        identifier: candidate,
      });
      if (canceled || !next) {
        return;
      }
      if (
        getAccountTypeByEmail(next.email) !== AccountType.USER ||
        next.state !== State.ACTIVE
      ) {
        setUser(undefined);
        return;
      }
      setUser(next);
    };

    void load();

    return () => {
      canceled = true;
    };
  }, [candidate, userStore]);

  if (!user) {
    return null;
  }

  const displayName = user.title || user.email.split("@")[0];
  return (
    <span className="inline-flex items-center gap-1">
      <span
        className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full text-[10px] font-medium text-white"
        style={{ backgroundColor: getAvatarColor(displayName) }}
      >
        {getInitials(displayName)}
      </span>
      <span className="text-xs text-control">{displayName}</span>
    </span>
  );
}

function PotentialApprovers({ users }: { users: UserMessage[] }) {
  const { t } = useTranslation();

  if (users.length === 0) {
    return null;
  }

  if (users.length <= 3) {
    return (
      <div className="flex flex-col items-start gap-1">
        {users.map((user) => (
          <ApprovalCandidateRow key={user.name} user={user} />
        ))}
      </div>
    );
  }

  const visibleUsers = users.slice(0, 3);
  const names = visibleUsers
    .map((user) => user.title || user.email.split("@")[0])
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
            <ApprovalCandidateRow key={user.name} showEmail user={user} />
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
  showEmail = false,
  user,
}: {
  showEmail?: boolean;
  user: UserMessage;
}) {
  const { t } = useTranslation();
  const currentUser = useVueState(() => useCurrentUserV1().value);
  const displayName = user.title || user.email.split("@")[0];
  const isCurrentUser = currentUser?.name === user.name;

  return (
    <div className="inline-flex items-center gap-1.5">
      <span
        className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full text-[10px] font-medium text-white"
        style={{ backgroundColor: getAvatarColor(displayName) }}
      >
        {getInitials(displayName)}
      </span>
      <span className="text-xs text-control">{displayName}</span>
      {isCurrentUser && (
        <span className="rounded-full bg-success/10 px-1.5 py-0.5 text-[10px] text-success">
          {t("common.you")}
        </span>
      )}
      {showEmail && (
        <span className="text-xs text-control-light">{user.email}</span>
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
  const page = useIssueDetailContext();
  const currentUser = useVueState(() => useCurrentUserV1().value);
  const groupStore = useGroupStore();
  const projectStore = useProjectV1Store();
  const projectIamPolicyStore = useProjectIamPolicyStore();
  const userStore = useUserStore();
  const [potentialApprovers, setPotentialApprovers] = useState<UserMessage[]>(
    []
  );
  const [reRequesting, setReRequesting] = useState(false);
  const projectName = `${projectNamePrefix}${page.projectId}`;
  const currentUserEmail = currentUser?.email ?? "";
  const project = useVueState(() => projectStore.getProjectByName(projectName));
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
      ? displayRoleTitle(step)
      : t("custom-approval.approval-flow.node.approver");
  }, [step, t]);

  const canReRequest = useMemo(() => {
    return (
      issue.creator === `${userNamePrefix}${currentUserEmail}` &&
      stepApprover?.status === Issue_Approver_Status.REJECTED
    );
  }, [currentUserEmail, issue.creator, stepApprover?.status]);

  const groupNamesKey = useVueState(() => {
    const policy = projectIamPolicyStore.getProjectIamPolicy(projectName);
    const names: string[] = [];
    for (const binding of policy.bindings) {
      if (binding.role !== step || isBindingPolicyExpired(binding)) {
        continue;
      }
      for (const member of binding.members) {
        if (member.startsWith(groupBindingPrefix)) {
          names.push(member);
        }
      }
    }
    return [...new Set(names)].sort().join("\u0000");
  });
  const groupNames = useMemo(() => {
    return groupNamesKey ? groupNamesKey.split("\u0000") : [];
  }, [groupNamesKey]);

  useEffect(() => {
    void groupStore.batchGetOrFetchGroups(groupNames).catch(() => undefined);
  }, [groupNames, groupStore]);

  const candidateEmailsKey = useVueState(() => {
    const policy = projectIamPolicyStore.getProjectIamPolicy(projectName);
    const memberMap = memberMapToRolesInProjectIAM(policy, step);
    const candidates: string[] = [];
    for (const fullname of memberMap.keys()) {
      if (fullname.startsWith(userNamePrefix)) {
        candidates.push(fullname);
      }
    }
    return [...new Set(candidates)].sort().join("\u0000");
  });
  const candidateEmails = useMemo(() => {
    return candidateEmailsKey ? candidateEmailsKey.split("\u0000") : [];
  }, [candidateEmailsKey]);

  const isCurrentUserInCandidates = useMemo(() => {
    return candidateEmails.includes(`${userNamePrefix}${currentUserEmail}`);
  }, [candidateEmails, currentUserEmail]);

  const filteredCandidateEmails = useMemo(() => {
    if (
      !project.allowSelfApproval &&
      issue.creator === `${userNamePrefix}${currentUserEmail}`
    ) {
      return candidateEmails.filter(
        (candidate) => candidate !== `${userNamePrefix}${currentUserEmail}`
      );
    }
    return candidateEmails;
  }, [
    candidateEmails,
    currentUserEmail,
    issue.creator,
    project.allowSelfApproval,
  ]);

  const showSelfApprovalTip = useMemo(() => {
    return (
      status === "current" &&
      !project.allowSelfApproval &&
      issue.creator === `${userNamePrefix}${currentUserEmail}` &&
      isCurrentUserInCandidates
    );
  }, [
    currentUserEmail,
    isCurrentUserInCandidates,
    issue.creator,
    project.allowSelfApproval,
    status,
  ]);

  useEffect(() => {
    let canceled = false;

    const load = async () => {
      if (status !== "current" || filteredCandidateEmails.length === 0) {
        setPotentialApprovers([]);
        return;
      }

      const users = await userStore.batchGetOrFetchUsers(
        filteredCandidateEmails.map(ensureUserFullName)
      );
      if (canceled) {
        return;
      }

      const next = users
        .filter((user) => {
          return (
            user &&
            user.state === State.ACTIVE &&
            getAccountTypeByEmail(user.email) === AccountType.USER
          );
        })
        .sort((left, right) => {
          if (left.email === currentUserEmail) return -1;
          if (right.email === currentUserEmail) return 1;
          return left.title.localeCompare(right.title);
        });
      setPotentialApprovers(next);
    };

    void load();

    return () => {
      canceled = true;
    };
  }, [currentUserEmail, filteredCandidateEmails, status, userStore]);

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
