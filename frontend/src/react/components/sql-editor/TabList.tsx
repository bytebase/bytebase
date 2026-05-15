import {
  closestCenter,
  DndContext,
  type DragEndEvent,
  PointerSensor,
  useSensor,
  useSensors,
} from "@dnd-kit/core";
import {
  horizontalListSortingStrategy,
  SortableContext,
} from "@dnd-kit/sortable";
import { Plus } from "lucide-react";
import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import scrollIntoView from "scroll-into-view-if-needed";
import { HeaderProfileMenuMount } from "@/react/components/HeaderProfileMenuMount";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/react/components/ui/alert-dialog";
import { Button } from "@/react/components/ui/button";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { useSQLEditorStore } from "@/react/stores/sqlEditor";
import { useSQLEditorTabStore } from "@/react/stores/sqlEditor/tab-vue-state";
import type { SQLEditorTab } from "@/types/sqlEditor/tab";
import { tabListEvents } from "@/views/sql-editor/TabList/events";
import { TabContextMenu, type TabContextMenuHandle } from "./TabContextMenu";
import { TabItem } from "./TabItem/TabItem";

type CloseTabAction =
  | "CLOSE"
  | "CLOSE_OTHERS"
  | "CLOSE_TO_THE_RIGHT"
  | "CLOSE_SAVED"
  | "CLOSE_ALL";

type PendingClose = {
  tab: SQLEditorTab;
  resolve: (confirmed: boolean) => void;
};

/**
 * Replaces frontend/src/views/sql-editor/TabList/TabList.vue.
 * Horizontal tab bar at the top of the SQL editor. Drag-reorder via
 * @dnd-kit, overflow-x scroll, "+" button to add a new worksheet, and
 * right-click context menu delegated to TabContextMenu.
 */
