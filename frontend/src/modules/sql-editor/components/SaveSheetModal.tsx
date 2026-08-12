import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { useSQLEditorEvent } from "@/hooks/useSQLEditorEvent";
import { useSheetContextByView } from "@/modules/sql-editor/model/Sheet";
import { useSQLEditorStore } from "@/modules/sql-editor/store";
import { useAppStore } from "@/stores/app";
import type { SQLEditorTab } from "@/types";
import { UNKNOWN_ID } from "@/types";
import { extractSavedQueryID } from "@/utils";
import { FolderForm } from "./FolderForm";

export function SaveSheetModal() {
  const { t } = useTranslation();
  const abortAutoSave = useSQLEditorStore((s) => s.abortAutoSave);
  const maybeUpdateSavedQuery = useSQLEditorStore(
    (s) => s.maybeUpdateSavedQuery
  );
  const createSavedQuery = useSQLEditorStore((s) => s.createSavedQuery);
  const sheetContext = useSheetContextByView("my");

  const [open, setOpen] = useState(false);
  const [title, setTitle] = useState("");
  const [folder, setFolder] = useState("");
  const [rawTab, setRawTab] = useState<SQLEditorTab | undefined>(undefined);

  // A manual save opens the modal when the saved query has never been saved
  // OR is still untitled, so the user can give it a title rather than
  // silently persisting an "Untitled" savedQuery. Auto-save never reaches
  // here (it calls `maybeUpdateSavedQuery` directly).
  const needShowModal = (tab: SQLEditorTab) =>
    !tab.savedQuery || !tab.title.trim();

  const doSaveSheet = async (
    tab?: SQLEditorTab,
    tabTitle?: string,
    tabFolder?: string
  ) => {
    const effectiveTab = tab ?? rawTab;
    if (!effectiveTab) {
      setOpen(false);
      return;
    }
    // Silent path passes title + folder explicitly because React state setters
    // are async — reading state here would observe the prior modal's values.
    // Empty titles are allowed; surfaces render an "Untitled" placeholder.
    const effectiveTitle = tabTitle ?? title;
    const effectiveFolder = tabFolder ?? folder;

    abortAutoSave();

    const { savedQuery, connection, statement, id: tabId } = effectiveTab;
    const folders = sheetContext.getFoldersForSavedQuery(effectiveFolder);

    const sheetId = extractSavedQueryID(savedQuery ?? "");
    if (sheetId !== String(UNKNOWN_ID)) {
      await maybeUpdateSavedQuery({
        tabId,
        savedQuery,
        title: effectiveTitle,
        database: connection.database,
        statement,
        folders,
      });
    } else {
      await createSavedQuery({
        tabId,
        title: effectiveTitle,
        statement,
        database: connection.database,
        folders,
      });
    }

    setOpen(false);
  };

  useSQLEditorEvent("save-sheet", ({ tab, editTitle }) => {
    setTitle(tab.title);
    setRawTab(tab);

    // Compute the folder synchronously: for an already-saved saved query, use
    // its current pwd; otherwise reset. We then both reflect this in state
    // (for the modal-open path) AND pass it explicitly into doSaveSheet
    // (for the silent path) — bypassing React's async setState batching.
    let nextFolder = "";
    if (tab.savedQuery) {
      const savedQuery = useAppStore
        .getState()
        .getSavedQueryByName(tab.savedQuery);
      if (savedQuery) {
        nextFolder = sheetContext.getPwdForSavedQuery(savedQuery);
      }
    }
    setFolder(nextFolder);

    if (needShowModal(tab) || editTitle) {
      setOpen(true);
    } else {
      void doSaveSheet(tab, tab.title, nextFolder);
    }
  });

  const close = () => setOpen(false);

  return (
    <Dialog open={open} onOpenChange={(next) => !next && close()}>
      <DialogContent className="w-[32rem] max-w-[calc(100vw-8rem)] 2xl:max-w-[calc(100vw-8rem)] p-4">
        <DialogTitle>{t("sql-editor.save-sheet")}</DialogTitle>
        <div className="flex flex-col gap-y-3">
          <div className="flex flex-col gap-y-1">
            <p>{t("common.title")}</p>
            <Input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={t("common.untitled")}
              maxLength={200}
              autoComplete="off"
            />
          </div>
          <FolderForm folder={folder} onFolderChange={setFolder} />
          <div className="flex justify-end gap-x-2 mt-4">
            <Button appearance="outline" onClick={close}>
              {t("common.close")}
            </Button>
            <Button onClick={() => void doSaveSheet()}>
              {t("common.save")}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
