import type * as monaco from "monaco-editor";
import { useEffect, useState } from "react";
import type { IStandaloneCodeEditor } from "./types";

// The find controller is not part of Monaco's public editor API; this is the
// shape of the state it exposes.
type FindController = monaco.editor.IEditorContribution & {
  getState: () => {
    isRevealed: boolean;
    onFindReplaceStateChange: (listener: () => void) => {
      dispose: () => void;
    };
  };
};

// Whether the editor's find widget is open. It owns the top-right corner, so
// other corner overlays yield to it.
export function useMonacoFindWidgetVisible(
  editor: IStandaloneCodeEditor
): boolean {
  const [visible, setVisible] = useState(false);
  useEffect(() => {
    const state = editor
      .getContribution<FindController>("editor.contrib.findController")
      ?.getState();
    if (!state) return;
    const sync = () => setVisible(state.isRevealed);
    sync();
    const subscription = state.onFindReplaceStateChange(sync);
    return () => subscription.dispose();
  }, [editor]);
  return visible;
}
