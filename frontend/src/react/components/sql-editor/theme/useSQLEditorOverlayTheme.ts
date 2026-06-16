import { useEffect } from "react";
import { getLayerRoot } from "@/react/components/ui/layer";
import { themeColorScheme, themeToCssVars } from "./derive";
import { useActiveSQLEditorTheme } from "./useActiveSQLEditorTheme";

/**
 * Themes the app-global "overlay" layer root while the SQL Editor is mounted, so
 * EVERY overlay that portals there follows the SQL Editor theme: dialogs (Save
 * Sheet, etc.), the request drawer, and the Base-UI Select/Popover/Tooltip
 * popups. These surfaces render outside the SQL Editor's DOM subtree, so the
 * scope's CSS vars can't cascade to them — setting the vars (plus a themed
 * default text color for un-classed text like checkbox labels) on the shared
 * overlay root is the single place that reaches all of them.
 *
 * Uses the **active** theme (the dark admin fallback in admin mode), not just
 * the selected one. The overlay root is global and admin's foreground is the
 * dark terminal that overlays sit over, so this mirrors how the global Monaco
 * theme is driven — otherwise a light-selected admin tab renders light popups
 * over the dark terminal.
 *
 * Reverted on unmount / theme change so the rest of the app's overlays are
 * unaffected. In the light theme the vars equal :root, so this is a visual no-op
 * outside a dark/named theme.
 */
export function useSQLEditorOverlayTheme() {
  const theme = useActiveSQLEditorTheme();

  useEffect(() => {
    const root = getLayerRoot("overlay");
    if (!root) return;
    const vars = themeToCssVars(theme.tokens);
    for (const [key, value] of Object.entries(vars)) {
      root.style.setProperty(key, value as string);
    }
    root.style.setProperty("color", "rgb(var(--color-main))");
    // Native controls (date pickers, scrollbars) in portaled popups follow
    // color-scheme, not our --color-* tokens.
    root.style.colorScheme = themeColorScheme(theme);
    return () => {
      for (const key of Object.keys(vars)) {
        root.style.removeProperty(key);
      }
      root.style.removeProperty("color");
      root.style.colorScheme = "";
    };
  }, [theme]);
}
