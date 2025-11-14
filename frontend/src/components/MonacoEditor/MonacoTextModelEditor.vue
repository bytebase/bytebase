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

    <div
      class="absolute top-[3px] right-[18px] flex gap-1 items-center justify-end z-30"
    >
      <slot name="corner-prefix" />

      <NPopover v-if="ready && !readonly" placement="bottom-end">
        <template #trigger>
          <div
            class="w-4 h-4 flex justify-center items-center cursor-pointer opacity-50 hover:opacity-100 transition-all"
          >
            <div
              class="w-3 h-3 rounded-full"
              :class="connectionStateIndicatorClass"
            />
          </div>
        </template>
        <template #default>
          <div class="flex flex-col gap-1">
            <div class="inline-flex gap-1">
              <span>Language server</span>
              <span>{{ connectionStateText }}</span>
            </div>
            <div
              v-if="connectionHeartbeatText"
              class="text-xs text-control-placeholder"
            >
              {{ connectionHeartbeatText }}
            </div>
          </div>
        </template>
      </NPopover>

      <slot name="corner-suffix" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { throttle } from "lodash-es";
import * as monaco from "monaco-editor";
import { NPopover } from "naive-ui";
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  shallowRef,
  toRef,
  watch,
} from "vue";
import { BBSpin } from "@/bbkit";
import type { SQLDialect } from "@/types";
import {
  type AutoCompleteContext,
  type AutoHeightOptions,
  type FormatContentOptions,
  useActiveRangeByCursor,
  useAdvices,
  useAutoComplete,
  useAutoHeight,
  useContent,
  useDecoration,
  useFormatContent,
  useLineHighlights,
  useLSPConnectionState,
  useModel,
  useOptionByKey,
  useOptions,
  useOverrideSuggestIcons,
  useSelectedContent,
  useSelection,
} from "./composables";
import { createMonacoEditor } from "./editor";
import type {
  AdviceOption,
  IStandaloneCodeEditor,
  IStandaloneEditorConstructionOptions,
  ITextModel,
  LineHighlightOption,
  MonacoModule,
  Selection as MonacoSelection,
} from "./types";

const props = withDefaults(
  defineProps<{
    enableDecorations?: boolean;
    model?: ITextModel;
    dialect?: SQLDialect;
    readonly?: boolean;
    autoFocus?: boolean;
    autoHeight?: AutoHeightOptions;
    autoCompleteContext?: AutoCompleteContext;
    advices?: AdviceOption[];
    lineHighlights?: LineHighlightOption[];
    options?: IStandaloneEditorConstructionOptions;
    formatContentOptions?: FormatContentOptions;
    placeholder?: string;
  }>(),
  {
    dialect: undefined,
    readonly: false,
    autoFocus: true,
    autoHeight: undefined,
    autoCompleteContext: undefined,
    advices: () => [],
    lineHighlights: () => [],
    options: undefined,
    formatContentOptions: undefined,
    placeholder: undefined,
  }
);

const emit = defineEmits<{
  (e: "update:content", content: string): void;
  (e: "update:active-content", activeContent: string): void;
  (e: "select-content", content: string): void;
  (e: "update:selection", selection: MonacoSelection | null): void;
  (e: "ready", monaco: MonacoModule, editor: IStandaloneCodeEditor): void;
}>();

const containerRef = ref<HTMLDivElement>();
// use shallowRef to avoid deep conversion which will cause page crash.
const editorRef = shallowRef<IStandaloneCodeEditor>();
const ready = ref(false);
const contentRef = ref("");
const activeContent = ref("");
const { connectionState, connectionHeartbeat } = useLSPConnectionState();
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
const connectionHeartbeatText = computed(() => {
  if (connectionState.value !== "ready") return "";
  const timestamp = connectionHeartbeat.value?.timestamp;
  if (!timestamp) return "";
  const time = dayjs(timestamp).format("YYYY-MM-DD HH:mm:ss.SSS UTCZZ");
  return `Last heartbeat at ${time}`;
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
    useFormatContent(
      monaco,
      editor,
      toRef(props, "dialect"),
      toRef(props, "formatContentOptions")
    );
    const content = useContent(monaco, editor);
    const selection = useSelection(editor);
    const selectedContent = useSelectedContent(editor, selection);
    const activeRangeByCursor = useActiveRangeByCursor(editor);
    useAdvices(monaco, editor, toRef(props, "advices"));
    useLineHighlights(monaco, editor, toRef(props, "lineHighlights"));
    useAutoHeight(monaco, editor, containerRef, toRef(props, "autoHeight"));
    useAutoComplete(
      monaco,
      editor,
      toRef(props, "autoCompleteContext"),
      toRef(props, "readonly")
    );
    useOverrideSuggestIcons(monaco, editor);

    ready.value = true;

    await nextTick();
    emit("ready", monaco, editor);

    if (!props.readonly && props.enableDecorations) {
      useDecoration(monaco, editor, selection, activeRangeByCursor);
    }

    // set the editor focus when the tab is selected
    if (!props.readonly && props.autoFocus) {
      editor.focus();
    }

    contentRef.value = content.value;

    // Combine content watcher with ref update
    watch(content, () => {
      contentRef.value = content.value;
      emit("update:content", content.value);
    });

    // Throttle the selection/activeRange processing to reduce overhead
    const throttledSelectionHandler = throttle(
      ([newSelection, newActiveRangeByCursor]: [
        monaco.Selection | null,
        monaco.IRange | undefined,
      ]) => {
        // Emit selection update
        emit("update:selection", newSelection);

        // Calculate active content
        const hasSelection =
          newSelection &&
          (newSelection.startLineNumber !== newSelection.endLineNumber ||
            newSelection.startColumn !== newSelection.endColumn);
        const activeRange = hasSelection
          ? newSelection
          : newActiveRangeByCursor;
        activeContent.value = activeRange
          ? props.model?.getValueInRange(activeRange) || ""
          : "";
        emit("update:active-content", activeContent.value);
      },
      100, // 100ms throttle for selection updates
      { leading: true, trailing: true }
    );

    // Combine selection and activeRange watchers to avoid duplicate processing
    watch(
      [() => selection.value, () => activeRangeByCursor.value],
      throttledSelectionHandler
    );

    // Keep selectedContent watcher separate as it has different trigger conditions
    watch(
      () => selectedContent.value,
      () => {
        emit("select-content", selectedContent.value);
      }
    );
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
.bb-monaco-editor :deep(.monaco-editor .peekview-title .filename) {
  display: none !important;
}
.bb-monaco-editor :deep(.monaco-editor .peekview-title .dirname) {
  margin-left: 0 !important;
}
.bb-monaco-editor :deep(.monaco-editor) {
  outline: none !important;
  box-shadow: none !important;
}
</style>
