import type * as MonacoType from "monaco-editor";
import { isRef, markRaw, ref, shallowRef, unref, watch } from "vue";
import type { Language, MaybeRef } from "@/types";
import { MonacoEditorReady } from "./editor";
import { getMonacoEditor } from "./lazy-editor";

const ready = ref(false);

MonacoEditorReady.then(() => (ready.value = true));

// Store TextModel uniq by filename.
const TextModelMapByFilename = new Map<string, MonacoType.editor.ITextModel>();

export const getUriByFilename = async (filename: string) => {
  const monaco = await getMonacoEditor();
  return monaco.Uri.parse(`file:///workspace/${filename}`);
};

const createTextModel = async (
  filename: string,
  content: string,
  language: string
) => {
  console.debug("[createTextModel]", filename);
  if (TextModelMapByFilename.has(filename)) {
    return TextModelMapByFilename.get(filename)!;
  }

  const monaco = await getMonacoEditor();
  const uri = await getUriByFilename(filename);
  const model = monaco.editor.createModel(content, language, uri);
  TextModelMapByFilename.set(filename, model);
  return model;
};

export const useMonacoTextModel = (
  filename: MaybeRef<string>,
  content: MaybeRef<string>,
  language: MaybeRef<Language>,
  sync: boolean = true
) => {
  const model = shallowRef<MonacoType.editor.ITextModel>();

  watch(
    [ready, () => unref(filename), () => unref(language)],
    async ([ready, filename, language]) => {
      if (!ready) return;
      const m = markRaw(
        await createTextModel(filename, unref(content), language)
      );

      if (sync && isRef(content)) {
        m.onDidChangeContent(() => {
          const c = m.getValue();
          if (c !== content.value) {
            // Write-back edited content to content ref.
            content.value = c;
          }
        });
      }

      model.value = m;
    },
    { immediate: true }
  );

  watch(
    [model, () => unref(content)],
    ([model, content]) => {
      if (!model) return;
      if (model.getValue() === content) return;
      model.setValue(content);
    },
    { immediate: true }
  );

  return model;
};
