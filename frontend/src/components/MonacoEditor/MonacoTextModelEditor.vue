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
import { useLegacyAutoComplete } from "@/plugins/sql-lsp/client";
import type { SQLDialect } from "@/types";
import {
  type AutoCompleteContext,
  type AutoHeightOptions,
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
import { shouldUseNewLSP } from "./dev";
import monaco, { createMonacoEditor } from "./editor";
import type {
  AdviceOption,
  IStandaloneCodeEditor,
  IStandaloneEditorConstructionOptions,
  ITextModel,
  MonacoModule,
} from "./types";

const props = withDefaults(
  defineProps<{
    model?: ITextModel;
    sqlDialect?: SQLDialect;
    readonly?: boolean;
    autoFocus?: boolean;
    autoHeight?: AutoHeightOptions;
    autoCompleteContext?: AutoCompleteContext;
    advices?: AdviceOption[];
    options?: IStandaloneEditorConstructionOptions;
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
  (e: "ready", monaco: MonacoModule, editor: IStandaloneCodeEditor): void;
}>();

const containerRef = ref<HTMLDivElement>();
// use shallowRef to avoid deep conversion which will cause page crash.
const editorRef = shallowRef<IStandaloneCodeEditor>();
const ready = ref(false);

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
    if (shouldUseNewLSP()) {
      useAutoComplete(monaco, editor, toRef(props, "autoCompleteContext"));
    } else {
      useLegacyAutoComplete(
        monaco,
        editor,
        toRef(props, "autoCompleteContext")
      );
    }

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
