import { CircleCheck } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";

// An open thread needs someone, so it carries the colored badge; a resolved
// one is quiet gray text.
export function ThreadStateBadge({ resolved }: { resolved: boolean }) {
  const { t } = useTranslation();
  if (resolved) {
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
  return (
    <Badge className="shrink-0" data-testid="thread-state" variant="attention">
      {t("plan.review.thread.open")}
    </Badge>
  );
}
