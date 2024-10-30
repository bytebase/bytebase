<template>
  <div class="w-full">
    <div class="flex flex-row items-center gap-2">
      <NTag round>
        {{ releaseFile.version }}
      </NTag>
      <h2 class="text-xl font-bold text-main truncate">
        {{ releaseFile.path}}
      </h2>
    </div>
    <p class="mt-3 text-control text-base space-x-4">
      <span>{{ "Hash" }}: {{ releaseFile.sheetSha256.slice(0, 8) }}</span>
    </p>
  </div>

  <NDivider />

  <div class="flex flex-col gap-y-2">
    <a
      id="statement"
      href="#statement"
      class="w-auto flex items-center text-base text-main mb-2 hover:underline"
    >
      {{ $t("common.statement") }}
      <button
        tabindex="-1"
        class="btn-icon ml-1"
        @click.prevent="copyStatement"
      >
        <ClipboardIcon class="w-4 h-4" />
      </button>
    </a>
    <MonacoEditor
      class="h-auto max-h-[480px] min-h-[120px] border rounded-[3px] text-sm overflow-clip relative"
      :content="releaseFile.statement"
      :readonly="true"
      :auto-height="{ min: 120, max: 480 }"
    />
  </div>
</template>

<script lang="ts" setup>
import { ClipboardIcon } from "lucide-vue-next";
import { NDivider, NTag } from "naive-ui";
import { MonacoEditor } from "@/components/MonacoEditor";
import { pushNotification } from "@/store";
import { type ComposedRelease } from "@/types";
import type { Release_File } from "@/types/proto/v1/release_service";
import { toClipboard } from "@/utils";

const props = defineProps<{
  release: ComposedRelease;
  releaseFile: Release_File;
}>();

const copyStatement = async () => {
  toClipboard(props.releaseFile.statement).then(() => {
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: `Statement copied to clipboard.`,
    });
  });
};
</script>
