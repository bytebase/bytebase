<template>
  <NButton :loading="downloading" @click="downloadSheet">
    {{ $t("common.download") }}
  </NButton>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { NButton } from "naive-ui";

import { sheetServiceClient } from "@/grpcweb";

const props = defineProps<{
  sheet: string;
}>();

const downloading = ref(false);

const downloadSheet = async () => {
  try {
    downloading.value = true;

    const response = await sheetServiceClient.getSheet({
      name: props.sheet,
      raw: true,
    });

    let filename = response.title;
    if (!filename.endsWith(".sql")) {
      filename = `${response.title}.sql`;
    }
    const content = new TextDecoder().decode(response.content);

    const blob = new Blob([content], { type: "text/plain" });
    const downloadLink = document.createElement("a");
    downloadLink.href = URL.createObjectURL(blob);
    downloadLink.download = filename;
    document.body.appendChild(downloadLink);
    downloadLink.click();
    URL.revokeObjectURL(downloadLink.href);
  } finally {
    downloading.value = false;
  }
};
</script>
