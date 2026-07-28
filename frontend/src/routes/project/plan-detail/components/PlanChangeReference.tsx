import { DatabaseIcon, Download, FolderTree } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import type {
  PlanChangeReferenceIcon,
  PlanChangeReference as PlanChangeReferenceModel,
  PlanChangeReferenceResources,
} from "../utils/changeReference";
import {
  derivePlanChangeReference,
  splitPlanChangeReferenceLabel,
} from "../utils/changeReference";

export const PLAN_CHANGE_REFERENCE_TOOLTIP_CLASS =
  "max-w-[min(24rem,calc(100vw-1rem))] px-3 py-2.5";

type PlanChangeReferenceDensity = "activity" | "tab";

const DENSITY_CLASS: Record<
  PlanChangeReferenceDensity,
  { label: string; reference: string }
> = {
  activity: {
    label: "max-w-72",
    reference: "max-w-80",
  },
  tab: {
    label: "max-w-56",
    reference: "max-w-64",
  },
};

export function PlanChangeReference({
  ariaHidden = false,
  className,
  density = "activity",
  reference,
}: {
  ariaHidden?: boolean;
  className?: string;
  density?: PlanChangeReferenceDensity;
  reference: PlanChangeReferenceModel;
}) {
  const measureRef = useRef<HTMLSpanElement>(null);
  const densityClass = DENSITY_CLASS[density];
  const overflowKey = [
    density,
    reference.label,
    reference.countLabel ?? "",
    reference.showIndex ? "1" : "0",
    reference.showIndexWithCountLabel ? "1" : "0",
  ].join("\u0000");
  const [overflowState, setOverflowState] = useState({
    key: "",
    value: false,
  });
  const isOverflowing =
    overflowState.key === overflowKey && overflowState.value;
  const useCountLabel = Boolean(isOverflowing && reference.countLabel);
  const displayLabel =
    useCountLabel && reference.countLabel
      ? reference.countLabel
      : reference.label;
  const showIndex = useCountLabel
    ? reference.showIndexWithCountLabel
    : reference.showIndex;

  useEffect(() => {
    const element = measureRef.current;
    if (!element) {
      setOverflowState({ key: overflowKey, value: false });
      return;
    }
    const measure = () => {
      const overflowing = element.scrollWidth > element.clientWidth;
      setOverflowState((current) =>
        current.key === overflowKey && current.value === overflowing
          ? current
          : { key: overflowKey, value: overflowing }
      );
    };
    measure();
    if (typeof ResizeObserver === "undefined") {
      return;
    }
    const observer = new ResizeObserver(measure);
    observer.observe(element);
    return () => observer.disconnect();
  }, [overflowKey]);

  const content = (
    <span
      aria-hidden={ariaHidden || undefined}
      aria-label={ariaHidden ? undefined : reference.accessibleLabel}
      className={cn(
        "inline-flex w-full min-w-0 items-center gap-1 leading-5",
        densityClass.reference,
        className
      )}
      data-plan-change-reference={reference.specId}
      data-plan-change-reference-label={displayLabel}
      data-plan-change-reference-overflow={isOverflowing || undefined}
    >
      {showIndex && (
        <span className="shrink-0 text-sm font-normal tabular-nums text-control-placeholder">
          {reference.index}
        </span>
      )}
      <span className="mr-0.5 shrink-0 text-control-light">
        <ChangeTypeIcon icon={reference.icon} />
      </span>
      <MiddleEllipsisLabel className={densityClass.label} text={displayLabel} />
    </span>
  );

  return (
    <span className={cn("inline-grid min-w-0", densityClass.reference)}>
      <span
        aria-hidden="true"
        className={cn(
          "invisible col-start-1 row-start-1 inline-flex min-w-0 items-center gap-1 overflow-hidden leading-5",
          className
        )}
        data-plan-change-reference-measure
      >
        {reference.showIndex && (
          <span className="shrink-0 text-sm font-normal tabular-nums text-control-placeholder">
            {reference.index}
          </span>
        )}
        <span className="mr-0.5 shrink-0 text-control-light">
          <ChangeTypeIcon icon={reference.icon} />
        </span>
        <span
          ref={measureRef}
          className={cn(
            "min-w-0 overflow-hidden whitespace-nowrap",
            densityClass.label
          )}
        >
          {reference.label}
        </span>
      </span>
      <span className="col-start-1 row-start-1 inline-flex min-w-0">
        {isOverflowing ? (
          <Tooltip
            content={<PlanChangeReferenceTooltip reference={reference} />}
            delayDuration={250}
            popupClassName={PLAN_CHANGE_REFERENCE_TOOLTIP_CLASS}
          >
            {content}
          </Tooltip>
        ) : (
          content
        )}
      </span>
    </span>
  );
}

export function PlanSpecChangeReference({
  className,
  resources,
  siblings,
  spec,
}: {
  className?: string;
  resources: PlanChangeReferenceResources;
  siblings: Plan_Spec[];
  spec: Plan_Spec;
}) {
  const { t } = useTranslation();
  const index = siblings.findIndex((sibling) => sibling.id === spec.id);
  if (index < 0) {
    return null;
  }
  const reference = derivePlanChangeReference({
    index,
    resources,
    siblings,
    spec,
    t,
  });
  return <PlanChangeReference className={className} reference={reference} />;
}

export function PlanChangeReferenceTooltip({
  reference,
}: {
  reference: PlanChangeReferenceModel;
}) {
  const { t } = useTranslation();
  return (
    <div className="flex w-80 max-w-full flex-col gap-1.5">
      <div className="flex items-center gap-1.5 text-main-text/70">
        <ChangeTypeIcon icon={reference.icon} />
        <span>
          {t("plan.spec.change-reference.tooltip-title", {
            index: reference.index,
          })}
        </span>
      </div>
      <div
        className="wrap-break-word text-sm font-medium leading-5 text-main-text"
        dir="auto"
      >
        {reference.fullLabel}
      </div>
      {reference.countLabel && reference.countLabel !== reference.fullLabel && (
        <div
          className="wrap-break-word border-main-text/15 border-t pt-1.5 text-[11px] leading-4 text-main-text/70"
          dir="auto"
        >
          {reference.countLabel}
        </div>
      )}
    </div>
  );
}

function ChangeTypeIcon({ icon }: { icon: PlanChangeReferenceIcon }) {
  if (icon === "database-group") {
    return <FolderTree className="size-3.5" />;
  }
  if (icon === "export") {
    return <Download className="size-3.5" />;
  }
  return <DatabaseIcon className="size-3.5" />;
}

function MiddleEllipsisLabel({
  className,
  text,
}: {
  className: string;
  text: string;
}) {
  const { prefix, suffix } = splitPlanChangeReferenceLabel(text);
  if (!suffix) {
    return (
      <span className={cn("min-w-0 truncate", className)} dir="auto">
        {prefix}
      </span>
    );
  }
  return (
    <span className={cn("flex min-w-0", className)} dir="auto">
      <span className="min-w-0 truncate">{prefix}</span>
      <span className="shrink-0">{suffix}</span>
    </span>
  );
}
