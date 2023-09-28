<template>
  <div class="flex flex-col gap-y-2">
    <div class="flex justify-end">
      <UploadProgressButton size="tiny" :upload="handleUploadFile">
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
  // try {
  //   state.isUploadingFile = true;
  //   const { filename, content: statement } = await readFileAsync(event, 100);
  //   handleStatementChange(statement);
  //   if (sheet.value) {
  //     sheet.value.title = filename;
  //   }
  //   resetTempEditState();
  //   updateEditorHeight();
  // } finally {
  //   state.isUploadingFile = false;
  // }
};

const adjustEditorHeight = () => {
  const editor = editorRef.value;
  if (!editor) return;
  editor.setEditorContentHeight(editorWrapperHeight.value);
};

watch(editorWrapperHeight, adjustEditorHeight);
</script>
