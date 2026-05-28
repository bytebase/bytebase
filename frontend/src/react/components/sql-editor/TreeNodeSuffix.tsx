import { MoreHorizontal, Star, Users, X } from "lucide-react";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useShallow } from "zustand/react/shallow";
import { Tooltip } from "@/react/components/ui/tooltip";
import { cn } from "@/react/lib/utils";
import { useAppStore } from "@/react/stores/app";
import { getSQLEditorTabsState } from "@/react/stores/sqlEditor/tab";
import { Worksheet_Visibility } from "@/types/proto-es/v1/worksheet_service_pb";
import {
  type SheetViewMode,
  useSheetContext,
  type WorksheetFolderNode,
} from "@/views/sql-editor/Sheet";

type Props = {
  readonly node: WorksheetFolderNode;
  readonly view: SheetViewMode;
  readonly onSharePanelShow: (
    e: React.MouseEvent,
    node: WorksheetFolderNode
  ) => void;
  readonly onContextMenuShow: (
    e: React.MouseEvent,
    node: WorksheetFolderNode
  ) => void;
  readonly onToggleStar: (args: {
    worksheet: string;
    starred: boolean;
  }) => void;
};

export function TreeNodeSuffix({
  node,
  view,
  onSharePanelShow,
  onContextMenuShow,
  onToggleStar,
}: Props) {
  const { t } = useTranslation();

  const { isWorksheetCreator } = useSheetContext();

  // `useShallow` is required: this selector builds a fresh object each
  // call, which would fail `useSyncExternalStore`'s `Object.is` snapshot
  // check and spin an infinite render loop. Shallow-comparing the fields
  // keeps the snapshot stable when the worksheet's values are unchanged.
  const worksheetLite = useAppStore(
    useShallow((state) => {
      if (!node.worksheet) {
        return undefined;
      }
      const sheet = state.getWorksheetByName(node.worksheet.name);
      if (!sheet) {
        return undefined;
      }
      return {
        name: sheet.name,
        starred: sheet.starred,
        visibility: sheet.visibility,
        creator: sheet.creator,
      };
    })
  );
  const worksheetCreatorTitle = useAppStore((state) =>
    worksheetLite?.creator
      ? state.getUserByIdentifier(worksheetLite.creator)?.title
      : undefined
  );
  const getOrFetchUserByIdentifier = useAppStore(
    (state) => state.getOrFetchUserByIdentifier
  );

  useEffect(() => {
    if (!worksheetLite?.creator || worksheetCreatorTitle) {
      return;
    }
    void getOrFetchUserByIdentifier({ identifier: worksheetLite.creator });
  }, [
    getOrFetchUserByIdentifier,
    worksheetCreatorTitle,
    worksheetLite?.creator,
  ]);

  const visibilityDisplayName = (visibility: Worksheet_Visibility) => {
    switch (visibility) {
      case Worksheet_Visibility.PRIVATE:
        return t("sql-editor.private");
      case Worksheet_Visibility.PROJECT_READ:
        return t("sql-editor.project-read");
      case Worksheet_Visibility.PROJECT_WRITE:
        return t("sql-editor.project-write");
      default:
        return "";
    }
  };

  const creatorForSheet = (creator: string) => {
    return worksheetCreatorTitle ?? creator;
  };

  // Draft view: show X button to close the draft tab
  if (view === "draft") {
    if (!node.worksheet) {
      return null;
    }
    return (
      <X
        className="size-4 text-control shrink-0"
        onClick={(e) => {
          e.stopPropagation();
          if (!node.worksheet?.name) return;
          // Draft nodes use tab.id as worksheet.name (drafts have no worksheet field).
          const tabsState = getSQLEditorTabsState();
          const tab = tabsState.tabsById.get(node.worksheet.name);
          if (tab && tab.status !== "CLEAN") {
            if (
              !window.confirm(
                t("sql-editor.hint-tips.confirm-to-close-unsaved-sheet.content")
              )
            ) {
              return;
            }
          }
          tabsState.closeTab(node.worksheet.name);
        }}
      />
    );
  }

  // Folder node: only show "More" button
  if (!node.worksheet) {
    return (
      <MoreHorizontal
        className="size-4 text-control shrink-0 cursor-pointer"
        onClick={(e) => {
          e.preventDefault();
          e.stopPropagation();
          onContextMenuShow(e, node);
        }}
      />
    );
  }

  // Worksheet node: visibility badge + star + more
  if (!worksheetLite) {
    return null;
  }

  const showVisibilityBadge =
    worksheetLite.visibility === Worksheet_Visibility.PROJECT_READ ||
    worksheetLite.visibility === Worksheet_Visibility.PROJECT_WRITE;

  return (
    <div className="inline-flex gap-x-1 items-center">
      {showVisibilityBadge && (
        <Tooltip
          content={
            <div>
              <div>
                {t("common.visibility")}
                {": "}
                {visibilityDisplayName(worksheetLite.visibility)}
              </div>
              {!isWorksheetCreator(worksheetLite) && (
                <div>
                  {t("common.creator")}
                  {": "}
                  {creatorForSheet(worksheetLite.creator)}
                </div>
              )}
            </div>
          }
        >
          <Users
            className="size-4 text-control-light shrink-0"
            onClick={(e) => {
              e.stopPropagation();
              onSharePanelShow(e, node);
            }}
          />
        </Tooltip>
      )}
      <Star
        className={cn(
          "size-4 shrink-0",
          worksheetLite.starred ? "text-yellow-400" : "text-control-light"
        )}
        onClick={(e) => {
          e.stopPropagation();
          onToggleStar({
            worksheet: worksheetLite.name,
            starred: !worksheetLite.starred,
          });
        }}
      />
      <MoreHorizontal
        className="size-4 text-control shrink-0 cursor-pointer"
        onClick={(e) => {
          e.preventDefault();
          e.stopPropagation();
          onContextMenuShow(e, node);
        }}
      />
    </div>
  );
}
