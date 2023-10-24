<template>
  <ErrorTipsButton
    icon
    style="--n-padding: 0 10px"
    :errors="errors"
    :button-props="{
      loading: isExporting,
    }"
    :disabled="isExporting"
    @click="handleExport"
  >
    <template #icon>
      <heroicons:arrow-down-tray />
    </template>
  </ErrorTipsButton>
</template>

<script setup lang="ts">
import dayjs from "dayjs";
import saveAs from "file-saver";
import JSZip from "jszip";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { ErrorTipsButton } from "@/components/v2";
import { escapeFilename } from "@/utils";
import { useChangelistDetailContext } from "../context";
import { zipFileForChange } from "./export";

const { t } = useI18n();
const { changelist } = useChangelistDetailContext();
const isExporting = ref(false);

const errors = computed(() => {
  const errors: string[] = [];
  if (changelist.value.changes.length === 0) {
    errors.push(t("changelist.error.select-at-least-one-change-to-export"));
  }
  return errors;
});

const handleExport = async () => {
  if (isExporting.value) {
    return;
  }

  isExporting.value = true;
  const zip = new JSZip();
  const { changes } = changelist.value;
  for (let i = 0; i < changes.length; i++) {
    await zipFileForChange(zip, changes[i], i);
  }

  try {
    const content = await zip.generateAsync({ type: "blob" });
    const basename = `${changelist.value.description}_${dayjs().format(
      "YYYYMMDD"
    )}`;
    const fileName = `${escapeFilename(basename)}.zip`;
    saveAs(content, fileName);
  } catch (error) {
    console.error(error);
  }

  isExporting.value = false;
};
</script>
