import { Check, User, X } from "lucide-react";
import { type ReactNode, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { getAvatarColor, getInitials } from "@/react/components/UserAvatar";
import { Badge } from "@/react/components/ui/badge";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/react/components/ui/popover";
import { displayRoleTitleFromList } from "@/react/lib/role";
import { cn } from "@/react/lib/utils";
import { useAppStore } from "@/react/stores/app";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { unknownUser } from "@/types";
import { ApprovalStatus } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { Issue_Approver_Status } from "@/types/proto-es/v1/issue_service_pb";
import type { User as UserProto } from "@/types/proto-es/v1/user_service_pb";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import {
  type ApprovalStepStatus,
  computeApprovalFlowLayout,
} from "./approvalFlowLayout";
import { useApprovalCandidates } from "./useApprovalCandidates";

export interface FlowStep {
  index: number;
  role: string;
  status: ApprovalStepStatus;
  approver?: string; // principal of the recorded decision
}

export function deriveSteps(issue: Issue): FlowStep[] {
  const roles = issue.approvalTemplate?.flow?.roles ?? [];
  return roles.map((role, index) => {
    const approver = issue.approvers[index];
    let status: ApprovalStepStatus = "pending";
    if (approver?.status === Issue_Approver_Status.APPROVED) {
      status = "approved";
    } else if (approver?.status === Issue_Approver_Status.REJECTED) {
      status = "rejected";
    } else {
      const blocked = roles.slice(0, index).some((_, i) => {
        const prev = issue.approvers[i];
        return prev?.status !== Issue_Approver_Status.APPROVED;
      });
      status = blocked ? "pending" : "current";
    }
    return { index, role, status, approver: approver?.principal };
  });
}

export function ReviewApprovalFlow({ issue }: { issue: Issue }) {
  const { t } = useTranslation();
  const hostRef = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(0);

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;
    const observer = new ResizeObserver((entries) => {
      setWidth(entries[0]?.contentRect.width ?? 0);
    });
    observer.observe(host);
    return () => observer.disconnect();
  }, []);

  const steps = useMemo(() => deriveSteps(issue), [issue]);
  const layout = useMemo(
    () =>
      // width is 0 on first paint before the ResizeObserver fires; fall back
      // to a wide value so the initial layout is horizontal and corrects once
      // the real width arrives.
      computeApprovalFlowLayout(
        steps.map((s) => s.status),
        width || 9999
      ),
    [steps, width]
  );

  // A skipped approval — or a resolved template with no approver roles — has no
  // flow to render. Show the "no approval required" note here so every caller
  // (the review section and the bypass confirm sheet) stays consistent, instead
  // of each guarding the empty case on its own (BYT-9745).
  if (issue.approvalStatus === ApprovalStatus.SKIPPED || steps.length === 0) {
    return (
      <div className="px-4 py-3 text-sm text-control-placeholder">
        {t("custom-approval.approval-flow.skip")}
      </div>
    );
  }

  return (
    <div ref={hostRef} className="min-w-0 px-4 py-3">
      {layout.kind === "vertical" ? (
        <VerticalFlow issue={issue} steps={steps} />
      ) : (
        <HorizontalFlow
          foldedApproved={layout.foldedApproved}
          issue={issue}
          namedPending={layout.namedPending}
          steps={steps}
        />
      )}
    </div>
  );
}

function HorizontalFlow({
  foldedApproved,
  issue,
  namedPending,
  steps,
}: {
  foldedApproved: number;
  issue: Issue;
  namedPending: number;
  steps: FlowStep[];
}) {
  const anchorIndex = steps.findIndex(
    (s) => s.status === "current" || s.status === "rejected"
  );
  const approved = steps.filter((s) => s.status === "approved");
  const foldedApprovedSteps = approved.slice(0, foldedApproved);
  const namedSteps = steps.filter(
    (s) =>
      !foldedApprovedSteps.includes(s) &&
      (anchorIndex === -1 || s.index <= anchorIndex + namedPending)
  );
  const foldedPendingSteps =
    anchorIndex === -1 ? [] : steps.slice(anchorIndex + 1 + namedPending);

  // Each item carries a stable key tied to its identity (chip name or step
  // index) so React reconciles correctly when the fold counts change with the
  // viewport — an array index key would graft a node onto the wrong slot.
  const items: { key: string; node: ReactNode }[] = [];
  if (foldedApprovedSteps.length > 0) {
    items.push({
      key: "approved-chip",
      node: <ApprovedChip steps={foldedApprovedSteps} />,
    });
  }
  for (const step of namedSteps) {
    items.push({
      key: `step-${step.index}`,
      node: <FlowNode issue={issue} step={step} />,
    });
  }
  if (foldedPendingSteps.length > 0) {
    items.push({
      key: "pending-chip",
      node: <PendingChip issue={issue} steps={foldedPendingSteps} />,
    });
  }

  return (
    <div className="flex min-w-0 items-start gap-x-2 overflow-hidden">
      {items.map((item, i) => (
        <div key={item.key} className="flex min-w-0 items-start gap-x-2">
          {i > 0 && (
            <div
              aria-hidden
              className="mt-3 h-px w-8 shrink-0 bg-control-border"
            />
          )}
          {item.node}
        </div>
      ))}
    </div>
  );
}

