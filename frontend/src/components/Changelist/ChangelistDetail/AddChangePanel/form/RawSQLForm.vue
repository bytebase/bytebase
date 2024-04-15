<template>
  <div class="flex flex-col gap-y-4 flex-1 overflow-hidden">
    <RawSQLEditor
      v-if="sheet"
      v-model:statement="statement"
      :readonly="false"
      class="flex-1 overflow-hidden relative"
      @upload="handleUploadEvent"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification, useLocalSheetStore } from "@/store";
import {
  MAX_UPLOAD_FILE_SIZE_MB,
  getSheetStatement,
  readFileAsArrayBuffer,
  setSheetStatement,
} from "@/utils";
import RawSQLEditor from "../../RawSQLEditor";
import { useAddChangeContext } from "../context";

const { t } = useI18n();
const { changeFromRawSQL: change } = useAddChangeContext();

const sheet = computed(() => {
  return useLocalSheetStore().getOrCreateSheetByName(change.value.sheet);
});

const statement = computed({
  get() {
    return getSheetStatement(sheet.value);
  },
  set(statement) {
    setSheetStatement(sheet.value, statement);
  },
});

const handleUploadEvent = async (e: Event) => {
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

  try {
    const { filename, arrayBuffer } = await readFileAsArrayBuffer(file);
    // TODO(steven): let user choose encoding.
    const decoder = new TextDecoder("utf-8");
    statement.value = decoder.decode(arrayBuffer);
    sheet.value.title = filename;
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: `Read file error`,
      description: String(error),
    });
  }

  cleanup();
};
</script>
