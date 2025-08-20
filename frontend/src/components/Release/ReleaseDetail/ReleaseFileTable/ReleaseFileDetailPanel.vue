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
    <p class="w-auto flex items-center text-base text-main mb-2 gap-x-2">
      {{ $t("common.statement") }}
      <CopyButton :content="statement" />
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
import { NDivider } from "naive-ui";
import { computed } from "vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import { CopyButton } from "@/components/v2";
import type {
  Release,
  Release_File,
} from "@/types/proto-es/v1/release_service_pb";
import { getReleaseFileStatement } from "@/utils";

const props = defineProps<{
  release: Release;
  releaseFile: Release_File;
}>();

const statement = computed(() => getReleaseFileStatement(props.releaseFile));
</script>
