import {
  CircleAlert,
  CircleDashed,
  Clock3,
  Loader2,
  MessagesSquare,
  Send,
  TriangleAlert,
  UserRound,
} from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { displayRoleTitleFromList } from "@/lib/role";
import { useAppStore } from "@/stores/app";
import { isThreadRoot } from "@/stores/app/issueComment";
import {
  type Issue,
  type IssueComment,
  IssueComment_ThreadState,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import { useFocusThread } from "../hooks/useFocusThread";
import type { PlanReviewState } from "../hooks/usePlanReviewState";
import { usePlanDetailStoreApi } from "../shared/stores/usePlanDetailStore";
import { focusPlanPhase } from "../shell/focusPhase";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import type { AutomatedReviewRow, PlanGate } from "../utils/planGates";
import { reviewerLabel, reviewGateLabel } from "../utils/reviewLabels";
import { BypassAndDeploySheet, useBypassDeploy } from "./review/bypassDeploy";

const NO_COMMENTS: IssueComment[] = [];

// Before the rollout exists, Deploy says what the automatic rollout waits on.
// Each line leads to where it is settled; the bypass sits beside them.
export function PlanDetailDeployWaiting({
  issue,
  plan,
  state,
}: {
  issue: Issue;
  plan: Plan;
  state: PlanReviewState;
}) {
  const { t } = useTranslation();
  const { gates } = state;
  // Not a gate: the server rolls out past it, so it sits apart from what the
  // rollout waits on, but an approver should look at it.
  const overridden = state.rows.filter((row) => row.findings.overridden > 0);
  return (
    <div
      className="mt-1.5 flex flex-col gap-y-2 rounded-sm border bg-background px-3 py-2.5"
      data-testid="deploy-waiting"
    >
      {gates.length === 0 ? (
        <div className="flex items-center gap-x-2 text-sm text-control">
          <Loader2 className="size-4 animate-spin text-control-light" />
          {t("plan.deploy-waiting.creating-rollout")}
        </div>
      ) : (
        <>
          <span className="text-xs text-control-light">
            {t("plan.deploy-waiting.waiting-on")}
          </span>
          <ul className="flex flex-col gap-y-1.5">
            {gates.map((gate) => (
              <GateLine gate={gate} key={gateKey(gate)} />
            ))}
          </ul>
        </>
      )}
      {!issue.draft && <DeployWaitingFooter issue={issue} plan={plan} />}
      {overridden.length > 0 && (
        <div className="flex flex-col gap-y-1.5 border-t pt-2">
          <span className="text-xs text-control-light">
            {t("plan.deploy-waiting.worth-a-look")}
          </span>
          {overridden.map((row) => (
            <OverriddenLine key={row.reviewer} row={row} />
          ))}
        </div>
      )}
    </div>
  );
}

const gateKey = (gate: PlanGate) =>
  gate.kind === "review" ? `review:${gate.row.reviewer}` : gate.kind;

function GateLine({ gate }: { gate: PlanGate }) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const storeApi = usePlanDetailStoreApi();
  const roleList = useAppStore((state) => state.roleList);
  const { focusInStatement } = useFocusThread();
  const comments = useAppStore((state) =>
    page.issue ? state.getIssueComments(page.issue.name) : NO_COMMENTS
  );

  let icon: ReactNode;
  let label: string;
  let onShow: (() => void) | undefined;
  switch (gate.kind) {
    case "submit":
      icon = <Send className="size-4 text-control-light" />;
      label = t("plan.deploy-waiting.submit");
      break;
    case "approval":
      icon = <UserRound className="size-4 text-control-light" />;
      label = gate.rejected
        ? t("plan.deploy-waiting.approval-rejected")
        : gate.role
          ? t("plan.deploy-waiting.approval-from", {
              role: displayRoleTitleFromList(gate.role, roleList),
            })
          : t("plan.deploy-waiting.approval");
      onShow = () => focusPlanPhase("review", page.expandPhase);
      break;
    case "review": {
      const { row } = gate;
      icon =
        row.conclusion === "open" ? (
          <CircleAlert
            className={
              row.findings.p0 > 0 ? "size-4 text-error" : "size-4 text-warning"
            }
          />
        ) : row.conclusion === "running" || row.conclusion === "loading" ? (
          <Loader2 className="size-4 animate-spin text-control-light" />
        ) : (
          <CircleDashed className="size-4 text-control-light" />
        );
      label = reviewGateLabel(t, row);
      onShow = () => {
        focusPlanPhase("review", page.expandPhase);
        storeApi.getState().requestReviewRow(row.reviewer);
      };
      break;
    }
    case "discussion":
      icon = <MessagesSquare className="size-4 text-warning" />;
      label = t("plan.deploy-waiting.open-discussions", { count: gate.open });
      onShow = () => {
        const first = comments.find(
          (comment) =>
            isThreadRoot(comment) &&
            !comment.reviewMetadata &&
            comment.threadState === IssueComment_ThreadState.OPEN
        );
        if (first) focusInStatement(first);
      };
      break;
  }

  return (
    <li className="flex items-center gap-x-2 text-sm text-main">
      <span className="flex size-4 shrink-0 items-center">{icon}</span>
      <span className="min-w-0 flex-1">{label}</span>
      {onShow && (
        <Button
          appearance="link"
          className="text-control-light"
          onClick={onShow}
          size="xs"
        >
          {t("plan.deploy-waiting.show")}
        </Button>
      )}
    </li>
  );
}

function OverriddenLine({ row }: { row: AutomatedReviewRow }) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const storeApi = usePlanDetailStoreApi();
  return (
    <div
      className="flex items-center gap-x-2 text-sm text-main"
      data-testid="deploy-overridden"
    >
      <TriangleAlert className="size-4 shrink-0 text-warning" />
      <span className="min-w-0 flex-1">
        {t("plan.deploy-waiting.overridden", {
          count: row.findings.overridden,
          reviewer: reviewerLabel(t, row.reviewer),
        })}
      </span>
      <Button
        appearance="link"
        className="text-control-light"
        onClick={() => {
          focusPlanPhase("review", page.expandPhase);
          storeApi.getState().requestReviewRow(row.reviewer);
        }}
        size="xs"
      >
        {t("plan.deploy-waiting.show")}
      </Button>
    </div>
  );
}

function DeployWaitingFooter({ issue, plan }: { issue: Issue; plan: Plan }) {
  const { t } = useTranslation();
  const bypass = useBypassDeploy(issue, plan);
  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-1 border-t pt-2">
      <span className="flex min-w-0 flex-1 items-center gap-x-1.5 text-xs text-control-light">
        <Clock3 className="size-3.5 shrink-0" />
        {t("plan.deploy-waiting.auto-create")}
      </span>
      {bypass.weight !== "none" && (
        <Button
          appearance={bypass.weight === "button" ? "outline" : "link"}
          className={
            bypass.weight === "link" ? "text-control-light" : undefined
          }
          disabled={bypass.creating}
          onClick={bypass.triggerDeploy}
          size="xs"
        >
          {bypass.creating && <Loader2 className="size-3.5 animate-spin" />}
          {t("plan.review.footer.bypass-and-deploy")}
        </Button>
      )}
      <BypassAndDeploySheet bypass={bypass} issue={issue} />
    </div>
  );
}
