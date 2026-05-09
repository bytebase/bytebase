import { ChevronLeft, Play, Save, Share2 } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/react/components/ui/popover";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorStore,
  useSQLEditorTabStore,
  useUIStateStore,
  useWorkSheetAndTabStore,
  useWorkSheetStore,
} from "@/store";
import type { SQLEditorQueryParams } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { isWorksheetWritableV1, keyboardShortcutStr } from "@/utils";
import { sqlEditorEvents } from "@/views/sql-editor/events";
import { AdminModeButton } from "./AdminModeButton";
import { ChooserGroup } from "./ChooserGroup";
import { OpenAIButton } from "./OpenAIButton";
import { QueryContextSettingPopover } from "./QueryContextSettingPopover";
import { SharePopoverBody } from "./SharePopoverBody";

type Props = {
  readonly onExecute?: (params: SQLEditorQueryParams) => void;
};

/**
 * Replaces frontend/src/views/sql-editor/EditorCommon/EditorAction.vue.
 * Top toolbar in the SQL editor: Run / QueryContextSettingPopover /
 * AdminModeButton / Save / Share / ChooserGroup / OpenAIButton.
 *
 * `onExecute` is optional because `TerminalPanel.vue` mounts the toolbar in
 * ADMIN mode where the Run button is not rendered.
 */
