import { ChevronDown, ChevronRight, Table as TableIcon } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { cn } from "@/react/lib/utils";
import type {
  DatabaseMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { findAncestor } from "@/utils/dom";
import { useHoverState } from "./hover-state";
import { ColumnIcon, IndexIcon } from "./TreeNode/icons";

const PAGE_SIZE = 500;
/**
 * Hard cap matching Vue's PAGE_SIZE × default page count behavior. The
 * design risk note flagged that `react-window` isn't a dep yet and the
 * table-row count for FlatTableList mode is bounded — pure DOM rendering
 * with a load-more cursor keeps this within ~500 rows (28px × 500 = ~14k
 * px of DOM) per page, which modern browsers handle without
 * virtualization. Revisit only if the cap becomes a real bottleneck.
 */

export interface FlatTableItem {
  readonly key: string;
  readonly schema: string;
  readonly metadata: TableMetadata;
}

type Props = {
  readonly metadata: DatabaseMetadata | undefined;
  readonly database: string;
  readonly search?: string;
  readonly onSelect: (item: FlatTableItem) => void;
  readonly onSelectAll: (item: FlatTableItem) => void;
  readonly onContextMenu: (e: React.MouseEvent, item: FlatTableItem) => void;
};

/**
 * Replaces `SchemaPane/FlatTableList.vue`. Flat list of tables with
 * inline expand-to-show-columns/indexes for large databases (Vue
 * threshold is >1000 tables). Search-filterable, paged at 500 rows
 * with a load-more button at the bottom.
 *
 * Hover sets the SchemaPane's hover state so the shared HoverPanel can
 * preview the table — same wiring as the Tree mode.
 */
export function FlatTableList({
  metadata,
  database,
  search,
  onSelect,
  onSelectAll,
  onContextMenu,
}: Props) {
  const { t } = useTranslation();
  const hoverState = useHoverState();

  const [expanded, setExpanded] = useState<Set<string>>(() => new Set());
  const [selectedKey, setSelectedKey] = useState<string | undefined>();
  const [pageIndex, setPageIndex] = useState(0);

  const filteredTables = useMemo<FlatTableItem[]>(() => {
    const tables: FlatTableItem[] = [];
    const pattern = search?.trim().toLowerCase();
    if (!metadata?.schemas) return tables;
    for (const schema of metadata.schemas) {
      for (const table of schema.tables ?? []) {
        const item: FlatTableItem = {
          key: `${schema.name}/${table.name}`,
          schema: schema.name,
          metadata: table,
        };
        if (!pattern || item.key.toLowerCase().includes(pattern)) {
          tables.push(item);
        }
      }
    }
    return tables;
  }, [metadata, search]);

  // Reset paging + expand state when the underlying list materially
  // changes (e.g. user typed a new search, or switched databases).
  useEffect(() => {
    setPageIndex(0);
    setExpanded(new Set());
  }, [metadata, search]);

  const pagedTables = useMemo(
    () => filteredTables.slice(0, (pageIndex + 1) * PAGE_SIZE),
    [filteredTables, pageIndex]
  );

  const hoverPositionRef = useRef(hoverState.setPosition);
  hoverPositionRef.current = hoverState.setPosition;

  const handleMouseEnter = (
    e: React.MouseEvent<HTMLDivElement>,
    item: FlatTableItem
  ) => {
    const target = {
      database,
      schema: item.schema,
      table: item.metadata.name,
    };
    // 150ms when sliding between rows (state already populated) so the
    // hover panel doesn't flicker; full open delay otherwise.
    const delay = hoverState.state ? 150 : undefined;
    hoverState.update(target, "before", delay);

    // Microtask: position computed against the actual DOM bounding rect,
    // matching Vue's `await nextTick()` before reading getBoundingClientRect.
    const wrapper = findAncestor(e.target as HTMLElement, ".bb-flat-table-row");
    if (!wrapper) {
      hoverState.update(undefined, "after", 150);
      return;
    }
    const bounding = wrapper.getBoundingClientRect();
    hoverState.setPosition({ x: e.clientX, y: bounding.bottom });
  };

  const handleMouseLeave = () => {
    hoverState.update(undefined, "after");
  };

  if (filteredTables.length === 0) {
    // Vue's source uses hardcoded English here; preserve verbatim for
    // 1:1 parity. If we ever i18n this, update the Vue side at the same
    // time so the two surfaces don't drift.
    return (
      <div className="flex flex-col items-center justify-center mt-16 text-control-light text-sm">
        {search ? "No tables found" : "No tables in this database"}
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full overflow-y-auto py-1">
      {pagedTables.map((item, index) => {
        const isExpanded = expanded.has(item.key);
        return (
          <div key={item.key}>
            <div
              className={cn(
                "bb-flat-table-row px-2 py-1 cursor-pointer flex items-center justify-between text-sm",
                "transition-colors hover:bg-control-bg/70",
                selectedKey === item.key && "bg-accent/10 font-medium"
              )}
              onClick={() => {
                setSelectedKey(item.key);
                onSelect(item);
              }}
              onDoubleClick={() => {
                setSelectedKey(item.key);
                onSelectAll(item);
              }}
              onContextMenu={(e) => {
                e.preventDefault();
                setSelectedKey(item.key);
                onContextMenu(e, item);
              }}
              onMouseEnter={(e) => handleMouseEnter(e, item)}
              onMouseLeave={handleMouseLeave}
            >
              <div className="flex items-center gap-1 truncate flex-1 min-w-0">
                <TableIcon className="size-3.5 shrink-0" />
                <span className="truncate">
                  {item.schema ? (
                    <span className="text-gray-500">{item.schema}.</span>
                  ) : null}
                  <span>{item.metadata.name}</span>
                </span>
              </div>
              <div className="flex items-center gap-2 text-xs text-gray-500 shrink-0 ml-2">
                <span>{item.metadata.columns.length} cols</span>
                {item.metadata.columns.length > 0 ? (
                  <Button
                    type="button"
                    variant="ghost"
                    size="xs"
                    className="size-5"
                    onClick={(e) => {
                      e.stopPropagation();
                      setExpanded((prev) => {
                        const next = new Set(prev);
                        if (next.has(item.key)) {
                          next.delete(item.key);
                        } else {
                          next.add(item.key);
                        }
                        return next;
                      });
                    }}
                  >
                    {isExpanded ? (
                      <ChevronDown className="size-3" />
                    ) : (
                      <ChevronRight className="size-3" />
                    )}
                  </Button>
                ) : null}
              </div>
            </div>

            {isExpanded ? (
              <div className="pl-4 pr-2 bg-gray-50 border-l-2 border-gray-200">
                {item.metadata.columns.length > 0 ? (
                  <div className="py-1">
                    <div className="text-xs font-medium text-gray-600 mb-1">
                      {t("database.columns")}
                    </div>
                    {item.metadata.columns.map((column) => (
                      <div
                        key={column.name}
                        className="flex flex-wrap items-center justify-between gap-1 py-0.5 text-xs pl-2"
                      >
                        <div className="flex items-center gap-1">
                          <ColumnIcon className="size-3 text-gray-400" />
                          <span>{column.name}</span>
                        </div>
                        <span className="text-gray-500">{column.type}</span>
                      </div>
                    ))}
                  </div>
                ) : null}
                {item.metadata.indexes && item.metadata.indexes.length > 0 ? (
                  <div className="py-1 border-t border-gray-200">
                    <div className="text-xs font-medium text-gray-600 mb-1">
                      {t("database.indexes")}
                    </div>
                    {item.metadata.indexes.map((index) => (
                      <div
                        key={index.name}
                        className="flex flex-wrap items-center justify-between gap-1 py-0.5 text-xs pl-2"
                      >
                        <div className="flex items-center gap-1">
                          <IndexIcon className="size-3 text-gray-400" />
                          <span>{index.name}</span>
                        </div>
                        <span className="text-gray-500">
                          {index.unique ? "UNIQUE" : ""}{" "}
                          {index.primary ? "PRIMARY" : ""}
                        </span>
                      </div>
                    ))}
                  </div>
                ) : null}
              </div>
            ) : null}

            {index === pagedTables.length - 1 &&
            (pageIndex + 1) * PAGE_SIZE < filteredTables.length ? (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="w-full"
                onClick={() => setPageIndex((p) => p + 1)}
              >
                {t("common.load-more")}
              </Button>
            ) : null}
          </div>
        );
      })}
    </div>
  );
}
