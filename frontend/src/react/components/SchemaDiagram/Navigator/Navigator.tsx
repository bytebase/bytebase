import { ChevronLeft } from "lucide-react";
import { useMemo, useState } from "react";
import { PanelSearchBox } from "@/react/components/sql-editor/Panels/common/PanelSearchBox";
import { cn } from "@/react/lib/utils";
import { getInstanceResource, hasSchemaProperty } from "@/utils";
import { useSchemaDiagramContext } from "../common/context";
import { SchemaSelector } from "./SchemaSelector";
import { NavigatorTree } from "./Tree";

interface NavigatorProps {
  /** Override the tree's default height. */
  treeHeight?: number;
}

/**
 * React port of `Navigator/Navigator.vue`. Collapsible left sidebar
 * holding the schema selector (Postgres-style multi-schema only), a
 * search input, and the schema → table tree.
 */
export function Navigator({ treeHeight = 480 }: NavigatorProps) {
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
        <div className="w-full flex-1 overflow-x-hidden overflow-y-auto p-1 pr-2">
          <NavigatorTree keyword={keyword} height={treeHeight} />
        </div>
      </div>

      <button
        type="button"
        onClick={() => setExpanded((prev) => !prev)}
        className={cn(
          "absolute rounded-full shadow-lg w-6 h-6 top-16 flex items-center justify-center bg-background hover:bg-control-bg cursor-pointer transition-all",
          expanded ? "left-full -translate-x-3" : "-left-3"
        )}
      >
        <ChevronLeft
          className={cn(
            "size-4 transition-transform",
            !expanded && "-scale-100 translate-x-[3px]"
          )}
        />
      </button>
    </div>
  );
}
