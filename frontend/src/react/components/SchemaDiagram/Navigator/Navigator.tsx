import { ChevronLeft } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { PanelSearchBox } from "@/react/components/sql-editor/Panels/common/PanelSearchBox";
import { cn } from "@/react/lib/utils";
import { getInstanceResource, hasSchemaProperty } from "@/utils";
import { useSchemaDiagramContext } from "../common/context";
import { SchemaSelector } from "./SchemaSelector";
import { NavigatorTree } from "./Tree";

interface NavigatorProps {
  /**
   * Optional explicit override for the virtualized tree height. Leave
   * unset to let the tree fill the available vertical space inside the
   * Navigator panel.
   */
  treeHeight?: number;
}

const FALLBACK_TREE_HEIGHT = 480;

/**
 * React port of `Navigator/Navigator.vue`. Collapsible left sidebar
 * holding the schema selector (Postgres-style multi-schema only), a
 * search input, and the schema → table tree.
 */
export function Navigator({ treeHeight }: NavigatorProps) {
  const ctx = useSchemaDiagramContext();
  const {
    databaseMetadata,
    selectedSchemaNames,
    setSelectedSchemaNames,
    database,
  } = ctx;

  const [expanded, setExpanded] = useState(true);
  const [keyword, setKeyword] = useState("");

  const showSchemaSelector = useMemo(
    () => hasSchemaProperty(getInstanceResource(database).engine),
    [database]
  );

  // react-arborist needs a numeric height for virtualization. Measure
  // the tree's flex container so the list fills the panel rather than
  // sitting at a hardcoded 480px (which leaves empty space on tall
  // viewports and clips on short ones).
  const treeContainerRef = useRef<HTMLDivElement | null>(null);
  const [measuredHeight, setMeasuredHeight] = useState<number | null>(null);
  useEffect(() => {
    if (!expanded) return;
    const el = treeContainerRef.current;
    if (!el) return;
    const observer = new ResizeObserver((entries) => {
      const entry = entries[0];
      if (entry) setMeasuredHeight(entry.contentRect.height);
    });
    observer.observe(el);
    return () => observer.disconnect();
  }, [expanded]);

  const effectiveTreeHeight =
    treeHeight ?? measuredHeight ?? FALLBACK_TREE_HEIGHT;

  return (
    <div className="relative h-full">
      <div
        className={cn(
          "bb-schema-diagram--navigator--main h-full overflow-hidden border-y border-control-border flex flex-col transition-all bg-background",
          expanded ? "w-72 shadow-sm border-l" : "w-0"
        )}
      >
        <div className="p-1 flex flex-col gap-y-2 shrink-0">
          {showSchemaSelector && (
            <SchemaSelector
              schemas={databaseMetadata.schemas}
              value={selectedSchemaNames}
              onChange={setSelectedSchemaNames}
            />
          )}
          <PanelSearchBox
            value={keyword}
            onChange={setKeyword}
            className="max-w-none"
          />
        </div>
        <div
          ref={treeContainerRef}
          className="w-full flex-1 overflow-x-hidden overflow-y-auto p-1 px-3 min-h-0"
        >
          <NavigatorTree keyword={keyword} height={effectiveTreeHeight} />
        </div>
      </div>

      {/*
       * The toggle is absolutely positioned and visually crosses the
       * boundary between the Navigator and the Canvas. Two stacking
       * pitfalls to keep in mind:
       *   - `Canvas` is the next flex sibling of `Navigator` inside a
       *     `SchemaDiagram` parent with `overflow-hidden`, so without
       *     `z-10` the half of the button that overhangs into Canvas
       *     gets painted UNDER Canvas (auto z-index + later-DOM-order
       *     wins for siblings in the same stacking context). The
       *     button effectively disappeared into the canvas.
       *   - When collapsed, Navigator's width is 0 so `-left-3` placed
       *     the button at x = -12..+12 relative to the parent's left
       *     edge — the left half got clipped by `overflow-hidden` and
       *     the right 12px was covered by Canvas, leaving the user no
       *     way to re-open. Anchoring the collapsed button at `left-1`
       *     keeps the whole 24px visible inside Canvas, on top.
       */}
      <button
        type="button"
        onClick={() => setExpanded((prev) => !prev)}
        className={cn(
          "absolute z-10 rounded-full shadow-lg w-6 h-6 top-16 flex items-center justify-center bg-background hover:bg-control-bg cursor-pointer transition-all",
          expanded ? "left-full -translate-x-3" : "left-1"
        )}
      >
        <ChevronLeft
          className={cn(
            "size-4 transition-transform",
            !expanded && "-scale-100"
          )}
        />
      </button>
    </div>
  );
}
