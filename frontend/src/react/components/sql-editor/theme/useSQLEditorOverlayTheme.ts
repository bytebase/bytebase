import { useEffect } from "react";
import { getLayerRoot } from "@/react/components/ui/layer";
import { useSQLEditorEditorState } from "@/react/stores/sqlEditor/editor";
import { themeToCssVars } from "./derive";
import { resolveThemeId } from "./presets";

/**
 * Themes the app-global "overlay" layer root while the SQL Editor is mounted, so
 * EVERY overlay that portals there follows the selected SQL Editor theme:
 * dialogs (Save Sheet, etc.), the request drawer, and the Base-UI
 * Select/Popover/Tooltip popups. These surfaces render outside the SQL Editor's
 * DOM subtree, so the scope's CSS vars can't cascade to them — setting the vars
 * (plus a themed default text color for un-classed text like checkbox labels) on
 * the shared overlay root is the single place that reaches all of them.
 *
 * Reverted on unmount / theme change so the rest of the app's overlays are
 * unaffected. In the light theme the vars equal :root, so this is a visual no-op
 * outside a dark/named theme.
 */
export function useSQLEditorOverlayTheme() {
  const themeId = useSQLEditorEditorState((s) => s.themeId);

  useEffect(() => {
    const root = getLayerRoot("overlay");
    if (!root) return;
    const vars = themeToCssVars(resolveThemeId(themeId).tokens);
    for (const [key, value] of Object.entries(vars)) {
      root.style.setProperty(key, value as string);
    }
    root.style.setProperty("color", "rgb(var(--color-main))");
    return () => {
      for (const key of Object.keys(vars)) {
        root.style.removeProperty(key);
      }
      root.style.removeProperty("color");
    };
  }, [themeId]);
}
