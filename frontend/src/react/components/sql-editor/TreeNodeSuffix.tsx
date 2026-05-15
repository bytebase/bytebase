import { MoreHorizontal, Star, Users, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { useSQLEditorTabStore } from "@/react/stores/sqlEditor/tab-vue-state";
import { useUserStore, useWorkSheetStore } from "@/store";
import { Worksheet_Visibility } from "@/types/proto-es/v1/worksheet_service_pb";
import type {
  SheetViewMode,
  WorksheetFolderNode,
} from "@/views/sql-editor/Sheet";
import { useSheetContext } from "@/views/sql-editor/Sheet";

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

  const worksheetStore = useWorkSheetStore();
  const tabStore = useSQLEditorTabStore();
  const userStore = useUserStore();
  const { isWorksheetCreator } = useSheetContext();

  const worksheetLite = useVueState(() => {
    if (!node.worksheet) {
      return undefined;
    }
    const sheet = worksheetStore.getWorksheetByName(node.worksheet.name);
    if (!sheet) {
      return undefined;
    }
    return {
      name: sheet.name,
      starred: sheet.starred,
      visibility: sheet.visibility,
      creator: sheet.creator,
    };
  });

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
    return userStore.getUserByIdentifier(creator)?.title ?? creator;
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
          const tab = tabStore.getTabById(node.worksheet.name);
          if (tab && tab.status !== "CLEAN") {
            if (
              !window.confirm(
                t("sql-editor.hint-tips.confirm-to-close-unsaved-sheet.content")
              )
            ) {
              return;
            }
          }
          tabStore.closeTab(node.worksheet.name);
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
