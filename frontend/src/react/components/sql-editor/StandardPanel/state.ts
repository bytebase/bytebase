import { type IStandaloneCodeEditor } from "@/react/components/monaco/types";

// Plain mutable holders — only ever read/written imperatively from event
// handlers (no `watch`, no React subscription), so a `{ value }` object
// matches every call site's `.value` access without dragging Vue's
// reactivity system in.
export const activeSQLEditorRef: { value: IStandaloneCodeEditor | undefined } =
  {
    value: undefined,
  };

/**
 * Tracks the live "active statement" — Monaco's delimited statement under
 * the cursor, or the full editor content as fallback. The React
 * `SQLEditor` writes here via `onActiveContentChange`; the React
 * `EditorMain` reads it when the toolbar's "Run" button is pressed.
 */
export const activeStatementRef: { value: string } = { value: "" };
