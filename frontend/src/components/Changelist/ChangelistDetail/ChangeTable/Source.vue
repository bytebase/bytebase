<template>
  <div>
    <template v-if="type === 'CHANGELOG'">
      <div class="flex items-center gap-x-1">
        <HistoryIcon :size="16" />
        <span>{{ $t("common.change-history") }}</span>
        <span v-if="changelog" class="textinfolabel">
          {{ changelog.version }}
        </span>
        <router-link
          v-if="changelog"
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
    <template v-if="type === 'RAW_SQL'">
      <div class="flex items-center gap-x-1">
        <FileIcon :size="16" />
        <span>{{ $t("changelist.change-source.raw-sql") }}</span>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { FileIcon, HistoryIcon } from "lucide-vue-next";
import { computed } from "vue";
import { useChangelogStore } from "@/store";
import type { Changelist_Change as Change } from "@/types/proto/v1/changelist_service";
import { extractIssueUID, getChangelistChangeSourceType } from "@/utils";

const props = defineProps<{
  change: Change;
}>();

const type = computed(() => {
  return getChangelistChangeSourceType(props.change);
});

const changelog = computed(() => {
  if (type.value !== "CHANGELOG") return undefined;
  return useChangelogStore().getChangelogByName(props.change.source);
});
</script>
