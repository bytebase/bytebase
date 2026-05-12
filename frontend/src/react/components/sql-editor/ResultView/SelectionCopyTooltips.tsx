import { CopyIcon, InfoIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { useSelectionContext } from "./context";

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
    copySelected,
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
                className="h-6 px-2 gap-x-1"
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
            return (
              <Button
                key={i}
                size="sm"
                variant="default"
                className="h-6 px-2 gap-x-1"
                onClick={copySelected}
              >
                <CopyIcon className="size-3" />
                {t("common.copy")}
              </Button>
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
