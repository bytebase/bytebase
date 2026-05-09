import {
  Database as DatabaseIcon,
  FileCode,
  FunctionSquare,
  Table2,
  View,
  X,
} from "lucide-react";
import { useEffect, useRef } from "react";
import { EllipsisText } from "@/react/components/ui/ellipsis-text";
import { cn } from "@/react/lib/utils";
import { extractDatabaseResourceName } from "@/utils";
import { useSchemaEditorContext } from "./context";
import type { EditStatus, TabContext } from "./types";

export function TabsContainer() {
  const { tabs, editStatus } = useSchemaEditorContext();
  const { tabList, currentTab, setCurrentTab, closeTab } = tabs;
  const activeTabRef = useRef<HTMLDivElement>(null);

  // Auto-scroll active tab into view
  useEffect(() => {
    if (activeTabRef.current) {
      activeTabRef.current.scrollIntoView({
        behavior: "smooth",
        block: "nearest",
        inline: "nearest",
      });
    }
  }, [currentTab?.id]);

  if (tabList.length === 0) return null;

  return (
    <div className="flex items-center justify-between border-b border-control-border">
      <div className="flex flex-1 items-center gap-x-1 overflow-x-auto px-1 py-1 scrollbar-hide">
        {tabList.map((tab) => {
          const isActive = tab.id === currentTab?.id;
          const status = getTabStatus(tab, editStatus);
          return (
            <div
              key={tab.id}
              ref={isActive ? activeTabRef : undefined}
              className={cn(
                "group flex w-40 shrink-0 cursor-pointer items-center gap-x-1 rounded-xs px-2 py-1 text-xs transition-colors",
                isActive
                  ? "bg-control-bg-hover text-control"
                  : "text-control-light hover:bg-control-bg-hover",
                status === "dropped" && "text-error line-through",
                status === "created" && "text-success",
                status === "updated" && "text-warning"
              )}
              onClick={() => setCurrentTab(tab.id)}
            >
              <TabIcon type={tab.type} />
              <EllipsisText text={getTabName(tab)} className="flex-1" />
              <button
                className="ml-auto hidden size-4 shrink-0 items-center justify-center rounded-xs hover:bg-control-border group-hover:flex"
                onClick={(e) => {
                  e.stopPropagation();
                  closeTab(tab.id);
                }}
              >
                <X className="size-3" />
              </button>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function TabIcon({ type }: { type: TabContext["type"] }) {
  const cls = "size-3.5 shrink-0";
  switch (type) {
    case "database":
      return <DatabaseIcon className={cls} />;
    case "table":
      return <Table2 className={cls} />;
    case "view":
      return <View className={cls} />;
    case "procedure":
      return <FileCode className={cls} />;
    case "function":
      return <FunctionSquare className={cls} />;
  }
}

function getTabName(tab: TabContext): string {
  const meta = tab.metadata as Record<string, { name: string }>;
  if (tab.type === "database") {
    return extractDatabaseResourceName(meta.database.name).databaseName;
  }
  const schema = meta.schema?.name;
  const objectName =
    meta.table?.name ??
    meta.view?.name ??
    meta.procedure?.name ??
    meta.function?.name ??
    "";
  if (schema) {
    return `${schema}.${objectName}`;
  }
  return objectName;
}

function getTabStatus(
  tab: TabContext,
  editStatus: ReturnType<typeof useSchemaEditorContext>["editStatus"]
): EditStatus {
  if (tab.type === "table") {
    return editStatus.getTableStatus(tab.database, tab.metadata);
  }
  return "normal";
}