function VerticalFlow({ issue, steps }: { issue: Issue; steps: FlowStep[] }) {
  return (
    <div className="flex flex-col">
      {steps.map((step, i) => (
        <FlowNode
          key={step.index}
          isLast={i === steps.length - 1}
          issue={issue}
          step={step}
          vertical
        />
      ))}
    </div>
  );
}

function StatusDot({
  status,
  index,
}: {
  status: ApprovalStepStatus;
  index: number;
}) {
  return (
    <div
      className={cn(
        "flex size-6 shrink-0 items-center justify-center rounded-full text-xs font-medium",
        status === "approved" && "bg-success text-white",
        status === "rejected" && "bg-error text-white",
        status === "current" && "bg-accent text-white",
        status === "pending" && "bg-control-bg text-control"
      )}
    >
      {status === "approved" && <Check className="size-3.5" />}
      {status === "rejected" && <X className="size-3.5" />}
      {status === "current" && <User className="size-3.5" />}
      {status === "pending" && index}
    </div>
  );
}

function FlowNode({
  issue,
  step,
  vertical = false,
  isLast = false,
}: {
  issue: Issue;
  step: FlowStep;
  vertical?: boolean;
  isLast?: boolean;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const roleList = useAppStore((state) => state.roleList);
  const approverUser = useAppStore((state) =>
    step.approver ? state.getUserByIdentifier(step.approver) : undefined
  );
  const roleName = displayRoleTitleFromList(step.role, roleList);

  const subtitle =
    step.status === "approved"
      ? t("plan.review.approval-flow.approved-by", {
          user: (approverUser ?? unknownUser(step.approver ?? "")).title,
        })
      : step.status === "rejected"
        ? t("plan.review.approval-flow.rejected-by", {
            user: (approverUser ?? unknownUser(step.approver ?? "")).title,
          })
        : undefined;

  return (
    <div
      className={cn(
        "flex min-w-0 items-start gap-x-2",
        !vertical && "shrink-0"
      )}
    >
      {vertical ? (
        // Timeline rail: the status dot plus a connector line dropping to the
        // next node — the vertical counterpart of HorizontalFlow's between-node
        // connectors. The rail stretches to the row height (self-stretch) so the
        // flex-1 line reaches the next dot; spacing comes from the text's
        // padding-bottom rather than a container gap, so the line stays unbroken.
        <div className="flex flex-col items-center self-stretch">
          <StatusDot index={step.index + 1} status={step.status} />
          {!isLast && <div className="w-px flex-1 bg-control-border" />}
        </div>
      ) : (
        <StatusDot index={step.index + 1} status={step.status} />
      )}
      <div className={cn("min-w-0", vertical && !isLast && "pb-3")}>
        <div className="flex items-center gap-x-1.5">
          <span className="truncate text-sm font-medium text-main">
            {roleName}
          </span>
          {step.status === "current" && (
            <Badge className="px-1.5 py-0 text-[10px]" variant="secondary">
              {t("plan.review.approval-flow.current")}
            </Badge>
          )}
        </div>
        {subtitle ? (
          <div className="mt-1 h-4 truncate text-xs leading-4 text-control-light">
            {subtitle}
          </div>
        ) : step.status === "current" || step.status === "pending" ? (
          <NodeReviewers
            issue={issue}
            projectId={page.projectId}
            role={step.role}
          />
        ) : null}
      </div>
    </div>
  );
}

// The reviewer line under a current/pending node: an avatar stack + "N
// reviewers" count, where the whole row is a hover target that reveals the full
// reviewer list (BYT-9711). Rendered for future stages too, not just current.
function NodeReviewers({
  issue,
  projectId,
  role,
}: {
  issue: Issue;
  projectId: string;
  role: string;
}) {
  const { t } = useTranslation();
  // Whether the project IAM policy is in cache yet — used to tell "still
  // loading" (show nothing) apart from "genuinely no eligible reviewers".
  const policyLoaded = useAppStore(
    (state) =>
      state.projectPoliciesByName[`${projectNamePrefix}${projectId}`] !==
      undefined
  );
  const { candidates } = useApprovalCandidates(issue, projectId, role);
  if (candidates.length === 0) {
    // No eligible reviewers means this stage can never be approved — surface it
    // explicitly rather than rendering nothing (BYT-9711). Stay silent until the
    // policy resolves so it doesn't flash during load.
    if (!policyLoaded) return null;
    return (
      <div className="mt-1 flex h-4 items-center text-xs leading-4 text-control-placeholder">
        {t("plan.review.approval-flow.no-reviewers")}
      </div>
    );
  }
  const visible = candidates.slice(0, 3);
  const overflow = candidates.length - visible.length;
  // A role can resolve to thousands of candidates; render a bounded, scrollable
  // slice in the popover and summarize the rest rather than mounting every row.
  const MAX_LISTED = 50;
  const listed = candidates.slice(0, MAX_LISTED);
  const remaining = candidates.length - listed.length;

  return (
    <Popover>
      <PopoverTrigger
        delay={100}
        nativeButton={false}
        openOnHover
        render={
          <div className="mt-1 flex h-4 w-fit cursor-default items-center gap-x-1.5">
            <div className="flex items-center -space-x-1">
              {visible.map((user) => (
                <InitialsAvatar
                  className="size-4 text-[9px] ring-2 ring-white"
                  key={user.name}
                  user={user}
                />
              ))}
              {overflow > 0 && (
                <span className="flex size-4 items-center justify-center rounded-full bg-control-bg text-[9px] font-medium text-control ring-2 ring-white">
                  +{overflow}
                </span>
              )}
            </div>
            <span className="text-xs text-control-light">
              {t("plan.review.approval-flow.n-reviewers", {
                count: candidates.length,
              })}
            </span>
          </div>
        }
      />
      <PopoverContent align="start" className="w-60" side="bottom">
        <div className="flex max-h-64 flex-col gap-y-2 overflow-y-auto">
          {listed.map((user) => (
            <div className="flex items-center gap-x-2" key={user.name}>
              <InitialsAvatar className="size-6 text-[10px]" user={user} />
              <div className="min-w-0">
                <div className="truncate text-sm text-main">
                  {user.title || user.email.split("@")[0]}
                </div>
                <div className="truncate text-xs text-control-light">
                  {user.email}
                </div>
              </div>
            </div>
          ))}
        </div>
        {remaining > 0 && (
          <div className="mt-2 border-t pt-1.5 text-xs text-control-placeholder">
            {t("plan.review.approval-flow.and-n-more", { count: remaining })}
          </div>
        )}
      </PopoverContent>
    </Popover>
  );
}

function ApprovedChip({ steps }: { steps: FlowStep[] }) {
  const { t } = useTranslation();
  return (
    <Popover>
      <PopoverTrigger
        delay={100}
        nativeButton={false}
        openOnHover
        render={
          <div className="flex shrink-0 cursor-default items-start gap-x-2">
            <div className="flex size-6 shrink-0 items-center justify-center rounded-full bg-success text-white">
              <Check className="size-3.5" />
            </div>
            <div className="min-w-0">
              <span className="text-sm font-medium text-control">
                {t("plan.review.approval-flow.n-approved", { n: steps.length })}
              </span>
              <div className="mt-1 flex h-4 items-center">
                <ChipAvatars principals={steps.map((s) => s.approver ?? "")} />
              </div>
            </div>
          </div>
        }
      />
      <PopoverContent align="start" className="w-64" side="bottom">
        <div className="flex flex-col gap-y-2.5">
          {steps.map((step) => (
            <ApprovedPopoverRow key={step.index} step={step} />
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
}

function ApprovedPopoverRow({ step }: { step: FlowStep }) {
  const { t } = useTranslation();
  const roleList = useAppStore((state) => state.roleList);
  const approverUser = useAppStore((state) =>
    step.approver ? state.getUserByIdentifier(step.approver) : undefined
  );
  const approver = approverUser ?? unknownUser(step.approver ?? "");
  const roleName = displayRoleTitleFromList(step.role, roleList);
  return (
    <div className="flex items-start gap-x-2">
      <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-success text-white">
        <Check className="size-3" />
      </span>
      <div className="min-w-0 flex-1">
        <div className="truncate text-sm font-medium text-main">{roleName}</div>
        <div className="mt-0.5 flex items-center gap-x-1 text-xs text-control-light">
          <InitialsAvatar
            className="size-4 shrink-0 text-[8px]"
            user={approver}
          />
          <span className="truncate">
            {t("plan.review.approval-flow.approved-by", {
              user: approver.title,
            })}
          </span>
        </div>
      </div>
    </div>
  );
}

// Small circular initials avatar. `className` carries the per-call size/ring
// variant; the common geometry + color derivation lives here.
function InitialsAvatar({
  className,
  user,
}: {
  className: string;
  user: UserProto;
}) {
  const name = user.title || user.email.split("@")[0];
  return (
    <span
      className={cn(
        "flex items-center justify-center rounded-full font-medium text-white",
        className
      )}
      style={{ backgroundColor: getAvatarColor(name) }}
    >
      {getInitials(name)}
    </span>
  );
}

function ChipAvatars({ principals }: { principals: string[] }) {
  const getUserByIdentifier = useAppStore((state) => state.getUserByIdentifier);
  return (
    <span className="flex items-center -space-x-1">
      {principals.slice(0, 3).map((principal, i) => (
        <InitialsAvatar
          className="size-4 text-[9px] ring-1 ring-white"
          key={`${principal}-${i}`}
          user={getUserByIdentifier(principal) ?? unknownUser(principal)}
        />
      ))}
    </span>
  );
}

function PendingChip({ issue, steps }: { issue: Issue; steps: FlowStep[] }) {
  const { t } = useTranslation();
  return (
    <Popover>
      <PopoverTrigger
        delay={100}
        nativeButton={false}
        openOnHover
        render={
          <div className="flex shrink-0 cursor-default items-start gap-x-2">
            <div className="size-6 shrink-0 rounded-full border-2 border-dashed border-control-border" />
            <span className="text-sm font-medium text-control">
              {t("plan.review.approval-flow.n-pending", { n: steps.length })}
            </span>
          </div>
        }
      />
      <PopoverContent align="start" className="w-64" side="bottom">
        <div className="flex flex-col gap-y-2.5">
          {steps.map((step) => (
            <PendingPopoverRow issue={issue} key={step.index} step={step} />
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
}

// One row inside the folded-pending popover: step number, role, candidate
// avatars, and reviewer count — mirrors a live flow node on the light surface.
function PendingPopoverRow({ issue, step }: { issue: Issue; step: FlowStep }) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const roleList = useAppStore((state) => state.roleList);
  const { candidates } = useApprovalCandidates(
    issue,
    page.projectId,
    step.role
  );
  const roleName = displayRoleTitleFromList(step.role, roleList);
  const visible = candidates.slice(0, 4);
  const overflow = candidates.length - visible.length;

  return (
    <div className="flex items-start gap-x-2">
      <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-control-bg text-[10px] font-medium text-control">
        {step.index + 1}
      </span>
      <div className="min-w-0 flex-1">
        <div className="truncate text-sm font-medium text-main">{roleName}</div>
        {candidates.length > 0 && (
          <div className="mt-1 flex items-center gap-x-1.5">
            <div className="flex items-center -space-x-1">
              {visible.map((user) => (
                <InitialsAvatar
                  className="size-4 text-[8px] ring-2 ring-background"
                  key={user.name}
                  user={user}
                />
              ))}
              {overflow > 0 && (
                <span className="flex size-4 items-center justify-center rounded-full bg-control-bg text-[8px] text-control ring-2 ring-background">
                  +{overflow}
                </span>
              )}
            </div>
            <span className="text-xs text-control-light">
              {t("plan.review.approval-flow.n-reviewers", {
                count: candidates.length,
              })}
            </span>
          </div>
        )}
      </div>
    </div>
  );
}
