<template>
  <div class="w-full">
    <div class="flex flex-row items-center gap-2">
      <p class="text-lg space-x-1">
        <span class="text-control">{{ $t("common.version") }}:</span>
        <span class="font-bold text-main">{{ releaseFile.version }}</span>
      </p>
    </div>
    <p class="mt-3 text-control text-sm space-x-4">
      <span v-if="releaseFile.path"
        >{{ $t("database.revision.filename") }}: {{ releaseFile.path }}</span
      >
      <span>{{ "Hash" }}: {{ releaseFile.sheetSha256.slice(0, 8) }}</span>
    </p>
  </div>

  <NDivider />

  <div class="flex flex-col gap-y-2">
    <p class="w-auto flex items-center text-base text-main mb-2">
      {{ $t("common.statement") }}
      <button
        tabindex="-1"
        class="btn-icon ml-1"
        @click.prevent="copyStatement"
      >
        <ClipboardIcon class="w-4 h-4" />
      </button>
    </p>
    <MonacoEditor
      class="h-auto max-h-[480px] min-h-[120px] border rounded-[3px] text-sm overflow-clip relative"
      :content="statement"
      :readonly="true"
      :auto-height="{ min: 120, max: 480 }"
    />
  </div>
</template>

<script lang="ts" setup>
import { ClipboardIcon } from "lucide-vue-next";
import { NDivider } from "naive-ui";
import { computed } from "vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import { pushNotification } from "@/store";
import { type ComposedRelease } from "@/types";
import type { Release_File } from "@/types/proto/v1/release_service";
import { getReleaseFileStatement, toClipboard } from "@/utils";

const props = defineProps<{
  release: ComposedRelease;
  releaseFile: Release_File;
}>();

const statement = computed(() => getReleaseFileStatement(props.releaseFile));

const copyStatement = async () => {
  toClipboard(statement.value).then(() => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: `Statement copied to clipboard.`,
    });
  });
};
</script>
