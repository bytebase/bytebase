import { create } from "@bufbuild/protobuf";
import {
  RotateCcw,
  ShieldAlert,
  ShieldCheck,
  ThumbsUp,
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
import { router } from "@/router";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
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
  getIssueCommentType,
  IssueCommentType,
  useIssueCommentStore,
} from "@/store/modules/v1/issueComment";
import { RiskLevel, State } from "@/types/proto-es/v1/common_pb";
import type { Issue, IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import {
  Issue_ApprovalStatus,
  Issue_Approver_Status,
  ListIssueCommentsRequestSchema,
  RequestIssueRequestSchema,
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
import { extractIssueUID } from "@/utils/v1/issue/issue";
import { usePlanDetailContext } from "../context/PlanDetailContext";

type ApprovalStepStatus = "approved" | "rejected" | "current" | "pending";

// Stable empty array for the comments selector — useVueState's getter must
// return a cached reference when we want React to skip re-renders, otherwise
// useSyncExternalStore will treat every render as a snapshot change.
const EMPTY_COMMENTS: IssueComment[] = [];

export function PlanDetailSidebarApprovalFlow() {
  return <PlanDetailApprovalFlowContent mode="sidebar" />;
}

export function PlanDetailReviewApprovalFlow() {
  return <PlanDetailApprovalFlowContent mode="review" />;
}

export function PlanDetailApprovalFlow() {
  return <PlanDetailSidebarApprovalFlow />;
}

function PlanDetailApprovalFlowContent({
  mode,
}: {
  mode: "review" | "sidebar";
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const projectStore = useProjectV1Store();
  const projectIamPolicyStore = useProjectIamPolicyStore();
  const issueCommentStore = useIssueCommentStore();
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

  useEffect(() => {
    if (mode !== "review") {
      return;
    }
    if (!issue?.name) {
      return;
    }
    void issueCommentStore
      .listIssueComments(
        create(ListIssueCommentsRequestSchema, {
          parent: issue.name,
          pageSize: 100,
        })
      )
      .catch(() => undefined);
  }, [issue?.name, issueCommentStore, mode]);

  const comments = useVueState(() => {
    if (!issue) return EMPTY_COMMENTS;
    const list = issueCommentStore.getIssueComments(issue.name);
    // The store returns a fresh `[]` on cache miss; normalise to a stable
    // reference so useSyncExternalStore can short-circuit re-renders.
    return list.length > 0 ? list : EMPTY_COMMENTS;
  });
  const lastRejection = useMemo(() => {
    if (!issue || issue.approvalStatus !== Issue_ApprovalStatus.REJECTED) {
      return undefined;
    }
    for (let i = comments.length - 1; i >= 0; i--) {
      const comment = comments[i];
      if (getIssueCommentType(comment) === IssueCommentType.APPROVAL) {
        return {
          comment: comment.comment || "",
          creator: comment.creator,
        };
      }
    }
    return undefined;
  }, [comments, issue]);

  if (!issue) {
    return null;
  }

  const approvalSteps = issue.approvalTemplate?.flow?.roles ?? [];
  const statusTag = getStatusTag(issue, approvalSteps.length, t);
  const issueUID = extractIssueUID(issue.name);
  const riskLevelText =
    issue.riskLevel === RiskLevel.LOW
      ? t("issue.risk-level.low")
      : issue.riskLevel === RiskLevel.MODERATE
        ? t("issue.risk-level.moderate")
        : issue.riskLevel === RiskLevel.HIGH
          ? t("issue.risk-level.high")
          : "";
  const ruleText = [
    issue.approvalTemplate?.title?.trim(),
    issue.approvalTemplate?.description?.trim(),
  ]
    .filter(Boolean)
    .join(" · ");

  if (issue.approvalStatus === Issue_ApprovalStatus.CHECKING) {
    if (mode === "review") {
      return (
        <div className="flex flex-col">
          <div className="flex items-center gap-x-2 p-4 text-sm text-control-placeholder">
            <div className="h-4 w-4 animate-spin rounded-full border-2 border-control-border border-t-accent" />
            <span>
              {t("custom-approval.issue-review.generating-approval-flow")}
            </span>
          </div>
        </div>
      );
    }
    return (
      <div>
        <div className="flex w-full flex-row flex-wrap items-center gap-2">
          <h3 className="textinfolabel">{t("issue.approval-flow.self")}</h3>
          <FeatureBadge feature={PlanFeature.FEATURE_APPROVAL_WORKFLOW} />
          <ApprovalRiskLevelIcon
            riskLevel={issue.riskLevel}
            title={issue.approvalTemplate?.title?.trim()}
          />
        </div>
        <div className="mt-2 flex items-center gap-x-2 text-sm text-control-placeholder">
          <div className="h-4 w-4 animate-spin rounded-full border-2 border-control-border border-t-accent" />
          <span>
            {t("custom-approval.issue-review.generating-approval-flow")}
          </span>
        </div>
      </div>
    );
  }

  if (
    issue.approvalStatus === Issue_ApprovalStatus.SKIPPED ||
    approvalSteps.length === 0
  ) {
    if (mode === "review") {
      return (
        <div className="flex flex-col">
          <div className="p-4 text-sm text-control-placeholder">
            {t("custom-approval.approval-flow.skip")}
          </div>
        </div>
      );
    }
    return (
      <div>
        <div className="flex w-full flex-row flex-wrap items-center gap-2">
          <h3 className="textinfolabel">{t("issue.approval-flow.self")}</h3>
          <FeatureBadge feature={PlanFeature.FEATURE_APPROVAL_WORKFLOW} />
          <ApprovalRiskLevelIcon
            riskLevel={issue.riskLevel}
            title={issue.approvalTemplate?.title?.trim()}
          />
        </div>
        <div className="mt-2 flex items-center gap-x-1 text-sm text-control-placeholder">
          {t("custom-approval.approval-flow.skip")}
        </div>
      </div>
    );
  }

  if (mode === "sidebar") {
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
          <div className="mt-1 flex flex-col gap-y-4 pl-1">
            {approvalSteps.map((step, index) => (
              <ApprovalStepItem
                key={`${step}-${index}`}
                issue={issue}
                readonly={page.readonly}
                step={step}
                stepIndex={index}
                stepNumber={index + 1}
                totalSteps={approvalSteps.length}
              />
            ))}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col">
      {issue.approvalStatus === Issue_ApprovalStatus.REJECTED &&
        lastRejection && (
          <div className="mx-4 mt-3 rounded-sm border border-warning bg-warning/10 px-3 py-2 text-sm text-warning">
            <div className="font-medium">
              {t("custom-approval.issue-review.rejected-by")}{" "}
              {lastRejection.creator}
            </div>
            {lastRejection.comment && (
              <div className="mt-1">{lastRejection.comment}</div>
            )}
          </div>
        )}

      <div className="px-4 py-3">
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
          <div className="mt-1 flex flex-col gap-y-4 pl-1">
            {approvalSteps.map((step, index) => (
              <ApprovalStepItem
                key={`${step}-${index}`}
                issue={issue}
                readonly={page.readonly}
                step={step}
                stepIndex={index}
                stepNumber={index + 1}
                totalSteps={approvalSteps.length}
              />
            ))}
          </div>
        </div>
      </div>

      <div className="flex items-center gap-x-2 border-t px-4 py-2.5 text-sm text-control-placeholder">
        <ApprovalRiskLevelIcon
          riskLevel={issue.riskLevel}
          title={issue.approvalTemplate?.title?.trim()}
        />
        {riskLevelText && <span>{riskLevelText}</span>}
        {riskLevelText && ruleText && <span>·</span>}
        {ruleText && <span className="min-w-0 truncate">{ruleText}</span>}
        <div className="flex-1" />
        <button
          className="text-accent hover:underline"
          onClick={() =>
            void router.push({
              name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
              params: {
                issueId: issueUID,
                projectId: page.projectId,
              },
            })
          }
          type="button"
        >
          {t("common.issue")} #{issueUID} · {t("plan.view-discussion")} →
        </button>
      </div>
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
          <ThumbsUp className="h-3.5 w-3.5 text-white" />
        ) : status === "rejected" ? (
          <X className="h-3.5 w-3.5 text-white" />
        ) : status === "current" ? (
          <User className="h-3.5 w-3.5 text-white" />
        ) : (
          <span className="text-xs font-medium text-gray-600">
            {stepNumber}
          </span>
        )}
      </div>

      <div>
        <div className="text-sm font-medium text-gray-900">{roleName}</div>
        <div className="mt-1 text-sm text-gray-600">
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
                    className="gap-x-1.5 border border-accent/30 bg-accent/10 text-accent hover:bg-accent/15"
                    disabled={reRequesting}
                    onClick={() => {
                      void handleReRequestReview();
                    }}
                    size="xs"
                    variant="ghost"
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
                <div className="rounded-sm border border-yellow-600 bg-yellow-50 px-1 py-0.5 text-xs text-yellow-600">
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
      if (canceled || !next) return;
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

  if (!user) return null;

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

  if (users.length === 0) return null;

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
        {t("custom-approval.issue-review.and-n-other-users", {
          count: remainingCount,
          names,
        })}
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
  const currentUser = useCurrentUserV1().value;
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
  if (issue.approvalStatus === Issue_ApprovalStatus.APPROVED) {
    return {
      className: "bg-success/10 text-success",
      label: t("issue.table.approved"),
    };
  }
  if (issue.approvalStatus === Issue_ApprovalStatus.REJECTED) {
    return {
      className: "bg-warning/10 text-warning",
      label: t("common.rejected"),
    };
  }
  if (issue.approvalStatus === Issue_ApprovalStatus.PENDING) {
    return {
      className: "bg-accent/10 text-accent",
      label: t("common.under-review"),
    };
  }
  return undefined;
}

function useApprovalStep(issue: Issue, step: string, stepIndex: number) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const { patchState } = page;
  const currentUser = useCurrentUserV1().value;
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
  const projectIamPolicy = useVueState(() =>
    projectIamPolicyStore.getProjectIamPolicy(projectName)
  );
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

  const groupNamesKey = useMemo(() => {
    if (!projectIamPolicy) return "";
    const names: string[] = [];
    for (const binding of projectIamPolicy.bindings) {
      if (binding.role !== step || isBindingPolicyExpired(binding)) continue;
      for (const member of binding.members) {
        if (member.startsWith(groupBindingPrefix)) {
          names.push(member);
        }
      }
    }
    return [...new Set(names)].sort().join("\u0000");
  }, [projectIamPolicy, step]);
  const groupNames = useMemo(
    () => (groupNamesKey ? groupNamesKey.split("\u0000") : []),
    [groupNamesKey]
  );

  useEffect(() => {
    void groupStore.batchGetOrFetchGroups(groupNames).catch(() => undefined);
  }, [groupNames, groupStore]);

  const candidateEmailsKey = useMemo(() => {
    if (!projectIamPolicy) return "";
    const memberMap = memberMapToRolesInProjectIAM(projectIamPolicy, step);
    const candidates: string[] = [];
    for (const fullname of memberMap.keys()) {
      if (fullname.startsWith(userNamePrefix)) {
        candidates.push(fullname);
      }
    }
    return [...new Set(candidates)].sort().join("\u0000");
  }, [projectIamPolicy, step]);
  const candidateEmails = useMemo(
    () => (candidateEmailsKey ? candidateEmailsKey.split("\u0000") : []),
    [candidateEmailsKey]
  );

  const isCurrentUserInCandidates = useMemo(
    () => candidateEmails.includes(`${userNamePrefix}${currentUserEmail}`),
    [candidateEmails, currentUserEmail]
  );

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
      if (canceled) return;

      const next = users
        .filter(
          (user) =>
            user &&
            user.state === State.ACTIVE &&
            getAccountTypeByEmail(user.email) === AccountType.USER
        )
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
    if (reRequesting) return;

    try {
      setReRequesting(true);
      const response = await issueServiceClientConnect.requestIssue(
        create(RequestIssueRequestSchema, { name: issue.name })
      );
      patchState({ issue: response });
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
