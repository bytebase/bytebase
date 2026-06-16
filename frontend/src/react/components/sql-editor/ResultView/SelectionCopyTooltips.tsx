import { ChevronDownIcon, CopyIcon, InfoIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
import { useSelectionContext } from "./context";
import { formatAsCSV, formatAsSQL, formatAsText } from "./copy-formats";

// react-i18next's `Trans` wipes children when placeholder tags are empty
// (`<action></action>` calls `React.cloneElement(component, {}, ...[])`,
// which replaces the original children with nothing). The Vue `i18n-t`
// preserved slot content; react-i18next doesn't. We split the localized
// template manually at the placeholder positions so the React Buttons
// keep their own children.
function splitTemplate(template: string) {
  const tokens: Array<string | "action" | "button"> = [];
  const regex = /<action><\/action>|<button><\/button>/g;
  let lastIndex = 0;
  let match: RegExpExecArray | null;
  while ((match = regex.exec(template)) !== null) {
    if (match.index > lastIndex) {
      tokens.push(template.slice(lastIndex, match.index));
    }
    tokens.push(match[0] === "<action></action>" ? "action" : "button");
    lastIndex = regex.lastIndex;
  }
  if (lastIndex < template.length) tokens.push(template.slice(lastIndex));
  return tokens;
}

export function SelectionCopyTooltips() {
  const { t } = useTranslation();
  const {
    state: { rows, columns },
    copy,
    canCopyAsInsert,
    deselect,
  } = useSelectionContext();

  if (rows.length === 0 && columns.length === 0) return null;

  const isMac = /mac/i.test(navigator.platform);
  const tokens = splitTemplate(t("sql-editor.copy-selected-results"));

  return (
    <div
      className="w-full absolute h-full bg-background flex flex-row justify-start items-center text-control"
      onClick={(e) => {
        e.preventDefault();
        e.stopPropagation();
      }}
    >
      <InfoIcon size={16} className="mr-2 text-control" />
      <p className="text-sm flex flex-row justify-start items-center gap-1">
        {tokens.map((token, i) => {
          if (token === "action") {
            return (
              <Button
                key={i}
                size="sm"
                variant="outline"
                className="h-6 px-2 gap-x-1 bg-control-bg text-control border-control-border"
                disabled
              >
                {isMac ? (
                  <span className="text-base leading-none">⌘</span>
                ) : (
                  <span className="tracking-tighter text-xs leading-none">
                    Ctrl
                  </span>
                )}
                C
              </Button>
            );
          }
          if (token === "button") {
            // Single split button: click copies as plain text; hovering the
            // caret reveals CSV and, for SQL engines, SQL.
            return (
              <span key={i} className="inline-flex items-center">
                <Button
                  size="sm"
                  variant="default"
                  className="h-6 px-2 gap-x-1 rounded-r-none"
                  onClick={() => copy("selected", formatAsText)}
                >
                  <CopyIcon className="size-3" />
                  {t("common.copy")}
                </Button>
                <DropdownMenu>
                  <DropdownMenuTrigger
                    openOnHover
                    delay={100}
                    render={
                      <Button
                        size="sm"
                        variant="default"
                        aria-label={t("common.copy")}
                        className="h-6 w-5 px-0 rounded-l-none border-l border-white/30"
                      >
                        <ChevronDownIcon className="size-3" />
                      </Button>
                    }
                  />
                  <DropdownMenuContent align="end" className="min-w-0">
                    <DropdownMenuItem
                      onClick={() => copy("selected", formatAsCSV)}
                      className="px-2 py-1 text-xs gap-x-1.5"
                    >
                      <CopyIcon className="size-3" />
                      {t("sql-editor.copy-selected-rows-as-csv")}
                    </DropdownMenuItem>
                    {canCopyAsInsert && (
                      <DropdownMenuItem
                        onClick={() => copy("selected", formatAsSQL)}
                        className="px-2 py-1 text-xs gap-x-1.5"
                      >
                        <CopyIcon className="size-3" />
                        {t("sql-editor.copy-selected-rows-as-sql")}
                      </DropdownMenuItem>
                    )}
                  </DropdownMenuContent>
                </DropdownMenu>
              </span>
            );
          }
          return <span key={i}>{token}</span>;
        })}
      </p>
      <div className="ml-1">
        <Button size="sm" variant="ghost" onClick={deselect}>
          {t("sql-editor.cancel-selection")}
        </Button>
      </div>
    </div>
  );
}
