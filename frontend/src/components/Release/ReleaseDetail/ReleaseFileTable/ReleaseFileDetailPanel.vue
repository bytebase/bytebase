<template>
  <div class="w-full border-b pb-4 mb-4">
    <h1 class="text-xl font-bold text-main truncate">
      {{ releaseFile.name }}
    </h1>
    <p class="mt-2 text-control text-base space-x-4">
      <span>{{ $t("common.version") }}: {{ releaseFile.version }}</span>
      <span>{{ "Hash" }}: {{ releaseFile.sheetSha256.slice(0, 8) }}</span>
    </p>
  </div>

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
