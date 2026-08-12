import { MoreHorizontal, Star, Users, X } from "lucide-react";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useShallow } from "zustand/react/shallow";
import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import {
  type SavedQueryFolderNode,
  type SheetViewMode,
  useSheetContext,
} from "@/modules/sql-editor/model/Sheet";
import { getSQLEditorTabsState } from "@/modules/sql-editor/store/tab";
import { useAppStore } from "@/stores/app";
import { SavedQuery_Visibility } from "@/types/proto-es/v1/saved_query_service_pb";

type Props = {
  readonly node: SavedQueryFolderNode;
  readonly view: SheetViewMode;
  readonly onSharePanelShow: (
    e: React.MouseEvent,
    node: SavedQueryFolderNode
  ) => void;
  readonly onContextMenuShow: (
    e: React.MouseEvent,
    node: SavedQueryFolderNode
  ) => void;
  readonly onToggleStar: (args: {
    savedQuery: string;
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

  const { isSavedQueryCreator } = useSheetContext();

  // `useShallow` is required: this selector builds a fresh object each
  // call, which would fail `useSyncExternalStore`'s `Object.is` snapshot
  // check and spin an infinite render loop. Shallow-comparing the fields
  // keeps the snapshot stable when the saved query's values are unchanged.
  const savedQueryLite = useAppStore(
    useShallow((state) => {
      if (!node.savedQuery) {
        return undefined;
      }
      const sheet = state.getSavedQueryByName(node.savedQuery.name);
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
  const savedQueryCreatorTitle = useAppStore((state) =>
    savedQueryLite?.creator
      ? state.getUserByIdentifier(savedQueryLite.creator)?.title
      : undefined
  );
  const getOrFetchUserByIdentifier = useAppStore(
    (state) => state.getOrFetchUserByIdentifier
  );

  useEffect(() => {
    if (!savedQueryLite?.creator || savedQueryCreatorTitle) {
      return;
    }
    void getOrFetchUserByIdentifier({ identifier: savedQueryLite.creator });
  }, [
    getOrFetchUserByIdentifier,
    savedQueryCreatorTitle,
    savedQueryLite?.creator,
  ]);

  const visibilityDisplayName = (visibility: SavedQuery_Visibility) => {
    switch (visibility) {
      case SavedQuery_Visibility.PRIVATE:
        return t("sql-editor.private");
      case SavedQuery_Visibility.PROJECT_READ:
        return t("sql-editor.project-read");
      case SavedQuery_Visibility.PROJECT_WRITE:
        return t("sql-editor.project-write");
      default:
        return "";
    }
  };

  const creatorForSheet = (creator: string) => {
    return savedQueryCreatorTitle ?? creator;
  };

  // Draft view: show X button to close the draft tab
  if (view === "draft") {
    if (!node.savedQuery) {
      return null;
    }
    return (
      <X
        className="size-4 text-control shrink-0"
        onClick={(e) => {
          e.stopPropagation();
          if (!node.savedQuery?.name) return;
          // Draft nodes use tab.id as savedQuery.name (drafts have no saved query field).
          const tabsState = getSQLEditorTabsState();
          const tab = tabsState.tabsById.get(node.savedQuery.name);
          if (tab && tab.status !== "CLEAN") {
            if (
              !window.confirm(
                t("sql-editor.hint-tips.confirm-to-close-unsaved-sheet.content")
              )
            ) {
              return;
            }
          }
          tabsState.closeTab(node.savedQuery.name);
        }}
      />
    );
  }

  // Folder node: only show "More" button
  if (!node.savedQuery) {
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

  // SavedQuery node: visibility badge + star + more
  if (!savedQueryLite) {
    return null;
  }

  const showVisibilityBadge =
    savedQueryLite.visibility === SavedQuery_Visibility.PROJECT_READ ||
    savedQueryLite.visibility === SavedQuery_Visibility.PROJECT_WRITE;

  return (
    <div className="inline-flex shrink-0 items-center gap-x-1">
      {showVisibilityBadge && (
        <Tooltip
          content={
            <div>
              <div>
                {t("common.visibility")}
                {": "}
                {visibilityDisplayName(savedQueryLite.visibility)}
              </div>
              {!isSavedQueryCreator(savedQueryLite) && (
                <div>
                  {t("common.creator")}
                  {": "}
                  {creatorForSheet(savedQueryLite.creator)}
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
          savedQueryLite.starred ? "text-yellow-400" : "text-control-light"
        )}
        onClick={(e) => {
          e.stopPropagation();
          onToggleStar({
            savedQuery: savedQueryLite.name,
            starred: !savedQueryLite.starred,
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
