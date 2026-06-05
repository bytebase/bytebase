import { CopyIcon } from "lucide-react";
import type { ReactElement } from "react";
import { useTranslation } from "react-i18next";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger,
} from "@/react/components/ui/context-menu";
import { useSelectionContext } from "./context";
import { formatAsCSV, formatAsSQL, formatAsText } from "./copy-formats";

/**
 * Right-click context menu for the result grid. Wraps the grid's scroll
 * container (passed as a single element via Base UI's `render` pattern) and
 * offers a plain "Copy", "Copy selected rows as CSV", and — for SQL engines —
 * "Copy selected rows as SQL". When copying is disallowed the children are
 * returned untouched.
 */
export function ResultCopyContextMenu({
  children,
}: {
  children: ReactElement;
}) {
  const { t } = useTranslation();
  const { state, disabled, copy, canCopyAsInsert } = useSelectionContext();

  if (disabled) return children;

  // Plain "Copy" honors any selection (cells / rows / columns). CSV and SQL are
  // row-oriented: they only narrow when rows are selected, so their scope and
  // label track row selection — otherwise they (accurately) copy all rows.
  const hasSelection = state.rows.length > 0 || state.columns.length > 0;
  const hasRowSelection = state.rows.length > 0;
  const scope = hasRowSelection ? "selected" : "all";

  return (
    <ContextMenu>
      <ContextMenuTrigger render={children} />
      <ContextMenuContent>
        <ContextMenuItem
          onClick={() => copy(hasSelection ? "selected" : "all", formatAsText)}
        >
          <CopyIcon className="size-4" />
          {hasSelection ? t("common.copy") : t("common.copy-all")}
        </ContextMenuItem>
        <ContextMenuItem onClick={() => copy(scope, formatAsCSV)}>
          <CopyIcon className="size-4" />
          {hasRowSelection
            ? t("sql-editor.copy-selected-rows-as-csv")
            : t("sql-editor.copy-all-rows-as-csv")}
        </ContextMenuItem>
        {canCopyAsInsert && (
          <ContextMenuItem onClick={() => copy(scope, formatAsSQL)}>
            <CopyIcon className="size-4" />
            {hasRowSelection
              ? t("sql-editor.copy-selected-rows-as-sql")
              : t("sql-editor.copy-all-rows-as-sql")}
          </ContextMenuItem>
        )}
      </ContextMenuContent>
    </ContextMenu>
  );
}
