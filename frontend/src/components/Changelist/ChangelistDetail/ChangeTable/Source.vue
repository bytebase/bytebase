<template>
  <div>
    <template v-if="type === 'CHANGELOG' && changelog">
      <div class="flex items-center gap-x-1">
        <HistoryIcon :size="16" />
        <span>{{ $t("common.changelog") }}</span>
        <span v-if="changelog.version" class="textinfolabel">
          {{ changelog.version }}
        </span>
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
    </template>
    <template v-else-if="type === 'RAW_SQL'">
      <div class="flex items-center gap-x-1">
        <FileIcon :size="16" />
        <span>{{ $t("changelist.change-source.raw-sql") }}</span>
      </div>
    </template>
    <!-- Fallback -->
    <template v-else>
      <div class="flex items-center gap-x-1">
        <FileIcon :size="16" />
        <span>Unknown change type</span>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computedAsync } from "@vueuse/core";
import { FileIcon, HistoryIcon } from "lucide-vue-next";
import { computed } from "vue";
import { useChangelogStore } from "@/store";
import type { Changelist_Change as Change } from "@/types/proto-es/v1/changelist_service_pb";
import { extractIssueUID, getChangelistChangeSourceType } from "@/utils";

const props = defineProps<{
  change: Change;
}>();

const type = computed(() => {
  return getChangelistChangeSourceType(props.change);
});

const changelog = computedAsync(async () => {
  if (type.value !== "CHANGELOG") return undefined;
  return await useChangelogStore().getOrFetchChangelogByName(
    props.change.source
  );
});
</script>
