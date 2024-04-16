<template>
  <div class="flex flex-col gap-y-2">
    <div v-if="!readonly" class="flex justify-end">
      <SQLUploadButton
        :loading="state.isUploadingFile"
        @update:sql="handleUpdateStatement"
      >
        {{ $t("issue.upload-sql") }}
      </SQLUploadButton>
    </div>

    <div
      ref="editorWrapperRef"
      class="flex-1 overflow-hidden relative"
      :data-height="editorWrapperHeight"
    >
      <MonacoEditor
        :content="statement"
        :readonly="readonly || isSheetOversize"
        class="border w-full h-full"
        @update:content="$emit('update:statement', $event)"
        @ready="handleEditorReady"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useElementSize } from "@vueuse/core";
import { reactive, ref, watch } from "vue";
import {
  type IStandaloneCodeEditor,
  type MonacoModule,
  MonacoEditor,
} from "@/components/MonacoEditor";
import SQLUploadButton from "@/components/misc/SQLUploadButton.vue";

interface LocalState {
  isUploadingFile: boolean;
}

defineProps<{
  statement: string;
  readonly: boolean;
  isSheetOversize?: boolean;
}>();

const emit = defineEmits<{
  (event: "update:statement", statement: string): void;
  (event: "upload", e: Event): void;
}>();

const state = reactive<LocalState>({
  isUploadingFile: false,
});
const editorWrapperRef = ref<HTMLDivElement>();
const { height: editorWrapperHeight } = useElementSize(editorWrapperRef);

const handleUpdateStatement = async (statement: string) => {
  try {
    state.isUploadingFile = true;
    emit("update:statement", statement);
  } finally {
    state.isUploadingFile = false;
  }
};

const handleEditorReady = (
  monaco: MonacoModule,
  editor: IStandaloneCodeEditor
) => {
  watch(
    editorWrapperHeight,
    () => {
      const container = editor.getContainerDomNode();
      container.style.height = `${editorWrapperHeight.value}px`;
    },
    { immediate: true }
  );
};
</script>
