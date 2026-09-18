import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { FullDateTime } from "@/components/HumanizeTs";
import { Button } from "@/components/ui/button";
import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { displayableInstantMs } from "@/utils/datetime";
import type { Section } from "./types";

// The full reading is offered only for an instant the formatters can render.
const fullTime = (timeMs: number | undefined) => {
  const instantMs =
    timeMs === undefined ? undefined : displayableInstantMs(timeMs);
  return instantMs === undefined ? undefined : (
    <FullDateTime tsMs={instantMs} />
  );
};

const ITEM_HEIGHT = 20;
const MAX_VISIBLE_ITEMS = 10;
const MAX_RENDERED_ITEMS = 50;

export interface SectionContentProps {
  section: Section;
  indent?: boolean;
  datasetKey?: string;
}

export function SectionContent({
  section,
  indent = false,
  datasetKey,
}: SectionContentProps) {
  const { t } = useTranslation();
  const [showAllItems, setShowAllItems] = useState(false);

  useEffect(() => {
    setShowAllItems(false);
  }, [datasetKey, section.id]);

  const visibleItems =
    showAllItems || section.items.length <= MAX_RENDERED_ITEMS
      ? section.items
      : section.items.slice(0, MAX_RENDERED_ITEMS);
  const hiddenItemCount = section.items.length - visibleItems.length;

  return (
    <div
      className="overflow-auto border-block-border border-t bg-control-bg/50"
      style={{ maxHeight: `${MAX_VISIBLE_ITEMS * ITEM_HEIGHT}px` }}
    >
      {visibleItems.map((item, index) => (
        <div
          key={item.key}
          className={cn(
            "flex items-start gap-x-2 py-0.5 hover:bg-control-bg",
            indent ? "px-6" : "px-3",
            index > 0 && "border-block-border border-t"
          )}
        >
          <span className="w-6 shrink-0 text-right text-control-placeholder tabular-nums">
            {index + 1}
          </span>
          <span className="shrink-0 text-control-placeholder tabular-nums">
            <Tooltip content={fullTime(item.timeMs)}>{item.time}</Tooltip>
          </span>
          {item.relativeTime ? (
            <span className="shrink-0 text-control-placeholder tabular-nums">
              {item.relativeTime}
            </span>
          ) : null}
          <span className={cn("shrink-0", item.levelClass)}>
            {item.levelIndicator}
          </span>
          <span className={cn("min-w-0 break-words", item.detailClass)}>
            {item.detail}
          </span>
          <span className="ml-auto flex shrink-0 items-center gap-x-2">
            {item.duration ? (
              <span className="text-info tabular-nums">{item.duration}</span>
            ) : null}
            {item.affectedRows !== undefined ? (
              <span className="text-control-placeholder">
                {item.affectedRows} {t("task.affected-rows")}
              </span>
            ) : null}
          </span>
        </div>
      ))}
      {hiddenItemCount > 0 ? (
        <Button
          type="button"
          appearance="secondary"
          size="sm"
          className="w-full rounded-none border-block-border border-t text-control-light hover:bg-control-bg-hover hover:text-control"
          onClick={() => setShowAllItems(true)}
        >
          <span>{t("common.load-more")}</span>
          <span className="tabular-nums">({hiddenItemCount})</span>
        </Button>
      ) : null}
    </div>
  );
}

export default SectionContent;
