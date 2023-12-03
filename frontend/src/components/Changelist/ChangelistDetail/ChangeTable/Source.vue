<template>
  <div>
    <template v-if="type === 'CHANGE_HISTORY'">
      <div class="flex items-center gap-x-1">
        <History :size="16" />
        <span>{{ $t("common.change-history") }}</span>
        <span v-if="changeHistory" class="textinfolabel">
          {{ changeHistory.version }}
        </span>
      </div>
    </template>
    <template v-if="type === 'BRANCH'">
      <div class="flex items-center gap-x-1">
        <GitBranch :size="16" />
        <span>{{ $t("common.branch") }}</span>
        <span v-if="branch" class="textinfolabel">
          {{ branch.branchId }}
        </span>
      </div>
    </template>
    <template v-if="type === 'RAW_SQL'">
      <div class="flex items-center gap-x-1">
        <File :size="16" />
        <span>{{ $t("changelist.change-source.raw-sql") }}</span>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { asyncComputed } from "@vueuse/core";
import { File, GitBranch, History } from "lucide-vue-next";
import { computed } from "vue";
import { useChangeHistoryStore, useBranchStore } from "@/store";
import { Changelist_Change as Change } from "@/types/proto/v1/changelist_service";
import { getChangelistChangeSourceType } from "@/utils";

const props = defineProps<{
  change: Change;
}>();

const type = computed(() => {
  return getChangelistChangeSourceType(props.change);
});

const changeHistory = computed(() => {
  if (type.value !== "CHANGE_HISTORY") return undefined;
  return useChangeHistoryStore().getChangeHistoryByName(props.change.source);
});

const branch = asyncComputed(() => {
  if (type.value !== "BRANCH") return undefined;
  return useBranchStore().fetchBranchByName(
    props.change.source,
    true /* useCache */
  );
}, undefined);
</script>
