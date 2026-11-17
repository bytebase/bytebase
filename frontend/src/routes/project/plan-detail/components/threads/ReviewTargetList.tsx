import { type Range, useVirtualizer } from "@tanstack/react-virtual";
import type { TFunction } from "i18next";
import { type RefObject, useCallback, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { EnvironmentBadge } from "@/components/EnvironmentLabel";
import { Button } from "@/components/ui/button";
import { SearchInput } from "@/components/ui/search-input";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app";
import { nullEnvironment } from "@/types/v1/environment";
import { extractDatabaseResourceName } from "@/utils/v1/database";
import { PlanTargetDisplay } from "../PlanTargetDisplay";
import { type TargetGroup, targetGroupKey } from "./productionTargets";

// A list with more databases than this gets a search box.
export const TARGET_SEARCH_THRESHOLD = 10;

// Headers and rows are one line each, so they share one fixed height.
const ROW_HEIGHT = 28;

// The name a group goes by: its environment's title, or what it lacks.
export const targetGroupLabel = (t: TFunction, group: TargetGroup): string => {
  switch (group.kind) {
    case "environment":
      return group.environment.title;
    case "unassigned":
      return t("common.unassigned");
    case "unknown":
      return t("common.unknown");
  }
};

// A group's mark: its environment's own badge, the badge's unassigned form,
// or plain text for databases that could not be read.
export function TargetGroupBadge({
  group,
  hasEnvTierFeature,
}: {
  group: TargetGroup;
  hasEnvTierFeature: boolean;
}) {
  const { t } = useTranslation();
  if (group.kind === "unknown") {
    return (
      <span className="shrink-0 text-xs text-control-light italic">
        {t("common.unknown")}
      </span>
    );
  }
  return (
    <EnvironmentBadge
      className="max-w-28"
      environment={
        group.kind === "environment" ? group.environment : nullEnvironment()
      }
      hasEnvTierFeature={hasEnvTierFeature}
    />
  );
}

type ListItem =
  | { kind: "header"; key: string; group: TargetGroup; count: number }
  | { kind: "target"; key: string; group?: TargetGroup; target: string };

// The popover body for a finding's databases: every one, grouped by
// environment under headers that stay in view, and searchable by database or
// instance name once the list is long. Without groups, while the databases
// are still loading, it is one list in the order given.
export function ReviewTargetList({
  groups,
  hasEnvTierFeature,
  searchRef,
  targets,
}: {
  groups: TargetGroup[] | undefined;
  hasEnvTierFeature: boolean;
  searchRef?: RefObject<HTMLInputElement | null>;
  targets: string[];
}) {
  const { t } = useTranslation();
  const databasesByName = useAppStore((state) => state.databasesByName);
  const [query, setQuery] = useState("");
  const scrollRef = useRef<HTMLDivElement>(null);
  const showSearch = targets.length > TARGET_SEARCH_THRESHOLD;
  const normalized = query.trim().toLowerCase();

  const items = useMemo(() => {
    const matches = (target: string) => {
      if (!normalized) return true;
      const { databaseName, instanceName } =
        extractDatabaseResourceName(target);
      const instanceTitle =
        databasesByName[target]?.instanceResource?.title || instanceName;
      return (
        databaseName.toLowerCase().includes(normalized) ||
        instanceTitle.toLowerCase().includes(normalized)
      );
    };
    if (!groups) {
      return targets
        .filter(matches)
        .map<ListItem>((target) => ({ kind: "target", key: target, target }));
    }
    return groups.flatMap<ListItem>((group) => {
      const groupKey = targetGroupKey(group);
      const matched = group.targets.filter(matches);
      if (matched.length === 0) return [];
      return [
        {
          kind: "header",
          key: `group:${groupKey}`,
          group,
          count: matched.length,
        },
        ...matched.map<ListItem>((target) => ({
          kind: "target",
          key: `${groupKey}:${target}`,
          group,
          target,
        })),
      ];
    });
  }, [databasesByName, groups, normalized, targets]);

  const headerIndexes = useMemo(
    () =>
      items.flatMap((item, index) => (item.kind === "header" ? [index] : [])),
    [items]
  );
  // The header of the group at the top of the view stays pinned there.
  const pinnedHeaderRef = useRef<number | undefined>(undefined);
  const rangeExtractor = useCallback(
    (range: Range) => {
      pinnedHeaderRef.current = headerIndexes.findLast(
        (index) => index <= range.startIndex
      );
      const indexes = new Set<number>();
      if (pinnedHeaderRef.current !== undefined) {
        indexes.add(pinnedHeaderRef.current);
      }
      const first = Math.max(0, range.startIndex - range.overscan);
      const last = Math.min(range.count - 1, range.endIndex + range.overscan);
      for (let index = first; index <= last; index++) indexes.add(index);
      return [...indexes].sort((a, b) => a - b);
    },
    [headerIndexes]
  );

  const virtualizer = useVirtualizer({
    count: items.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => ROW_HEIGHT,
    getItemKey: (index) => items[index].key,
    rangeExtractor,
    overscan: 8,
  });

  return (
    <div className="flex flex-col gap-y-2">
      {showSearch && (
        <SearchInput
          ref={searchRef}
          size="sm"
          value={query}
          placeholder={t("plan.review.thread.targets.search-placeholder")}
          aria-label={t("plan.review.thread.targets.search-placeholder")}
          onChange={(event) => {
            setQuery(event.target.value);
            // Show a new search's results from the first match.
            if (scrollRef.current) scrollRef.current.scrollTop = 0;
          }}
        />
      )}
      {items.length === 0 ? (
        <div className="flex flex-col items-center gap-y-2 py-4 text-center text-sm text-control-light">
          <span>
            {t("plan.review.thread.targets.no-match", { query: query.trim() })}
          </span>
          <Button
            size="sm"
            appearance="outline"
            onClick={() => {
              setQuery("");
              searchRef?.current?.focus();
            }}
          >
            {t("plan.review.thread.targets.clear-search")}
          </Button>
        </div>
      ) : (
        // Rows hold nothing focusable, so the list itself takes focus for
        // keyboard scrolling.
        <div
          ref={scrollRef}
          aria-label={t("plan.review.thread.targets.list-label")}
          className="max-h-80 overflow-y-auto rounded-xs focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-accent"
          tabIndex={0}
        >
          <div
            role="list"
            className="relative w-full"
            style={{ height: `${virtualizer.getTotalSize()}px` }}
          >
            {virtualizer.getVirtualItems().map((virtualItem) => {
              const item = items[virtualItem.index];
              const pinned =
                item.kind === "header" &&
                pinnedHeaderRef.current === virtualItem.index;
              return (
                <div
                  key={virtualItem.key}
                  data-index={virtualItem.index}
                  // Screen readers hear each row's environment from the row,
                  // so the visual header is hidden from them.
                  aria-hidden={item.kind === "header" ? true : undefined}
                  role={item.kind === "target" ? "listitem" : undefined}
                  className={cn(
                    "top-0 left-0 flex w-full items-center",
                    item.kind === "header" && "z-1 gap-x-1.5 bg-background",
                    pinned ? "sticky" : "absolute"
                  )}
                  style={{
                    height: `${virtualItem.size}px`,
                    transform: pinned
                      ? undefined
                      : `translateY(${virtualItem.start}px)`,
                  }}
                >
                  {item.kind === "header" ? (
                    <>
                      <TargetGroupBadge
                        group={item.group}
                        hasEnvTierFeature={hasEnvTierFeature}
                      />
                      <span className="text-xs text-control-light tabular-nums">
                        {item.count}
                      </span>
                    </>
                  ) : (
                    <>
                      <PlanTargetDisplay
                        className="text-xs"
                        keyword={normalized}
                        target={item.target}
                      />
                      {item.group && (
                        <span className="sr-only">
                          {targetGroupLabel(t, item.group)}
                        </span>
                      )}
                    </>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
