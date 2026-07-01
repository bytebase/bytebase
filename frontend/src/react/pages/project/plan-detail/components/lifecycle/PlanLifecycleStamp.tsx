// Terminal lifecycle stamp shown at the far left, before the title — a status
// badge for the plan's final state (like GitHub's Open/Closed pill), not an
// action. Active states and advances render on the right via PlanLifecycleSlot.
import { Ban, CircleCheckBig } from "lucide-react";
import { useTranslation } from "react-i18next";
import { LifecycleStamp } from "./LifecycleStamp";
import type { PlanLifecycleHeaderState } from "./planLifecycleHeaderState";

export function PlanLifecycleStamp({
  state,
}: {
  state: PlanLifecycleHeaderState;
}) {
  const { t } = useTranslation();

  switch (state.kind) {
    case "closed":
      return (
        <LifecycleStamp>
          <Ban className="size-4" />
          {t("common.closed")}
        </LifecycleStamp>
      );
    case "deployed":
      return (
        <LifecycleStamp tone="success">
          <CircleCheckBig className="size-4" />
          {t("plan.lifecycle.deployed")}
        </LifecycleStamp>
      );
    default:
      return null;
  }
}
