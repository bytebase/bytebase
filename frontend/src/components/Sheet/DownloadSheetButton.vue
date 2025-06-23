<template>
  <NButton :loading="downloading" @click="downloadSheet">
    {{ $t("common.download") }}
  </NButton>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { ref } from "vue";
import { create } from "@bufbuild/protobuf";
import { sheetServiceClientConnect } from "@/grpcweb";
import { GetSheetRequestSchema } from "@/types/proto-es/v1/sheet_service_pb";
import { convertNewSheetToOld } from "@/utils/v1/sheet-conversions";

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
    const newResponse = await sheetServiceClientConnect.getSheet(request);
    const response = convertNewSheetToOld(newResponse);

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
