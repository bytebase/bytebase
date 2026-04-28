import { shallowRef } from "vue";
import { type IStandaloneCodeEditor } from "@/components/MonacoEditor";

export const activeSQLEditorRef = shallowRef<IStandaloneCodeEditor>();

/**
 * Tracks the live "active statement" — Monaco's delimited statement
 * under the cursor, or the full editor content as fallback. The React
 * `SQLEditor` writes here via `onActiveContentChange`; Vue `EditorMain`
 * reads it when the toolbar's "Run" button is pressed (since
 * `ReactPageMount` doesn't pass refs across the framework boundary).
 */
export const activeStatementRef = shallowRef<string>("");