export function TabList() {
  const { t } = useTranslation();
  const tabStore = useSQLEditorTabStore();
  const createWorksheet = useSQLEditorStore((s) => s.createWorksheet);

  // `deep: true` so React re-renders when a tab's nested fields change
  // (e.g. `tab.connection` after setConnection, or `tab.status` after
  // auto-save). `tabStore.updateTab` mutates tabs in place via
  // `Object.assign`, so the tab references stay the same and a shallow
  // watcher would miss the change — visible symptom: the tab keeps the
  // disconnect icon until the user hovers (which forces a re-render).
  const tabs = useVueState(() => [...tabStore.openTabList], { deep: true });
  const currentTabId = useVueState(() => tabStore.currentTabId);

  const scrollRef = useRef<HTMLDivElement>(null);
  const contextMenuRef = useRef<TabContextMenuHandle>(null);

  const [loading, setLoading] = useState(false);
  const [scrollState, setScrollState] = useState({
    moreLeft: false,
    moreRight: false,
  });
  const [pendingClose, setPendingClose] = useState<PendingClose | null>(null);

  const recalculateScrollState = useCallback(() => {
    contextMenuRef.current?.hide();
    const el = scrollRef.current;
    if (!el) return;
    const { scrollWidth, offsetWidth, scrollLeft } = el;
    setScrollState({
      moreLeft: scrollLeft > 0,
      moreRight:
        scrollWidth > offsetWidth
          ? scrollLeft + offsetWidth < scrollWidth
          : false,
    });
  }, []);

  // Initial measure + resize observer.
  useLayoutEffect(() => {
    recalculateScrollState();
    const el = scrollRef.current;
    if (!el) return;
    const ro = new ResizeObserver(() => recalculateScrollState());
    ro.observe(el);
    return () => ro.disconnect();
  }, [recalculateScrollState]);

  // Prevent the trackpad / horizontal wheel from triggering browser
  // back/forward navigation when the tab list can't scroll further.
  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    const handler = (e: WheelEvent) => {
      const maxX = el.scrollWidth - el.clientWidth;
      const target = el.scrollLeft + e.deltaX;
      if (target < 0 || target > maxX) {
        e.preventDefault();
      }
    };
    el.addEventListener("wheel", handler, { passive: false });
    return () => el.removeEventListener("wheel", handler);
  }, []);

  // Scroll the selected tab into view whenever the active tab changes.
  useEffect(() => {
    if (!currentTabId) return;
    requestAnimationFrame(() => {
      const el = document.querySelector(
        `[data-tab-id="${currentTabId}"]`
      ) as HTMLElement | null;
      if (el) {
        scrollIntoView(el, { scrollMode: "if-needed" });
      }
    });
  }, [currentTabId]);

  const confirmCloseUnsaved = useCallback(
    (tab: SQLEditorTab) =>
      new Promise<boolean>((resolve) => {
        setPendingClose({ tab, resolve });
      }),
    []
  );

  const removeTab = useCallback(
    async (tab: SQLEditorTab, focusWhenConfirm = false) => {
      if (tab.mode === "WORKSHEET" && tab.status !== "CLEAN") {
        if (focusWhenConfirm) {
          tabStore.setCurrentTabId(tab.id);
        }
        const confirmed = await confirmCloseUnsaved(tab);
        if (!confirmed) return false;
      }
      tabStore.closeTab(tab.id);
      requestAnimationFrame(recalculateScrollState);
      return true;
    },
    [tabStore, confirmCloseUnsaved, recalculateScrollState]
  );

  const handleAddTab = async () => {
    if (loading) return;
    setLoading(true);
    try {
      await createWorksheet({});
      requestAnimationFrame(() => {
        const el = scrollRef.current;
        if (el) el.scrollTo(el.scrollWidth, 0);
        requestAnimationFrame(recalculateScrollState);
      });
    } finally {
      setLoading(false);
    }
  };

  const handleSelect = (tab: SQLEditorTab) => {
    tabStore.setCurrentTabId(tab.id);
  };

  const handleClose = (tab: SQLEditorTab) => {
    void removeTab(tab);
  };

  const handleContextMenu = (
    tab: SQLEditorTab,
    index: number,
    e: React.MouseEvent
  ) => {
    contextMenuRef.current?.show(tab, index, e);
  };

  // Drag-reorder.
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } })
  );

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;
    const oldIndex = tabs.findIndex((t) => t.id === active.id);
    const newIndex = tabs.findIndex((t) => t.id === over.id);
    if (oldIndex < 0 || newIndex < 0) return;
    const next = [...tabs];
    const [moved] = next.splice(oldIndex, 1);
    next.splice(newIndex, 0, moved);
    // The store exposes `openTabList` as a writable computed. Assigning to
    // it reorders the underlying tmp list.
    tabStore.openTabList = next;
  };

  // Listen for close-tab events from the context menu (batch actions).
  useEffect(() => {
    const unsubscribe = tabListEvents.on(
      "close-tab",
      async ({ tab, index, action }) => {
        const snapshot = [...tabStore.openTabList];
        const max = snapshot.length - 1;
        const remove = async (t: SQLEditorTab) => {
          await removeTab(t, true);
          await new Promise((r) => requestAnimationFrame(r));
        };
        switch (action as CloseTabAction) {
          case "CLOSE":
            await remove(tab);
            return;
          case "CLOSE_OTHERS": {
            for (let i = max; i > index; i--) await remove(snapshot[i]);
            for (let i = index - 1; i >= 0; i--) await remove(snapshot[i]);
            return;
          }
          case "CLOSE_TO_THE_RIGHT": {
            for (let i = max; i > index; i--) await remove(snapshot[i]);
            return;
          }
          case "CLOSE_SAVED": {
            for (let i = max; i >= 0; i--) {
              const t = snapshot[i];
              if (t.status === "CLEAN") await remove(t);
            }
            return;
          }
          case "CLOSE_ALL": {
            for (let i = max; i >= 0; i--) await remove(snapshot[i]);
            return;
          }
        }
      }
    );
    return () => {
      unsubscribe();
    };
  }, [tabStore, removeTab]);

  return (
    <div
      className={cn(
        "bb-sql-editor-tab-list flex justify-between items-center",
        "box-border text-control-light text-sm border-b pr-2 gap-1"
      )}
    >
      <div
        ref={scrollRef}
        onScroll={recalculateScrollState}
        className={cn(
          "flex-1 overflow-x-auto overflow-y-hidden scrollbar-thin",
          scrollState.moreLeft && "more-left",
          scrollState.moreRight && "more-right"
        )}
      >
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragEnd={handleDragEnd}
        >
          <SortableContext
            items={tabs.map((tab) => tab.id)}
            strategy={horizontalListSortingStrategy}
          >
            <div className="relative flex flex-nowrap h-9 pt-0.5">
              {tabs.map((tab, index) => (
                <TabItem
                  key={tab.id}
                  tab={tab}
                  index={index}
                  onSelect={handleSelect}
                  onClose={handleClose}
                  onContextMenu={handleContextMenu}
                />
              ))}
              <div className="shrink-0 sticky right-0 bg-background flex items-stretch justify-end">
                <button
                  type="button"
                  className={cn(
                    "bg-control-bg/20 hover:bg-accent/10 py-1 px-1.5",
                    "border-t border-x rounded-t hover:border-accent disabled:opacity-50"
                  )}
                  disabled={loading}
                  onClick={handleAddTab}
                  aria-label={t("common.add")}
                >
                  <Plus className="size-5" strokeWidth={2.5} />
                </button>
              </div>
            </div>
          </SortableContext>
        </DndContext>
      </div>

      {/* Profile menu (avatar + workspace branding) sits to the right of
          the tab scroll area, matching the Vue TabList layout. */}
      <HeaderProfileMenuMount size="small" />

      <TabContextMenu ref={contextMenuRef} />

      <AlertDialog
        open={pendingClose !== null}
        // Vue's confirm dialog used `closeOnEsc: false`, `maskClosable: false`,
        // `closable: false` — the user MUST click Cancel or "Close sheet".
        // Cancel Base UI's close when the reason is Esc / outside-click so
        // the dialog stays open and forces an explicit choice.
        onOpenChange={(
          open: boolean,
          eventDetails?: { reason?: string; cancel?: () => void }
        ) => {
          if (
            !open &&
            (eventDetails?.reason === "escape-key" ||
              eventDetails?.reason === "outside-press")
          ) {
            eventDetails.cancel?.();
            return;
          }
          if (!open && pendingClose) {
            pendingClose.resolve(false);
            setPendingClose(null);
          }
        }}
      >
        <AlertDialogContent className="max-w-md p-6">
          <AlertDialogTitle>
            {t("sql-editor.hint-tips.confirm-to-close-unsaved-sheet.title")}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t("sql-editor.hint-tips.confirm-to-close-unsaved-sheet.content")}
          </AlertDialogDescription>
          <AlertDialogFooter className="mt-4 flex justify-end gap-x-2">
            <Button
              variant="outline"
              onClick={() => {
                pendingClose?.resolve(false);
                setPendingClose(null);
              }}
            >
              {t("common.cancel")}
            </Button>
            <Button
              // Vue's confirm dialog used naive-ui's primary (accent) button
              // for the affirmative action — not destructive (red). Match.
              variant="default"
              onClick={() => {
                pendingClose?.resolve(true);
                setPendingClose(null);
              }}
            >
              {t("sql-editor.close-sheet")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
