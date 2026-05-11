import { shallowRef } from "vue";
import { type IStandaloneCodeEditor } from "@/components/MonacoEditor";

export const activeSQLEditorRef = shallowRef<IStandaloneCodeEditor>();

/**
 * Tracks the live "active statement" — Monaco's delimited statement
 * under the cursor, or the full editor content as fallback. The React
 * `SQLEditor` writes here via `onActiveContentChange`; the React
 * `EditorMain` reads it when the toolbar's "Run" button is pressed
 * (and Stage 22's still-Vue AI plugin can read it via the same Vue
 * `shallowRef` since the bridge shares Vue's reactivity system).
 */
export const activeStatementRef = shallowRef<string>("");
