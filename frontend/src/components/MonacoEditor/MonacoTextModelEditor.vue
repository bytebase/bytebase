<template>
  <div ref="containerRef" v-bind="$attrs" class="relative">
    <div
      v-if="!isEditorLoaded"
      class="absolute inset-0 flex flex-col items-center justify-center"
    >
      <BBSpin />
    </div>
  </div>
  <div
    class="fixed right-0 bottom-0 flex flex-col gap-y-2 text-xs font-mono bg-red-300/50 z-50"
  >
    <div>isEditorLoaded: {{ isEditorLoaded }}</div>
  </div>
</template>

<script lang="ts" setup>
import type monaco from "monaco-editor";
import {
  onMounted,
  ref,
  toRef,
  nextTick,
  shallowRef,
  onBeforeUnmount,
} from "vue";
import { watchEffect } from "vue";
import { SQLDialect } from "@/types";
import {
  useAdvices,
  useContent,
  useFormatContent,
  useModel,
  useOptionByKey,
  useOptions,
  useSelectedContent,
  useSuggestOptionByLanguage,
} from "./composables";
import type { AdviceOption, MonacoModule } from "./types";

const props = withDefaults(
  defineProps<{
    model?: monaco.editor.ITextModel;
    sqlDialect?: SQLDialect;
    readonly?: boolean;
    autoFocus?: boolean;
    advices?: AdviceOption[];
    options?: monaco.editor.IStandaloneEditorConstructionOptions;
  }>(),
  {
    model: undefined,
    sqlDialect: undefined,
    readonly: false,
    autoFocus: true,
    advices: () => [],
    options: undefined,
  }
);

const emit = defineEmits<{
  (e: "update:content", content: string): void;
  (e: "update:selected-content", content: string): void;
  (e: "save", content: string): void;
  (
    e: "ready",
    monaco: MonacoModule,
    editor: monaco.editor.IStandaloneCodeEditor
  ): void;
}>();

const containerRef = ref<HTMLDivElement>();
// use shallowRef to avoid deep conversion which will cause page crash.
const editorRef = shallowRef<monaco.editor.IStandaloneCodeEditor>();

const isEditorLoaded = ref(false);

onMounted(async () => {
  const { initializeMonacoServices } = await import("./services");
  await initializeMonacoServices();

  const { default: monaco, createMonacoEditor } = await import("./editor");
  if (!containerRef.value) {
    // Give up creating monaco editor if the component has been unmounted
    // very quickly.
    console.debug(
      "<MonacoEditor> has been unmounted before monaco-editor initialized"
    );
    return;
  }

  try {
    const editor = await createMonacoEditor({
      container: containerRef.value,
      options: {
        readOnly: props.readonly,
        ...props.options,
      },
    });
    editorRef.value = editor;

    // Use "plugin" composable features
    useOptionByKey(monaco, editor, "readOnly", toRef(props, "readonly"));
    useOptions(monaco, editor, toRef(props, "options"));
    useModel(monaco, editor, toRef(props, "model"));
    useSuggestOptionByLanguage(monaco, editor);
    useFormatContent(monaco, editor, toRef(props, "sqlDialect"));
    const content = useContent(monaco, editor);
    const selectedContent = useSelectedContent(monaco, editor);
    useAdvices(monaco, editor, toRef(props, "advices"));

    isEditorLoaded.value = true;
    await nextTick();
    emit("ready", monaco, editor);

    // set the editor focus when the tab is selected
    if (!props.readonly && props.autoFocus) {
      editor.focus();
    }

    watchEffect(() => {
      emit("update:content", content.value);
    });
    watchEffect(() => {
      emit("update:selected-content", selectedContent.value);
    });
  } catch (ex) {
    console.error("[MonacoEditorV2] initialize failed", ex);
  }
});

onBeforeUnmount(() => {
  editorRef.value?.dispose();
});

defineExpose({
  editorRef,
});
</script>

<style lang="postcss" scoped></style>
