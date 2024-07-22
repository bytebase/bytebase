<template>
  <div ref="containerRef" v-bind="$attrs" class="relative bb-monaco-editor">
    <div
      v-if="!ready"
      class="absolute inset-0 flex flex-col items-center justify-center"
    >
      <BBSpin />
    </div>
    <div
      v-if="ready && !contentRef && placeholder"
      class="bb-monaco-editor--placeholder absolute pointer-events-none font-mono text-control-placeholder"
    >
      {{ placeholder }}
    </div>

    <NPopover v-if="ready" placement="left">
      <template #trigger>
        <div
          class="absolute top-[3px] right-[18px] w-4 h-4 flex items-center justify-center cursor-pointer z-50 opacity-50 hover:opacity-100 transition-all"
        >
          <div
            class="w-3 h-3 rounded-full"
            :class="connectionStateIndicatorClass"
          />
        </div>
      </template>
      <template #default>
        <div class="inline-flex gap-1">
          <span>Language server</span>
          <span>{{ connectionStateText }}</span>
        </div>
      </template>
    </NPopover>
  </div>
</template>

<script lang="ts" setup>
import { NPopover } from "naive-ui";
import {
  onMounted,
  ref,
  toRef,
  nextTick,
  shallowRef,
  onBeforeUnmount,
  watch,
  watchEffect,
  computed,
} from "vue";
import { BBSpin } from "@/bbkit";
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
  useLineHighlights,
  useLSPConnectionState,
} from "./composables";
import monaco, { createMonacoEditor } from "./editor";
import type {
  AdviceOption,
  IStandaloneCodeEditor,
  IStandaloneEditorConstructionOptions,
  ITextModel,
  LineHighlightOption,
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
    lineHighlights?: LineHighlightOption[];
    options?: IStandaloneEditorConstructionOptions;
    placeholder?: string;
  }>(),
  {
    model: undefined,
    sqlDialect: undefined,
    readonly: false,
    autoFocus: true,
    autoHeight: undefined,
    autoCompleteContext: undefined,
    advices: () => [],
    lineHighlights: () => [],
    options: undefined,
    placeholder: undefined,
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
const contentRef = ref("");
const { connectionState } = useLSPConnectionState();
const connectionStateIndicatorClass = computed(() => {
  const state = connectionState.value;
  if (state === "ready") {
    return "bg-green-500";
  }
  if (state === "initial" || state === "reconnecting") {
    return "bg-yellow-500";
  }
  return "bg-gray-500";
});
const connectionStateText = computed(() => {
  const state = connectionState.value;
  if (state === "ready") {
    return "connected";
  }
  if (state === "initial" || state === "reconnecting") {
    return "connecting";
  }
  return "disconnected";
});

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
    useLineHighlights(monaco, editor, toRef(props, "lineHighlights"));
    useAutoHeight(monaco, editor, containerRef, toRef(props, "autoHeight"));
    useAutoComplete(monaco, editor, toRef(props, "autoCompleteContext"));

    ready.value = true;

    await nextTick();
    emit("ready", monaco, editor);

    // set the editor focus when the tab is selected
    if (!props.readonly && props.autoFocus) {
      editor.focus();
    }

    contentRef.value = content.value;
    watch(content, () => {
      emit("update:content", content.value);
      contentRef.value = content.value;
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
  get codeEditor() {
    return editorRef.value;
  },
});
</script>

<style>
.editor-widget.suggest-widget .signature-label {
  margin-left: 0.5rem;
}
</style>

<style scoped>
.bb-monaco-editor .bb-monaco-editor--placeholder {
  font-family: "Droid Sans Mono", monospace;
  font-size: 14px;
  left: 52px;
  top: 8px;
}
</style>
