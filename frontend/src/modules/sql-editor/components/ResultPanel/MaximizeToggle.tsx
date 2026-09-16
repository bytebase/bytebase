import { Maximize2, Minimize2 } from "lucide-react";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Tooltip } from "@/components/ui/tooltip";
import { useSQLEditorStore } from "@/modules/sql-editor/store";

const OVERLAY_SELECTOR = '[role="dialog"], [role="alertdialog"], [role="menu"]';

/**
 * Whether an overlay is on screen to own the Escape key. A popup kept in the
 * DOM while closed carries `hidden` or Base UI's `data-closed`, and matching
 * one of those would disable Escape for good.
 */
const hasOpenOverlay = () =>
  Array.from(document.querySelectorAll(OVERLAY_SELECTOR)).some(
    (el) => !el.hasAttribute("hidden") && !el.hasAttribute("data-closed")
  );

/**
 * Fills the editor area with the result pane, and back.
 *
 * It sits beside the query tabs rather than inside a result view, so every
 * result — rows, an error, a query plan later — gets the same control.
 *
 * Escape restores, except while a dialog or menu is open: that surface owns
 * the key, and restoring underneath it would close two things with one press.
 */
export function MaximizeToggle() {
  const { t } = useTranslation();
  const maximized = useSQLEditorStore((s) => s.resultPanelMaximized);
  const setMaximized = useSQLEditorStore((s) => s.setResultPanelMaximized);

  useEffect(() => {
    if (!maximized) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Escape" || event.defaultPrevented) return;
      if (hasOpenOverlay()) return;
      setMaximized(false);
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [maximized, setMaximized]);

  const label = maximized ? t("common.restore") : t("common.maximize");
  return (
    <Tooltip content={label} side="bottom">
      <Button
        size="sm"
        appearance="secondary"
        className="mr-1 shrink-0"
        aria-label={label}
        aria-pressed={maximized}
        onClick={() => setMaximized(!maximized)}
      >
        {maximized ? (
          <Minimize2 className="size-4" />
        ) : (
          <Maximize2 className="size-4" />
        )}
      </Button>
    </Tooltip>
  );
}
