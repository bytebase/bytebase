import { ChevronDown, ChevronUp, MessagesSquare } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";

// Touch has no hover and keeps focus on the tapped arrow, so coarse pointers
// get no ring: the flash on the target card is the feedback there.
const ARROW_CLASS =
  "h-auto w-7 rounded-none border-l border-control-border px-0 text-control-light hover:text-control focus-visible:ring-inset focus-visible:ring-offset-0 pointer-coarse:focus-visible:ring-0";

// Steps through the unresolved threads placed on the displayed statement.
// `count` is what the arrows can visit; `remainder` is how many unresolved
// threads of this change are not placed on this version.
export function StatementThreadWalker({
  announcement,
  count,
  onNext,
  onPrevious,
  remainder,
}: {
  announcement: string;
  count: number;
  onNext: () => void;
  onPrevious: () => void;
  remainder: number;
}) {
  const { t } = useTranslation();
  const title = [
    t("plan.review.thread.walker.count", { count }),
    remainder > 0
      ? t("plan.review.thread.walker.remainder", { count: remainder })
      : undefined,
  ]
    .filter(Boolean)
    .join(" · ");
  return (
    <div
      className="flex h-7 items-stretch overflow-hidden rounded-sm border border-control-border bg-background text-xs shadow-sm pointer-coarse:shadow-none"
      data-testid="thread-walker"
      title={title}
    >
      <span className="inline-flex items-center gap-1 px-2 font-medium text-accent tabular-nums">
        <MessagesSquare className="size-3.5" />
        {count}
      </span>
      <Button
        appearance="secondary"
        aria-label={t("plan.review.thread.walker.previous")}
        className={ARROW_CLASS}
        onClick={onPrevious}
        size="xs"
      >
        <ChevronUp className="size-4" />
      </Button>
      <Button
        appearance="secondary"
        aria-label={t("plan.review.thread.walker.next")}
        className={ARROW_CLASS}
        onClick={onNext}
        size="xs"
      >
        <ChevronDown className="size-4" />
      </Button>
      <span aria-live="polite" className="sr-only">
        {announcement}
      </span>
    </div>
  );
}
