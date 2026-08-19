/**
 * SavedQueryPane — React port of SavedQueryPane.vue (Stage 12, Phase 4).
 *
 * Hosts the SQL editor's saved query sidebar: search, filter menu, multi-select
 * toolbar, and one SheetTree per visible view.
 */

import { FunnelIcon, TrashIcon, XIcon } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  type SelectionAction,
  SelectionActionBar,
} from "@/components/SelectionActionBar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { SearchInput } from "@/components/ui/search-input";
import { cn } from "@/lib/utils";
import type {
  SavedQueryFolderNode,
  SheetViewMode,
} from "@/modules/sql-editor/model/Sheet";
import {
  useSheetContext,
  useSheetContextByView,
} from "@/modules/sql-editor/model/Sheet";
import { FilterMenuItem } from "./FilterMenuItem";
import { SheetTree, type SheetTreeHandle } from "./SheetTree";

function collectSelectableNodes(
  node: SavedQueryFolderNode | undefined
): SavedQueryFolderNode[] {
  if (!node || node.loadMore) return [];
  return [
    node,
    ...node.children.flatMap((child) => collectSelectableNodes(child)),
  ];
}

export function SavedQueryPane() {
  const { t } = useTranslation();

  const sheetContext = useSheetContext();
  const { filter, filterChanged, setFilter } = sheetContext;

  const myViewContext = useSheetContextByView("my");

  const mineSheetTreeRef = useRef<SheetTreeHandle>(null);

  const [multiSelectMode, setMultiSelectMode] = useState(false);
  const [checkedNodes, setCheckedNodes] = useState<SavedQueryFolderNode[]>([]);
  const [loading, setLoading] = useState(false);
  const [showFilterMenu, setShowFilterMenu] = useState(false);

  // Reset selection whenever multi-select mode exits or "my" view is hidden.
  useEffect(() => {
    if (!multiSelectMode || !filter.showMine) {
      setCheckedNodes([]);
    }
  }, [multiSelectMode, filter.showMine]);

  const views = useMemo<SheetViewMode[]>(() => {
    const results: SheetViewMode[] = [];
    if (filter.showShared) results.push("shared");
    if (filter.showDraft) results.push("draft");
    return results;
  }, [filter.showShared, filter.showDraft]);

  const selectableNodes = useMemo(
    () => collectSelectableNodes(myViewContext.sheetTree),
    [myViewContext.sheetTree]
  );
  const checkedKeySet = useMemo(
    () => new Set(checkedNodes.map((node) => node.key)),
    [checkedNodes]
  );
  const allSelected =
    selectableNodes.length > 0 &&
    selectableNodes.every((node) => checkedKeySet.has(node.key));

  const updateFilter = (patch: Partial<typeof filter>) => {
    setFilter((prev) => ({ ...prev, ...patch }));
  };

  const handleKeywordChange = (keyword: string) => {
    updateFilter({ keyword });
  };

  const handleMultiDelete = async () => {
    setLoading(true);
    try {
      await mineSheetTreeRef.current?.handleMultiDelete(checkedNodes);
    } finally {
      setLoading(false);
    }
  };

  const batchActions: SelectionAction[] = [
    {
      key: "cancel",
      label: t("common.cancel"),
      icon: XIcon,
      onClick: () => setMultiSelectMode(false),
      disabled: loading,
    },
    {
      key: "delete",
      label: t("common.delete"),
      icon: TrashIcon,
      onClick: handleMultiDelete,
      disabled: checkedNodes.length === 0 || loading,
      tone: "destructive",
    },
  ];

  const showMultiSelectToolbar = multiSelectMode && filter.showMine;
  const hasAnyView = filter.showMine || views.length > 0;

  return (
    <div className="relative flex h-full min-w-0 max-w-full flex-col gap-1 overflow-hidden py-1 text-sm">
      <div className="flex min-w-0 items-center gap-x-1 px-1">
        <SearchInput
          size="sm"
          value={filter.keyword}
          onChange={(e) => handleKeywordChange(e.target.value)}
          placeholder={t("sheet.search-sheets")}
          wrapperClassName="max-w-full"
        />
        <DropdownMenu open={showFilterMenu} onOpenChange={setShowFilterMenu}>
          <DropdownMenuTrigger
            aria-label={t("sheet.search-sheets")}
            className="inline-flex items-center justify-center size-7 rounded-xs text-control hover:bg-control-bg cursor-pointer outline-hidden focus-visible:ring-2 focus-visible:ring-accent"
          >
            <FunnelIcon
              className={cn("size-4", filterChanged && "text-accent")}
            />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" sideOffset={4}>
            <FilterMenuItem
              label={t("sheet.filter.show-mine")}
              value={filter.showMine}
              onValueChange={(val) => updateFilter({ showMine: val })}
            />
            <FilterMenuItem
              label={t("sheet.filter.show-shared")}
              value={filter.showShared}
              onValueChange={(val) => updateFilter({ showShared: val })}
            />
            <FilterMenuItem
              label={t("sheet.filter.show-draft")}
              value={filter.showDraft}
              onValueChange={(val) => updateFilter({ showDraft: val })}
            />
            <FilterMenuItem
              label={t("sheet.filter.only-show-starred")}
              value={filter.onlyShowStarred}
              onValueChange={(val) => updateFilter({ onlyShowStarred: val })}
            />
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <div
        className={cn(
          "relative flex min-w-0 max-w-full flex-1 flex-col gap-y-2 overflow-y-auto overflow-x-hidden",
          showMultiSelectToolbar && "pb-16"
        )}
      >
        {filter.showMine && (
          <SheetTree
            key="my"
            ref={mineSheetTreeRef}
            view="my"
            multiSelectMode={multiSelectMode}
            checkedNodes={checkedNodes}
            onMultiSelectModeChange={setMultiSelectMode}
            onCheckedNodesChange={setCheckedNodes}
          />
        )}
        {views.map((view) => (
          // Non-"my" trees intentionally omit multi-select callbacks. Vue
          // bound v-model only on the `my` tree; wiring them everywhere let
          // a shared/draft right-click populate the my tree's checkedNodes,
          // which the toolbar's Delete flow acts on.
          <SheetTree key={view} view={view} />
        ))}
        {!hasAnyView && (
          <div className="mt-10 text-center text-sm text-control-light">
            {t("common.no-data")}
          </div>
        )}
      </div>

      {showMultiSelectToolbar && (
        <SelectionActionBar
          count={checkedNodes.length}
          label={t("common.n-selected", { n: checkedNodes.length })}
          allSelected={allSelected}
          onToggleSelectAll={() => {
            setCheckedNodes(allSelected ? [] : selectableNodes);
          }}
          actions={batchActions}
          maxVisibleActions={2}
          forceVisible
          placement="container"
          hideLabel
          density="compact"
        />
      )}
    </div>
  );
}
