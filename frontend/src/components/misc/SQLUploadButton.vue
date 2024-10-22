<template>
  <NButton tag="div" v-bind="$attrs" @click="handleClick">
    <template #icon>
      <slot name="icon">
        <UploadIcon class="w-4 h-4" />
      </slot>

      <input
        ref="inputRef"
        type="file"
        accept=".sql,.txt,application/sql,text/plain"
        class="hidden"
        @change="handleUpload"
      />
    </template>

    <span v-if="!iconOnly">
      <slot />
    </span>

    <FileContentPreviewModal
      v-if="selectedFile"
      :file="selectedFile"
      @cancel="cleanup"
      @confirm="handleStatementConfirm"
    />
  </NButton>
</template>

<script lang="ts" setup>
import { UploadIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import FileContentPreviewModal from "@/components/FileContentPreviewModal.vue";
import { pushNotification } from "@/store";
import { MAX_UPLOAD_FILE_SIZE_MB } from "@/utils";

defineProps<{
  iconOnly?: boolean;
}>();

const emit = defineEmits<{
  (event: "update:sql", text: string, filename: string): void;
}>();

const { t } = useI18n();
const inputRef = ref<HTMLInputElement>();
const selectedFile = ref<File | null>(null);

const handleClick = () => {
  inputRef.value?.click();
};

const cleanup = () => {
  selectedFile.value = null;
  if (!inputRef.value) {
    return;
  }
  inputRef.value.files = null;
  inputRef.value.value = "";
};

const handleUpload = async (e: Event) => {
  const target = e.target as HTMLInputElement;
  const file = (target.files || [])[0];
  const cleanup = () => {
    // Note that once selected a file, selecting the same file again will not
    // trigger <input type="file">'s change event.
    // So we need to do some cleanup stuff here.
    target.files = null;
    target.value = "";
  };

  if (!file) {
    return cleanup();
  }
  if (file.size > MAX_UPLOAD_FILE_SIZE_MB * 1024 * 1024) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("issue.upload-sql-file-max-size-exceeded", {
        size: `${MAX_UPLOAD_FILE_SIZE_MB}MB`,
      }),
    });
    return cleanup();
  }

  selectedFile.value = file;

  cleanup();
};

const handleStatementConfirm = (statement: string) => {
  emit("update:sql", statement, selectedFile.value!.name);
  cleanup();
};
</script>
