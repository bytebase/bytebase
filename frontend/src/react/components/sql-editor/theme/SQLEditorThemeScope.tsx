import { createContext, type ReactNode, useContext, useMemo } from "react";
import { cn } from "@/react/lib/utils";
import { themeToCssVars } from "./derive";
import { PRESET_BY_ID } from "./presets";
import type { SQLEditorTheme } from "./types";

const SQLEditorThemeContext = createContext<SQLEditorTheme>(PRESET_BY_ID.light);

export const useSQLEditorTheme = (): SQLEditorTheme =>
  useContext(SQLEditorThemeContext);

type Props = {
  theme: SQLEditorTheme;
  children: ReactNode;
  className?: string;
  // Render the container with `display: contents` (no box) — inline CSS custom
  // properties still cascade to children, so tokens scope without affecting
  // layout. Use for the root and portal wrappers where an extra box would
  // disturb sizing.
  asContents?: boolean;
};

/**
 * Provides `theme` via context AND writes its chrome tokens as inline CSS
 * custom properties so descendant Tailwind classes (text-control, bg-*, …)
 * re-theme via cascade. Nest it to override a subtree (admin terminal, portals).
 */
export function SQLEditorThemeScope({
  theme,
  children,
  className,
  asContents = false,
}: Props) {
  const style = useMemo(() => themeToCssVars(theme.tokens), [theme]);
  return (
    <SQLEditorThemeContext.Provider value={theme}>
      <div className={cn(asContents && "contents", className)} style={style}>
        {children}
      </div>
    </SQLEditorThemeContext.Provider>
  );
}
