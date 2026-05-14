import { create } from "@bufbuild/protobuf";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { EllipsisText } from "@/react/components/ui/ellipsis-text";
import { cn } from "@/react/lib/utils";
import { useSQLEditorTabStore, useWorkSheetStore } from "@/store";
import { WorksheetSchema } from "@/types/proto-es/v1/worksheet_service_pb";
import type { SQLEditorTab } from "@/types/sqlEditor/tab";
import { tabListEvents } from "@/views/sql-editor/TabList/events";

type Props = {
  readonly tab: SQLEditorTab;
};

/**
 * Replaces frontend/src/views/sql-editor/TabList/TabItem/Label.vue.
 * Tab title with:
 *  - Double-click to enter in-place rename.
 *  - Listens for external `rename-tab` events (fired from the context menu)
 *    so right-click → Rename still works during the Vue → React migration.
 *  - Ellipsis + native tooltip via EllipsisText.
 */
export function Label({ tab }: Props) {
  const { t } = useTranslation();
  const tabStore = useSQLEditorTabStore();
  const worksheetStore = useWorkSheetStore();

  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(tab.title);
  const inputRef = useRef<HTMLInputElement>(null);

  const readonly = tab.viewState.view !== "CODE";
  const displayTitle = tab.title || t("common.untitled");
  // Captures whether the tab was already current at mousedown time, before
  // the parent TabItem's onMouseDown handler runs and switches activation.
  // We can't use a render-captured value: React commits the re-render
  // between mousedown and click, so by the time onClick fires the new render
  // already shows this tab as current. Inner mousedown handlers fire before
  // the parent's (bubble order), giving us a moment to read the *previous*
  // value from the store.
  const wasCurrentAtMouseDownRef = useRef(false);

  const beginEdit = () => {
    if (readonly) return;
    setDraft(tab.title);
    setEditing(true);
  };

  const cancelEdit = () => {
    setEditing(false);
    setDraft(tab.title);
  };

  const confirmEdit = () => {
    const title = draft.trim();
    tabStore.updateTab(tab.id, { title });
    if (tab.worksheet) {
      void worksheetStore.patchWorksheet(
        create(WorksheetSchema, {
          name: tab.worksheet,
          title,
        }),
        ["title"]
      );
    }
    setEditing(false);
  };

  // Select all text on mount (fires only when a new edit session starts via
  // the `editing` transition).
  useEffect(() => {
    if (editing) {
      // Defer so the input is actually mounted before we focus + select.
      requestAnimationFrame(() => {
        inputRef.current?.focus();
        inputRef.current?.select();
      });
    }
  }, [editing]);

  // Keep draft in sync when the tab title updates from elsewhere (auto-save,
  // worksheet fetch) while not editing.
  useEffect(() => {
    if (!editing) {
      setDraft(tab.title);
    }
  }, [tab.title, editing]);

  // Respond to external rename-tab events (fired from the context menu).
  // `readonly` + `tab.id` are the only closure values we care about; the
  // other helpers are referentially stable via the Pinia store singletons.
  useEffect(() => {
    const unsubscribe = tabListEvents.on("rename-tab", (payload) => {
      if (payload.tab.id !== tab.id) return;
      tabStore.setCurrentTabId(tab.id);
      if (readonly) return;
      setDraft(tab.title);
      setEditing(true);
    });
    return () => {
      unsubscribe();
    };
  }, [tab.id, tab.title, readonly, tabStore]);

  return (
    <div
      className={cn(
        "label relative min-w-24 max-w-64 overflow-hidden",
        tab.status === "CLEAN" && "clean",
        tab.status === "DIRTY" && "dirty",
        tab.status === "SAVING" && "saving"
      )}
    >
      {/* Keep the title mounted (just `invisible`) while editing so the
          parent retains its line-height; otherwise the empty text collapses
          the relative container and the `absolute inset-0` input shrinks to
          zero height → the cursor is visible but the text field is not. */}
      <EllipsisText
        text={displayTitle}
        className={cn(
          "text-sm leading-5 block",
          !tab.title && "text-control-placeholder italic",
          editing && "invisible"
        )}
      />
      {/* Invisible click layer — EllipsisText strips its own handlers.
          Click-to-rename behaviour:
          - Click on the *current* tab's label → enter rename mode.
          - Click on a non-current tab → just activates the tab (handled by
            the parent TabItem's onMouseDown). No rename on the first click.
          We snapshot `wasCurrentAtMouseDownRef` at this layer's mousedown
          (which fires before the parent's via bubble order). The parent's
          mousedown then mutates the store; by the time onClick fires, the
          ref still holds the pre-activation value. */}
      {!editing && !readonly && (
        <div
          className="absolute inset-0 cursor-text"
          onMouseDown={() => {
            wasCurrentAtMouseDownRef.current = tabStore.currentTabId === tab.id;
          }}
          onClick={() => {
            if (wasCurrentAtMouseDownRef.current) beginEdit();
          }}
        />
      )}
      {editing && (
        <input
          ref={inputRef}
          type="text"
          className="absolute inset-0 border-0 border-b p-0 text-sm leading-5 bg-background"
          value={draft}
          placeholder={t("common.untitled")}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={confirmEdit}
          onKeyDown={(e) => {
            if (e.nativeEvent.isComposing) return;
            if (e.key === "Enter") {
              (e.target as HTMLInputElement).blur();
            } else if (e.key === "Escape") {
              cancelEdit();
            }
          }}
        />
      )}
    </div>
  );
}
