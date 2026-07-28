// Resolver for the header's primary advance (BYT-9925, BYT-9936).
//
// Pure functions that turn plan/project state into the data the advance control
// renders: everything unresolved, and — separately — the one thing the reader
// may deliberately override. Sibling to planLifecycleHeaderState.ts, which
// answers "which control does the header show"; this answers "what does that
// control say when pressed".
import { getPlanCheckSummaryWithFallback } from "@/lib/plan/check";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

type T = (key: string, options?: Record<string, unknown>) => string;

// One unresolved condition standing between the reader and the next lifecycle
// state. `wait` clears on its own (a running check); `fix` is theirs to resolve.
// The distinction is what stops a running check from reading as a chore.
export interface AdvanceBlocker {
  id: string;
  message: string;
  kind: "fix" | "wait";
}

// A deliberate override: the action can proceed, but only once the reader says
// so. `verb` names what confirming does, so the control reads correctly out of
// context.
export interface AdvanceDecision {
  headline: string;
  body: string;
  verb: string;
}

export interface AdvanceState {
  blockers: AdvanceBlocker[];
  decision?: AdvanceDecision;
}

// Stable empty values, so a lifecycle state that renders no advance does not
// hand out a fresh array identity on every render.
export const NO_BLOCKERS: AdvanceBlocker[] = [];
export const NO_ADVANCE: AdvanceState = { blockers: NO_BLOCKERS };

const fix = (id: string, message: string): AdvanceBlocker => ({
  id,
  kind: "fix",
  message,
});

export const getCreatePlanBlockers = ({
  emptySpecCount,
  permissionReason,
  title,
  t,
}: {
  emptySpecCount: number;
  permissionReason?: string;
  title: string;
  t: T;
}): AdvanceBlocker[] => {
  const blockers: AdvanceBlocker[] = [];
  if (!title.trim()) {
    blockers.push(fix("title", t("plan.title-required")));
  }
  if (emptySpecCount > 0) {
    blockers.push(fix("statement", t("plan.navigator.statement-empty")));
  }
  if (permissionReason) {
    blockers.push(fix("permission", permissionReason));
  }
  return blockers;
};

export const getSubmitReviewAdvance = ({
  emptySpecCount,
  isEditing,
  permissionReason,
  plan,
  project,
  selectedLabelCount,
  t,
}: {
  emptySpecCount: number;
  isEditing: boolean;
  permissionReason?: string;
  plan: Plan;
  project: Pick<Project, "enforceSqlReview" | "forceIssueLabels">;
  selectedLabelCount: number;
  t: T;
}): AdvanceState => {
  const blockers: AdvanceBlocker[] = [];
  const checks = getPlanCheckSummaryWithFallback(
    [],
    plan.planCheckRunStatusCount
  );
  // AVAILABLE (queued, not yet started) is a backend plan-check status that the
  // shared summary does not count, and a queued check still has to finish.
  const queued = plan.planCheckRunStatusCount?.AVAILABLE ?? 0;

  if (emptySpecCount > 0) {
    blockers.push(fix("statement", t("plan.navigator.statement-empty")));
  }
  if (plan.specs.some((spec) => spec.config?.case === "exportDataConfig")) {
    blockers.push(
      fix("data-export", t("issue.data-export.creation-not-supported"))
    );
  }
  if (checks.running > 0 || queued > 0) {
    blockers.push({
      id: "checks-running",
      kind: "wait",
      message: t(
        "custom-approval.issue-review.disallow-approve-reason.some-task-checks-are-still-running"
      ),
    });
  }

  // Failed checks are evaluated once and land on exactly one side: a blocker
  // where the project forbids the override, otherwise the reader's to
  // authorize. Keeping both outcomes in one branch stops them drifting apart.
  // The override is offered only for plans that are purely database changes.
  const isPureDatabaseChange =
    plan.specs.length > 0 &&
    plan.specs.every((spec) => spec.config?.case === "changeDatabaseConfig");
  let decision: AdvanceDecision | undefined;
  if (checks.error > 0) {
    if (project.enforceSqlReview) {
      blockers.push(
        fix(
          "checks-failed",
          t(
            "custom-approval.issue-review.disallow-approve-reason.some-task-checks-didnt-pass"
          )
        )
      );
    } else if (isPureDatabaseChange) {
      decision = {
        body: t("issue.checks-warning-hint"),
        headline: t("plan.lifecycle.gate-checks-failed"),
        verb: t("plan.submit-review-anyway"),
      };
    }
  }

  if (project.forceIssueLabels && selectedLabelCount === 0) {
    blockers.push(fix("labels", t("plan.labels-required-for-review")));
  }
  if (isEditing) {
    blockers.push(
      fix("editing", t("plan.editor.save-changes-before-continuing"))
    );
  }
  if (permissionReason) {
    blockers.push(fix("permission", permissionReason));
  }

  return { blockers, decision };
};
