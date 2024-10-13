import { shallowRef } from "vue";
import { type IStandaloneCodeEditor } from "@/components/MonacoEditor";

export const activeSQLEditorRef = shallowRef<IStandaloneCodeEditor>();
