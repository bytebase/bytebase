<template>
  <div class="flex flex-row items-center justify-between gap-x-2 group">
    <div class="flex flex-row items-center gap-x-2">
      <NTag>
        <span class="inline-block w-[30px] text-center">
          {{ getChangelogChangeType(changelog.type) }}
        </span>
      </NTag>
      <div
        class="flex flex-row items-center gap-x-1 border-b border-transparent group-hover:border-control-border cursor-pointer"
        @click="$emit('click-item', change)"
      >
        <RichDatabaseName
          :database="database"
          :show-instance="false"
          :show-arrow="false"
          :show-production-environment-icon="false"
          tooltip="instance"
        />
        <span>@</span>
        <span class="text-sm">{{ changelog.version }}</span>
        <router-link
          :to="{
            path: `/${changelog.issue}`,
          }"
          class="normal-link text-sm hover:!no-underline"
          target="_blank"
          @click.stop
        >
          #{{ extractIssueUID(changelog.issue) }}
        </router-link>
      </div>
    </div>
    <div
      class="flex flex-row items-center justify-end gap-x-1 invisible group-hover:visible"
    >
      <NButton
        size="small"
        quaternary
        style="--n-padding: 0 6px"
        @click.stop="$emit('remove-item', change)"
      >
        <template #icon>
          <heroicons:x-mark />
        </template>
      </NButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computedAsync } from "@vueuse/core";
import { NButton, NTag } from "naive-ui";
import { computed } from "vue";
import { RichDatabaseName } from "@/components/v2";
import { useChangelogStore, useDatabaseV1Store } from "@/store";
import { unknownDatabase } from "@/types";
import type { Changelist_Change as Change } from "@/types/proto/v1/changelist_service";
import { Changelog } from "@/types/proto/v1/database_service";
import { extractDatabaseResourceName, extractIssueUID } from "@/utils";
import { getChangelogChangeType } from "@/utils/v1/changelog";

const props = defineProps<{
  change: Change;
}>();

defineEmits<{
  (event: "click-item", change: Change): void;
  (event: "remove-item", change: Change): void;
}>();

const changelog = computed(() => {
  const name = props.change.source;
  return (
    useChangelogStore().getChangelogByName(name) ??
    Changelog.fromPartial({
      name,
      version: "<<Unknown Changelog>>",
    })
  );
});

const database = computedAsync(() => {
  const { database } = extractDatabaseResourceName(changelog.value.name);
  return useDatabaseV1Store().getOrFetchDatabaseByName(database);
}, unknownDatabase());
</script>
