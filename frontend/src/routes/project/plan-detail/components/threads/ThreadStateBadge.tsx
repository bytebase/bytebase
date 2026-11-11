import { Ban, CircleCheck, History } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";

// A thread's state as text with an icon and no fill. "Outdated" takes the
// warning color, since it tells the reader to doubt the comment; the others
// stay gray. Open, current threads are the default and carry no mark.
export function ThreadStateBadge({
  state,
}: {
  state: "resolved" | "outdated" | "unavailable";
}) {
  const { t } = useTranslation();
  const Icon =
    state === "resolved" ? CircleCheck : state === "outdated" ? History : Ban;
  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center gap-x-1 text-xs",
        state === "outdated" ? "text-warning-hover" : "text-control"
      )}
      data-testid="thread-state"
    >
      <Icon className="size-3" />
      {state === "resolved"
        ? t("plan.review.thread.resolved")
        : state === "outdated"
          ? t("plan.review.thread.anchor.outdated")
          : t("plan.review.thread.anchor.statement-unavailable")}
    </span>
  );
}
