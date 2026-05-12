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
      // `bg-background` is the light page bg and doesn't dark-switch
      // automatically. The tip is `absolute h-full` over the result
      // toolbar — without an opaque dark bg under it the white shows
      // through admin mode's `bg-dark-bg`. Match the result-view chrome
      // for both themes.
      className="w-full absolute h-full bg-background dark:bg-dark-bg flex flex-row justify-start items-center text-control dark:text-gray-100"
      onClick={(e) => {
        e.preventDefault();
        e.stopPropagation();
      }}
    >
      <InfoIcon size={16} className="mr-2 text-control dark:text-gray-100" />
      <p className="text-sm flex flex-row justify-start items-center gap-1">
        {tokens.map((token, i) => {
          if (token === "action") {
            return (
              <Button
                key={i}
                size="sm"
                variant="outline"
                // `outline` is `bg-transparent + text-control` and the
                // tip sits over `bg-dark-bg` in admin mode → invisible.
                // Force an opaque dark surface with a light shortcut
                // label so the keyboard hint reads against the toolbar.
                className="h-6 px-2 gap-x-1 dark:bg-gray-700 dark:text-gray-100 dark:border-zinc-600 dark:disabled:opacity-100"
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
        {/*
         * `ghost` is `text-control` with no background — invisible
         * label against the admin-mode dark backdrop. Lift the label
         * to a light gray and give it a hover surface so the cancel
         * action is reachable.
         */}
        <Button
          size="sm"
          variant="ghost"
          className="dark:text-gray-100 dark:hover:bg-gray-700"
          onClick={deselect}
        >
          {t("sql-editor.cancel-selection")}
        </Button>
      </div>
    </div>
  );
}
