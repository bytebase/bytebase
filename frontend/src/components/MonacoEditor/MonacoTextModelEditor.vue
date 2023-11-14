<template>
  <div ref="containerRef" v-bind="$attrs" class="relative">
    <div
      v-if="!ready"
      class="absolute inset-0 flex flex-col items-center justify-center"
    >
      <BBSpin />
    </div>
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
  watch,
  watchEffect,
} from "vue";
import type { SQLDialect } from "@/types";
import {
  AutoCompleteContext,
  AutoHeightOptions,
  useAdvices,
  useAutoComplete,
  useAutoHeight,
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
    autoHeight?: AutoHeightOptions;
    autoCompleteContext?: AutoCompleteContext;
    advices?: AdviceOption[];
    options?: monaco.editor.IStandaloneEditorConstructionOptions;
  }>(),
  {
    model: undefined,
    sqlDialect: undefined,
    readonly: false,
    autoFocus: true,
    autoHeight: undefined,
    autoCompleteContext: undefined,
    advices: () => [],
    options: undefined,
  }
);

const emit = defineEmits<{
  (e: "update:content", content: string): void;
  (e: "select-content", content: string): void;
  (
    e: "ready",
    monaco: MonacoModule,
    editor: monaco.editor.IStandaloneCodeEditor
  ): void;
}>();

const containerRef = ref<HTMLDivElement>();
// use shallowRef to avoid deep conversion which will cause page crash.
const editorRef = shallowRef<monaco.editor.IStandaloneCodeEditor>();
const ready = ref(false);

onMounted(async () => {
  const { default: monaco, createMonacoEditor } = await import("./editor");

  const container = containerRef.value;
  if (!container) {
    // Give up creating monaco editor if the component has been unmounted
    // very quickly.
    console.debug(
      "<MonacoEditor> has been unmounted before monaco-editor initialized"
    );
    return;
  }

  try {
    const editor = await createMonacoEditor({
      container,
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
    useAutoHeight(monaco, editor, containerRef, toRef(props, "autoHeight"));
    useAutoComplete(monaco, editor, toRef(props, "autoCompleteContext"));

    ready.value = true;

    await nextTick();
    emit("ready", monaco, editor);

    // set the editor focus when the tab is selected
    if (!props.readonly && props.autoFocus) {
      editor.focus();
    }

    watch(content, () => {
      emit("update:content", content.value);
    });
    watchEffect(() => {
      emit("select-content", selectedContent.value);
    });
  } catch (ex) {
    console.error("[MonacoEditor] initialize failed", ex);
  }
});

onBeforeUnmount(() => {
  editorRef.value?.dispose();
});

defineExpose({
  editorRef,
});
</script>
