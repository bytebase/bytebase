<template>
  <div class="flex flex-col gap-y-2">
    <div v-if="!readonly" class="flex justify-end">
      <UploadProgressButton :upload="handleUploadFile">
        {{ $t("issue.upload-sql") }}
      </UploadProgressButton>
    </div>

    <div
      ref="editorWrapperRef"
      class="flex-1 overflow-hidden"
      :data-height="editorWrapperHeight"
    >
      <MonacoEditor
        ref="editorRef"
        :value="statement"
        :readonly="readonly || isSheetOversize"
        class="border w-full h-full"
        @change="$emit('update:statement', $event)"
        @ready="adjustEditorHeight"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useElementSize } from "@vueuse/core";
import { ref, watch } from "vue";
import MonacoEditor from "@/components/MonacoEditor";
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
const editorRef = ref<InstanceType<typeof MonacoEditor>>();
const { height: editorWrapperHeight } = useElementSize(editorWrapperRef);

const handleUploadFile = async (event: Event) => {
  emit("upload", event);
};

const adjustEditorHeight = () => {
  const editor = editorRef.value;
  if (!editor) return;
  editor.setEditorContentHeight(editorWrapperHeight.value);
};

watch(editorWrapperHeight, adjustEditorHeight);
</script>
