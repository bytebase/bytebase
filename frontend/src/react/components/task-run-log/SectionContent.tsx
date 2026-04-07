import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { cn } from "@/react/lib/utils";
import type { Section } from "./types";

const ITEM_HEIGHT = 20;
const MAX_VISIBLE_ITEMS = 10;
const MAX_RENDERED_ITEMS = 50;

export interface SectionContentProps {
  section: Section;
  indent?: boolean;
}

export function SectionContent({
  section,
  indent = false,
}: SectionContentProps) {
  const { t } = useTranslation();
  const [showAllItems, setShowAllItems] = useState(false);
  const visibleItems =
    showAllItems || section.items.length <= MAX_RENDERED_ITEMS
      ? section.items
      : section.items.slice(0, MAX_RENDERED_ITEMS);
  const hiddenItemCount = section.items.length - visibleItems.length;

  return (
    <div
      className="bg-gray-50 border-t border-gray-100 overflow-auto"
      style={{ maxHeight: `${MAX_VISIBLE_ITEMS * ITEM_HEIGHT}px` }}
    >
      {visibleItems.map((item, index) => (
        <div
          key={item.key}
          className={cn(
            "flex items-start gap-x-2 py-0.5 hover:bg-gray-100",
            indent ? "px-6" : "px-3",
            index > 0 && "border-t border-gray-100"
          )}
        >
          <span className="w-6 shrink-0 text-right tabular-nums text-gray-300">
            {index + 1}
          </span>
          <span className="shrink-0 tabular-nums text-gray-400">
            {item.time}
          </span>
          {item.relativeTime ? (
            <span className="shrink-0 tabular-nums text-gray-300">
              {item.relativeTime}
            </span>
          ) : null}
          <span className={cn("shrink-0", item.levelClass)}>
            {item.levelIndicator}
          </span>
          <span className={cn("break-all", item.detailClass)}>
            {item.detail}
          </span>
          <span className="ml-auto flex shrink-0 items-center gap-x-2">
            {item.duration ? (
              <span className="tabular-nums text-blue-500">
                {item.duration}
              </span>
            ) : null}
            {item.affectedRows !== undefined ? (
              <span className="text-gray-400">
                {item.affectedRows} {t("task.affected-rows")}
              </span>
            ) : null}
          </span>
        </div>
      ))}
      {hiddenItemCount > 0 ? (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className={cn(
            "flex w-full items-center justify-center gap-x-2 rounded-none border-t border-gray-100 px-3 py-1.5 text-gray-500 hover:bg-gray-100 hover:text-gray-700",
            indent ? "px-6" : "px-3"
          )}
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