export function EditorAction({ onExecute }: Props) {
  const { t } = useTranslation();
  const tabStore = useSQLEditorTabStore();
  const editorStore = useSQLEditorStore();
  const uiStateStore = useUIStateStore();
  const worksheetStore = useWorkSheetStore();
  const sheetAndTabStore = useWorkSheetAndTabStore();
  const { instance: instanceRef } = useConnectionOfCurrentSQLEditorTab();

  const [shareOpen, setShareOpen] = useState(false);

  // Track each tab field as a separate primitive reactive subscription
  // — Pinia mutates the tab proxy in place via `Object.assign`, so a
  // getter that just returns `tabStore.currentTab` would only fire on
  // tab switches (the proxy reference changing), not on `statement` /
  // `status` / `connection.*` field flips that happen within the same
  // tab. Splitting into one `useVueState` per field gives Vue a clean
  // primitive value to compare on each change, so each individual
  // mutation fires its own re-render (Run button enables as soon as
  // the user types, Save button enables on `status` → DIRTY, etc.).
  const currentTab = useVueState(() => tabStore.currentTab);
  const tabStatement = useVueState(() => tabStore.currentTab?.statement ?? "");
  const tabStatus = useVueState(() => tabStore.currentTab?.status);
  const tabMode = useVueState(() => tabStore.currentTab?.mode);
  const tabWorksheet = useVueState(() => tabStore.currentTab?.worksheet ?? "");
  const tabConnectionTable = useVueState(
    () => tabStore.currentTab?.connection.table ?? ""
  );
  const isDisconnected = useVueState(() => tabStore.isDisconnected);
  const instance = useVueState(() => instanceRef.value);
  const resultRowsLimit = useVueState(() => editorStore.resultRowsLimit);
  const currentWorksheet = useVueState(() => sheetAndTabStore.currentSheet);

  const isAdminMode = tabMode === "ADMIN";
  const showSheetsFeature = tabMode === "WORKSHEET";
  const isEmptyStatement = !currentTab || tabStatement === "";

  const queryTip =
    instance.engine === Engine.COSMOSDB && !tabConnectionTable
      ? t("database.table.select-tip")
      : "";

  const allowQuery = (() => {
    if (isDisconnected) return false;
    if (isEmptyStatement) return false;
    if (instance.engine === Engine.COSMOSDB) {
      return !!tabConnectionTable;
    }
    return true;
  })();

  const canWriteSheet = (() => {
    if (!tabWorksheet) return false;
    const sheet = worksheetStore.getWorksheetByName(tabWorksheet);
    return sheet ? isWorksheetWritableV1(sheet) : false;
  })();

  const allowSave = (() => {
    if (!showSheetsFeature || !currentTab) return false;
    if (tabWorksheet) {
      if (!canWriteSheet) return false;
      const sheet = worksheetStore.getWorksheetByName(tabWorksheet);
      if (sheet && sheet.database !== currentTab.connection.database) {
        return true;
      }
    }
    // Only disable when status is CLEAN (nothing to save).
    // SAVING is allowed — manual save will abort auto-save and proceed.
    return tabStatus !== "CLEAN";
  })();

  const allowShare = (() => {
    if (!currentTab) return false;
    if (tabStatus !== "CLEAN") return false;
    if (isEmptyStatement || isDisconnected) return false;
    if (tabWorksheet && !canWriteSheet) return false;
    return true;
  })();

  const showQueryContextSettingPopover =
    !!currentTab && !!instance && !isAdminMode;

  const handleRunQuery = () => {
    if (!currentTab || !onExecute) return;
    const statement = currentTab.selectedStatement || currentTab.statement;
    onExecute({
      statement,
      connection: { ...currentTab.connection },
      engine: instance.engine,
      explain: false,
      selection: currentTab.editorState.selection,
    });
    uiStateStore.saveIntroStateByKey({
      key: "data.query",
      newState: true,
    });
  };

  const exitAdminMode = () => {
    // Inlined to avoid pulling `@/types` (monaco-editor transitive) into the
    // React bundle. Matches `DEFAULT_SQL_EDITOR_TAB_MODE` in `@/types/sqlEditor/tab`.
    tabStore.updateCurrentTab({ mode: "WORKSHEET" });
  };

  const handleClickSave = () => {
    if (!currentTab) return;
    void sqlEditorEvents.emit("save-sheet", { tab: currentTab });
  };

  return (
    <div className="w-full flex flex-wrap gap-y-2 justify-between sm:items-center p-2 border-b bg-background">
      <div className="action-left gap-x-2 flex overflow-x-auto sm:overflow-x-hidden items-center">
        {isAdminMode && (
          <Button
            variant="outline"
            className="h-8 px-1.5 gap-1 border-dashed text-sm"
            onClick={(e) => {
              e.stopPropagation();
              exitAdminMode();
            }}
          >
            <ChevronLeft className="size-4" />
            <span>{t("sql-editor.admin-mode.exit")}</span>
          </Button>
        )}

        {!isAdminMode && (
          <div className="inline-flex">
            <Tooltip content={queryTip} side="bottom">
              <Button
                variant="default"
                size="sm"
                className={cn("h-7 px-1.5 gap-1 rounded-r-none text-sm")}
                disabled={!allowQuery}
                onClick={handleRunQuery}
              >
                <Play className="size-4 fill-current" />
                <span className="inline-flex items-center">
                  (limit&nbsp;{resultRowsLimit})
                </span>
              </Button>
            </Tooltip>
            <QueryContextSettingPopover
              disabled={!showQueryContextSettingPopover || !allowQuery}
            />
          </div>
        )}

        <AdminModeButton size="sm" hideText />

        {showSheetsFeature && (
          <>
            <Tooltip
              content={
                <span className="inline-flex items-center gap-1">
                  <span>{t("common.save")}</span>
                  <span>({keyboardShortcutStr("cmd_or_ctrl+S")})</span>
                </span>
              }
              side="bottom"
            >
              <Button
                variant="outline"
                size="sm"
                className="h-7 px-1.5"
                disabled={!allowSave}
                onClick={handleClickSave}
                aria-label={t("common.save")}
              >
                <Save className="size-4" />
              </Button>
            </Tooltip>

            <Popover
              open={shareOpen}
              onOpenChange={(next) => {
                if (!allowShare && next) return;
                setShareOpen(next);
              }}
            >
              <Tooltip content={t("common.share")} side="bottom">
                <PopoverTrigger
                  render={
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-7 px-1.5"
                      disabled={!allowShare}
                      aria-label={t("common.share")}
                    >
                      <Share2 className="size-4" />
                    </Button>
                  }
                />
              </Tooltip>
              <PopoverContent align="end" sideOffset={4}>
                <SharePopoverBody
                  worksheet={currentWorksheet}
                  onUpdated={() => setShareOpen(false)}
                />
              </PopoverContent>
            </Popover>
          </>
        )}
      </div>
      <div className="action-right gap-x-2 flex overflow-x-auto sm:overflow-x-hidden sm:justify-end items-center">
        <ChooserGroup />
        <OpenAIButton
          size="sm"
          statement={currentTab?.selectedStatement || currentTab?.statement}
        />
      </div>
    </div>
  );
}
