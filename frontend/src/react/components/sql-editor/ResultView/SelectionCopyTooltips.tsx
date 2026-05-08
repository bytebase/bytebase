import { CopyIcon, InfoIcon } from "lucide-react";
import { Trans, useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { useSelectionContext } from "./context";

export function SelectionCopyTooltips() {
  const { t } = useTranslation();
  const {
    state: { rows, columns },
    copySelected,
    deselect,
  } = useSelectionContext();

  if (rows.length === 0 && columns.length === 0) return null;

  const isMac = /mac/i.test(navigator.platform);

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
        <Trans
          i18nKey="sql-editor.copy-selected-results"
          components={{
            action: (
              <Button
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
            ),
            button: (
              <Button
                size="sm"
                variant="default"
                className="h-6 px-2 gap-x-1"
                onClick={copySelected}
              >
                <CopyIcon className="size-3" />
                {t("common.copy")}
              </Button>
            ),
          }}
        />
      </p>
      <div className="ml-1">
        <Button size="sm" variant="ghost" onClick={deselect}>
          {t("sql-editor.cancel-selection")}
        </Button>
      </div>
    </div>
  );
}
