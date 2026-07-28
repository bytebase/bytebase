// The single lifecycle slot rendered next to the plan title. It switches on the
// resolved header state and shows one of: an advance action, a read-only status
// control, a progress indicator, or a terminal stamp. Draft states (create,
// ready-for-review) are rendered by PlanDetailHeader because they are coupled to
// the title/create flow; everything else lives here.
import { Loader2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import { FrontierStatusStamp, RunStageAction } from "./DeployStageActions";
import { LifecycleStamp } from "./LifecycleStamp";
import { PlanStatusAction } from "./PlanStatusAction";
import type { PlanLifecycleHeaderState } from "./planLifecycleHeaderState";
import { ReviewYourTurnAction } from "./ReviewYourTurnAction";

export function PlanLifecycleSlot({
  state,
}: {
  state: PlanLifecycleHeaderState;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();

  switch (state.kind) {
    case "incomplete":
      return (
        <LifecycleStamp size="md" tone="error">
          {t("plan.lifecycle.incomplete")}
        </LifecycleStamp>
      );
    case "review-generating":
      // The approval flow is still being generated — show the Review affordance
      // disabled with a loading spinner; no click action until it resolves.
      return (
        <Button className="gap-x-1.5" disabled>
          <Loader2 className="size-4 animate-spin" />
          {t("plan.review.action")}
        </Button>
      );
    case "review-your-turn":
      // review-your-turn only resolves while an open issue exists.
      return page.issue ? <ReviewYourTurnAction issue={page.issue} /> : null;
    case "plan-status":
      return page.issue ? (
        <PlanStatusAction issue={page.issue} reason={state.reason} />
      ) : null;
    case "preparing-rollout":
      return (
        <LifecycleStamp size="md">
          <Loader2 className="size-3.5 animate-spin" />
          {t("plan.lifecycle.preparing-rollout")}
        </LifecycleStamp>
      );
    case "run-stage":
      return <RunStageAction stage={state.stage} />;
    case "running-stage":
      return <FrontierStatusStamp stage={state.stage} />;
    default:
      // create / ready-for-review render in the header; closed / deployed are
      // terminal stamps rendered left of the title and none renders nothing.
      return null;
  }
}
