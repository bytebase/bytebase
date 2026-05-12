/**
 * WorksheetPane — React port of WorksheetPane.vue (Stage 12, Phase 4).
 *
 * Hosts the SQL editor's worksheet sidebar: search, filter menu, multi-select
 * toolbar, move-to-folder dialog, and one SheetTree per visible view.
 */

import { FolderInputIcon, FunnelIcon, TrashIcon, XIcon } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
import { SearchInput } from "@/react/components/ui/search-input";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import type {
  SheetViewMode,
  WorksheetFolderNode,
} from "@/views/sql-editor/Sheet";
import {
  useSheetContext,
  useSheetContextByView,
} from "@/views/sql-editor/Sheet";
import { FilterMenuItem } from "./FilterMenuItem";
import { FolderForm } from "./FolderForm";
import { SheetTree, type SheetTreeHandle } from "./SheetTree";

export function WorksheetPane() {
  const { t } = useTranslation();

  const sheetContext = useSheetContext();
  const { filter: filterRef, batchUpdateWorksheetFolders } = sheetContext;
  const filterChanged = useVueState(() => sheetContext.filterChanged.value);
  const filter = useVueState(() => filterRef.value);

  const { getFoldersForWorksheet } = useSheetContextByView("my");

  const mineSheetTreeRef = useRef<SheetTreeHandle>(null);

  const [multiSelectMode, setMultiSelectMode] = useState(false);
  const [checkedNodes, setCheckedNodes] = useState<WorksheetFolderNode[]>([]);
  const [showReorgModal, setShowReorgModal] = useState(false);
  const [pendingMoveFolder, setPendingMoveFolder] = useState("");
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

  const checkedWorksheets = useMemo(
    () =>
      checkedNodes
        .filter((node) => node.worksheet)
        .map((node) => node.worksheet!.name),
    [checkedNodes]
  );

  const updateFilter = (patch: Partial<typeof filter>) => {
    filterRef.value = { ...filterRef.value, ...patch };
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

  const handleMoveWorksheets = async () => {
    setLoading(true);
    try {
      const folders = getFoldersForWorksheet(pendingMoveFolder);
      await batchUpdateWorksheetFolders(
        checkedWorksheets.map((worksheet) => ({ name: worksheet, folders }))
      );
      setShowReorgModal(false);
      setMultiSelectMode(false);
      setPendingMoveFolder("");
    } finally {
      setLoading(false);
    }
  };

  const closeReorgModal = () => {
    setShowReorgModal(false);
    setPendingMoveFolder("");
  };

  const showMultiSelectToolbar = multiSelectMode && filter.showMine;
  const hasAnyView = filter.showMine || views.length > 0;

  return (
    <div className="h-full flex flex-col gap-1 overflow-hidden py-1 text-sm">
      <div className="flex items-center gap-x-1 px-1">
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

      <div className="relative flex-1 flex flex-col gap-y-2 overflow-y-auto worksheet-scroll">
        {showMultiSelectToolbar && (
          <div className="sticky top-0 z-10 flex flex-wrap items-center justify-start gap-y-1 gap-x-1 bg-control-bg py-2 px-1">
            <Button
              variant="ghost"
              size="xs"
              className="text-error hover:text-error"
              disabled={checkedNodes.length === 0 || loading}
              onClick={handleMultiDelete}
            >
              <TrashIcon className="size-3.5" />
              {t("common.delete")}
            </Button>
            <Button
              variant="ghost"
              size="xs"
              disabled={checkedWorksheets.length === 0 || loading}
              onClick={() => {
                setPendingMoveFolder("");
                setShowReorgModal(true);
              }}
            >
              <FolderInputIcon className="size-3.5" />
              {t("sheet.move-worksheets")}
            </Button>
            <Button
              variant="ghost"
              size="xs"
              disabled={loading}
              onClick={() => setMultiSelectMode(false)}
            >
              <XIcon className="size-3.5" />
              {t("common.cancel")}
            </Button>
          </div>
        )}

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
          // which the toolbar's Delete + Move-to-folder flows act on.
          <SheetTree key={view} view={view} />
        ))}
        {!hasAnyView && (
          <div className="mt-10 text-center text-sm text-control-light">
            {t("common.no-data")}
          </div>
        )}
      </div>

      <Dialog
        open={showReorgModal}
        onOpenChange={(open) => !open && closeReorgModal()}
      >
        <DialogContent className="w-lg max-w-[calc(100vw-8rem)] p-6">
          <DialogTitle>{t("sheet.move-worksheets")}</DialogTitle>
          <div className="mt-3 flex flex-col gap-y-3">
            <FolderForm
              folder={pendingMoveFolder}
              onFolderChange={setPendingMoveFolder}
            />
            <div className="flex justify-end gap-x-2 mt-4">
              <Button variant="outline" onClick={closeReorgModal}>
                {t("common.close")}
              </Button>
              <Button onClick={handleMoveWorksheets} disabled={loading}>
                {t("common.save")}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
