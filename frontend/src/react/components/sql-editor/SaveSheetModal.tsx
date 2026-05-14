import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { Input } from "@/react/components/ui/input";
import { useSQLEditorEvent } from "@/react/hooks/useSQLEditorEvent";
import { useSQLEditorWorksheetStore, useWorkSheetStore } from "@/store";
import type { SQLEditorTab } from "@/types";
import { UNKNOWN_ID } from "@/types";
import { extractWorksheetID } from "@/utils";
import { useSheetContextByView } from "@/views/sql-editor/Sheet";
import { FolderForm } from "./FolderForm";

export function SaveSheetModal() {
  const { t } = useTranslation();
  const worksheetStore = useWorkSheetStore();
  const editorWorksheetStore = useSQLEditorWorksheetStore();
  const sheetContext = useSheetContextByView("my");

  const [open, setOpen] = useState(false);
  const [title, setTitle] = useState("");
  const [folder, setFolder] = useState("");
  const [rawTab, setRawTab] = useState<SQLEditorTab | undefined>(undefined);

  const needShowModal = (tab: SQLEditorTab) => !tab.worksheet;

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

    editorWorksheetStore.abortAutoSave();

    const { worksheet, connection, statement, id: tabId } = effectiveTab;
    const folders = sheetContext.getFoldersForWorksheet(effectiveFolder);

    const sheetId = extractWorksheetID(worksheet ?? "");
    if (sheetId !== String(UNKNOWN_ID)) {
      await editorWorksheetStore.maybeUpdateWorksheet({
        tabId,
        worksheet,
        title: effectiveTitle,
        database: connection.database,
        statement,
        folders,
      });
    } else {
      await editorWorksheetStore.createWorksheet({
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

    // Compute the folder synchronously: for an already-saved worksheet, use
    // its current pwd; otherwise reset. We then both reflect this in state
    // (for the modal-open path) AND pass it explicitly into doSaveSheet
    // (for the silent path) — bypassing React's async setState batching.
    let nextFolder = "";
    if (tab.worksheet) {
      const worksheet = worksheetStore.getWorksheetByName(tab.worksheet);
      if (worksheet) {
        nextFolder = sheetContext.getPwdForWorksheet(worksheet);
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
            />
          </div>
          <FolderForm folder={folder} onFolderChange={setFolder} />
          <div className="flex justify-end gap-x-2 mt-4">
            <Button variant="outline" onClick={close}>
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
