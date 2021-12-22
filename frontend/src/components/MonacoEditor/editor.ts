import * as monaco from "monaco-editor";
import AutoCompletion from "./AutoCompletion";

const setup = async (triggerCharacters: string[] = []) => {
  monaco.languages.typescript.typescriptDefaults.setCompilerOptions({
    ...monaco.languages.typescript.typescriptDefaults.getCompilerOptions(),
    noUnusedLocals: false,
    noUnusedParameters: false,
    allowUnreachableCode: true,
    allowUnusedLabels: true,
    strict: false,
    allowJs: true,
  });

  const completionItemProvider =
    monaco.languages.registerCompletionItemProvider("mysql", {
      triggerCharacters: [" ", "."],
      provideCompletionItems: (model, position) => {
        // console.log(model)
        // console.log(position)

        const autoCompletion = new AutoCompletion(model, position);

        const suggestions = autoCompletion.getCompletionItemsForKeywords(
          model,
          position
        );

        console.log(suggestions)

        return { suggestions };
      }
    });

  await Promise.all([
    // load workers
    (async () => {
      const [
        { default: EditorWorker }
      ] = await Promise.all([
        import("monaco-editor/esm/vs/editor/editor.worker.js?worker")
      ])

      // @ts-expect-error
      window.MonacoEnvironment = {
        getWorker(_: any, label: string) {
          return new EditorWorker()
        }
      }
    })()
  ])

  return { monaco, completionItemProvider };
};

export default setup;
