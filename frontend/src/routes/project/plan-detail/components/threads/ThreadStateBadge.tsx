import { CircleCheck } from "lucide-react";
import { useTranslation } from "react-i18next";

// Marks a resolved thread in quiet gray text. Open is a thread's default
// state and carries no mark.
export function ThreadStateBadge() {
  const { t } = useTranslation();
  return (
    <span
      className="inline-flex shrink-0 items-center gap-x-1 text-xs text-control-light"
      data-testid="thread-state"
    >
      <CircleCheck className="size-3" />
      {t("plan.review.thread.resolved")}
    </span>
  );
}
