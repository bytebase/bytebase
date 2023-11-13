<template>
  <div ref="containerRef" class="relative">
    <div
      v-if="!isEditorLoaded"
      class="absolute inset-0 flex flex-col items-center justify-center"
    >
      <BBSpin />
    </div>
  </div>
</template>

<script lang="ts" setup>
import type monaco from "monaco-editor";
import { v4 as uuidv4 } from "uuid";
import {
  onMounted,
  ref,
  toRef,
  nextTick,
  watch,
  shallowRef,
  onBeforeUnmount,
  computed,
} from "vue";
import { Language } from "@/types";
import { useOptionByKey } from "./composables";
import { useMonacoTextModel } from "./text-model";
import type { MonacoModule } from "./types";
import { extensionNameOfLanguage } from "./utils";

const props = withDefaults(
  defineProps<{
    original?: string;
    modified?: string;
    language?: Language;
    readonly?: boolean;
  }>(),
  {
    original: "",
    modified: "",
    language: "sql",
    readonly: false,
  }
);

const emit = defineEmits<{
  (e: "update:modified", modified: string): void;
  (
    e: "ready",
    monaco: MonacoModule,
    editor: monaco.editor.IStandaloneDiffEditor
  ): void;
}>();

const containerRef = ref<HTMLDivElement>();
// use shallowRef to avoid deep conversion which will cause page crash.
const editorRef = shallowRef<monaco.editor.IStandaloneDiffEditor>();

const isEditorLoaded = ref(false);

const useDiffModels = (
  monaco: MonacoModule,
  editor: monaco.editor.IStandaloneDiffEditor
) => {
  const language = toRef(props, "language");
  const original = useMonacoTextModel(
    computed(() => `${uuidv4()}.${extensionNameOfLanguage(props.language)}`),
    toRef(props, "original"),
    language
  );
  const modified = useMonacoTextModel(
    computed(() => `${uuidv4()}.${extensionNameOfLanguage(props.language)}`),
    toRef(props, "modified"),
    language
  );

  watch(
    [original, modified],
    () => {
      if (!original.value) return;
      if (!modified.value) return;
      editor.setModel({
        original: original.value,
        modified: modified.value,
      });
    },
    {
      immediate: true,
    }
  );
};

const useModifiedContent = (
  monaco: MonacoModule,
  editor: monaco.editor.IStandaloneDiffEditor
) => {
  const modified = ref(getModifiedContent(editor));
  const update = () => {
    modified.value = getModifiedContent(editor);
  };

  editor.onDidChangeModel(update);
  editor.onDidUpdateDiff(update);

  return modified;
};

const getModifiedContent = (editor: monaco.editor.IStandaloneDiffEditor) => {
  const model = editor.getModel();
  if (!model) return "";

  return model.modified.getValue();
};

onMounted(async () => {
  const { default: monaco, createMonacoDiffEditor } = await import("./editor");

  const container = containerRef.value;
  if (!container) {
    // Give up creating monaco editor if the component has been unmounted
    // very quickly.
    console.debug(
      "<MonacoEditor> has been unmounted before useMonaco is ready"
    );
    return;
  }

  const editor = await createMonacoDiffEditor({ container });
  useDiffModels(monaco, editor);
  // Use "plugin" composable features
  useOptionByKey(monaco, editor, "readOnly", toRef(props, "readonly"));
  const modifiedContent = useModifiedContent(monaco, editor);

  editorRef.value = editor;
  isEditorLoaded.value = true;

  await nextTick();
  nextTick(() => {
    emit("ready", monaco, editor);
  });
  watch(modifiedContent, () => {
    emit("update:modified", modifiedContent.value);
  });
});

onBeforeUnmount(() => {
  editorRef.value?.dispose();
});
</script>

<style lang="postcss">
.monaco-editor .monaco-mouse-cursor-text {
  box-shadow: none !important;
}
.monaco-editor .scroll-decoration {
  display: none !important;
}
.monaco-editor .line-numbers {
  @apply pr-2;
}
</style>
