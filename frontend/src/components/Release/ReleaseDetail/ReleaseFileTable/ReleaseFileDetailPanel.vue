<template>
  <div class="w-full">
    <div class="flex flex-row items-center gap-2">
      <p class="text-lg flex gap-x-1">
        <span class="text-control">{{ $t("common.version") }}:</span>
        <span class="font-bold text-main">{{ releaseFile.version }}</span>
      </p>
    </div>
    <p class="mt-3 text-control text-sm flex gap-x-4">
      <span v-if="releaseFile.path"
        >{{ $t("database.revision.filename") }}: {{ releaseFile.path }}</span
      >
      <span>{{ "Hash" }}: {{ releaseFile.sheetSha256.slice(0, 8) }}</span>
    </p>
  </div>

  <NDivider />

  <div class="flex flex-col gap-y-2">
    <p class="w-auto flex items-center text-base text-main mb-2 gap-x-2">
      {{ $t("common.statement") }}
      <CopyButton :content="fetchedStatement" />
    </p>
    <NSpin :show="fetching">
      <MonacoEditor
        class="h-auto max-h-[480px] min-h-[120px] border rounded-[3px] text-sm overflow-clip relative"
        :content="fetchedStatement"
        :readonly="true"
        :auto-height="{ min: 120, max: 480 }"
      />
    </NSpin>
  </div>
</template>

<script lang="ts" setup>
import { NDivider, NSpin } from "naive-ui";
import { onMounted, ref, watch } from "vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import { CopyButton } from "@/components/v2";
import { sheetServiceClientConnect } from "@/connect";
import type {
  Release,
  Release_File,
} from "@/types/proto-es/v1/release_service_pb";

const props = defineProps<{
  release: Release;
  releaseFile: Release_File;
}>();

const fetchedStatement = ref("");
const fetching = ref(false);

const fetchStatement = async () => {
  fetching.value = true;
  try {
    const sheet = await sheetServiceClientConnect.getSheet({
      name: props.releaseFile.sheet,
      raw: true,
    });
    if (sheet?.content) {
      fetchedStatement.value = new TextDecoder().decode(sheet.content);
    }
  } catch (error) {
    console.error("Failed to fetch statement", error);
  } finally {
    fetching.value = false;
  }
};

onMounted(fetchStatement);

watch(
  () => props.releaseFile,
  () => {
    fetchStatement();
  }
);
</script>
