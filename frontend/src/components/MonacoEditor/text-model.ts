import { Uri, editor } from "monaco-editor";
import slug from "slug";
import { isRef, markRaw, ref, shallowRef, unref, watch } from "vue";
import type { Language, MaybeRef } from "@/types";
import { MonacoEditorReady } from "./editor";

const ready = ref(false);

MonacoEditorReady.then(() => (ready.value = true));

// Store TextModel uniq by filename
const TextModelMapByFilename = new Map<string, editor.ITextModel>();

export const createTextModel = (
  filename: string,
  content: string,
  language: string
) => {
  console.debug("[createTextModel]", filename);
  const normalizedFilename = slug(filename);
  if (TextModelMapByFilename.has(normalizedFilename)) {
    return TextModelMapByFilename.get(normalizedFilename)!;
  }

  const uri = Uri.parse(`/workspace/${normalizedFilename}`);
  const model = editor.createModel(content, language, uri);
  TextModelMapByFilename.set(normalizedFilename, model);
  return model;
};

export const useMonacoTextModel = (
  filename: MaybeRef<string>,
  content: MaybeRef<string>,
  language: MaybeRef<Language>,
  sync: boolean = true
) => {
  const model = shallowRef<editor.ITextModel>();

  watch(
    [ready, () => unref(filename), () => unref(language)],
    ([ready, filename, language]) => {
      if (!ready) return;
      const m = markRaw(createTextModel(filename, unref(content), language));

      if (sync && isRef(content)) {
        m.onDidChangeContent(() => {
          const c = m.getValue();
          if (c !== content.value) {
            // Write-back edited content to content ref
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
