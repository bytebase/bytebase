<template>
  <NButton :loading="downloading" @click="downloadSheet">
    {{ $t("common.download") }}
  </NButton>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { NButton } from "naive-ui";
import { ref } from "vue";
import { sheetServiceClientConnect } from "@/connect";
import { GetSheetRequestSchema } from "@/types/proto-es/v1/sheet_service_pb";

const props = defineProps<{
  sheet: string;
}>();

const downloading = ref(false);

const downloadSheet = async () => {
  try {
    downloading.value = true;

    const request = create(GetSheetRequestSchema, {
      name: props.sheet,
      raw: true,
    });
    const response = await sheetServiceClientConnect.getSheet(request);

    // Extract sheet ID from the name to use as filename
    const sheetId = response.name.split("/").pop() || "sheet";
    const filename = `${sheetId}.sql`;
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
