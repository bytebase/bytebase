import { CircleCheck } from "lucide-react";
import { useTranslation } from "react-i18next";

// Marks a resolved thread in quiet text with no fill. Open is a thread's
// default state and carries no mark.
export function ThreadStateBadge() {
  const { t } = useTranslation();
  return (
    <span
      className="inline-flex shrink-0 items-center gap-x-1 text-xs text-control"
      data-testid="thread-state"
    >
      <CircleCheck className="size-3" />
      {t("plan.review.thread.resolved")}
    </span>
  );
}
