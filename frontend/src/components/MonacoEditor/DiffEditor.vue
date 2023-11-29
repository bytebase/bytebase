<template>
  <div ref="containerRef" class="relative bb-monaco-diff-editor">
    <div
      v-if="!isEditorLoaded"
      class="absolute inset-0 flex flex-col items-center justify-center"
    >
      <BBSpin />
    </div>
  </div>
</template>

<script lang="ts" setup>
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
import type { Language } from "@/types";
import type { AutoHeightOptions } from "./composables";
import { useAutoHeight, useOptionByKey } from "./composables";
import monaco, { createMonacoDiffEditor } from "./editor";
import { useMonacoTextModel } from "./text-model";
import type { IStandaloneDiffEditor, MonacoModule } from "./types";
import { extensionNameOfLanguage } from "./utils";

export type DiffEditorAutoHeightOptions = AutoHeightOptions & {
  alignment: "original" | "modified";
};

const props = withDefaults(
  defineProps<{
    original?: string;
    modified?: string;
    language?: Language;
    readonly?: boolean;
    autoHeight?: DiffEditorAutoHeightOptions;
  }>(),
  {
    original: "",
    modified: "",
    language: "sql",
    readonly: false,
    autoHeight: undefined,
  }
);

const emit = defineEmits<{
  (e: "update:modified", modified: string): void;
  (e: "ready", monaco: MonacoModule, editor: IStandaloneDiffEditor): void;
}>();

const containerRef = ref<HTMLDivElement>();
// use shallowRef to avoid deep conversion which will cause page crash.
const editorRef = shallowRef<IStandaloneDiffEditor>();

const isEditorLoaded = ref(false);

const useDiffModels = (monaco: MonacoModule, editor: IStandaloneDiffEditor) => {
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
  editor: IStandaloneDiffEditor
) => {
  const modified = ref(getModifiedContent(editor));
  const update = () => {
    modified.value = getModifiedContent(editor);
  };

  editor.onDidChangeModel(update);
  editor.onDidUpdateDiff(update);

  return modified;
};

const getModifiedContent = (editor: IStandaloneDiffEditor) => {
  const model = editor.getModel();
  if (!model) return "";

  return model.modified.getValue();
};

onMounted(async () => {
  const container = containerRef.value;
  if (!container) {
    // Give up creating monaco editor if the component has been unmounted
    // very quickly.
    console.debug(
      "<MonacoEditor> has been unmounted before monaco-editor initialized"
    );
    return;
  }

  const editor = await createMonacoDiffEditor({ container });
  useDiffModels(monaco, editor);
  // Use "plugin" composable features
  useOptionByKey(monaco, editor, "readOnly", toRef(props, "readonly"));
  const modifiedContent = useModifiedContent(monaco, editor);
  if (props.autoHeight) {
    useAutoHeight(
      monaco,
      props.autoHeight.alignment === "original"
        ? editor.getOriginalEditor()
        : editor.getModifiedEditor(),
      containerRef,
      toRef(props, "autoHeight")
    );
  }

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

<style lang="postcss" scoped>
.bb-monaco-diff-editor :deep(.monaco-editor .monaco-mouse-cursor-text) {
  box-shadow: none !important;
}
.bb-monaco-diff-editor :deep(.monaco-editor .scroll-decoration) {
  display: none !important;
}
.bb-monaco-diff-editor :deep(.monaco-editor .line-numbers) {
  @apply pr-2;
}
</style>
