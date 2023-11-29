<template>
  <div class="flex flex-col gap-y-2">
    <div v-if="!readonly" class="flex justify-end">
      <UploadProgressButton :upload="handleUploadFile">
        {{ $t("issue.upload-sql") }}
      </UploadProgressButton>
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
import { ref, watch } from "vue";
import {
  type IStandaloneCodeEditor,
  type MonacoModule,
  MonacoEditor,
} from "@/components/MonacoEditor";
import UploadProgressButton from "@/components/misc/UploadProgressButton.vue";

defineProps<{
  statement: string;
  readonly: boolean;
  isSheetOversize?: boolean;
}>();

const emit = defineEmits<{
  (event: "update:statement", statement: string): void;
  (event: "upload", e: Event): void;
}>();

const editorWrapperRef = ref<HTMLDivElement>();
const { height: editorWrapperHeight } = useElementSize(editorWrapperRef);

const handleUploadFile = async (event: Event) => {
  emit("upload", event);
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
