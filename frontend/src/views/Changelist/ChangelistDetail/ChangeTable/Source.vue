<template>
  <div>
    <template v-if="type === 'CHANGE_HISTORY'">
      <div class="flex items-center">
        <History :size="16" class="mr-1" />
        {{ $t("common.change-history") }}
      </div>
    </template>
    <template v-if="type === 'BRANCH'">
      <div class="flex items-center">
        <GitBranch :size="16" class="mr-1" />{{ $t("common.branch") }}
      </div>
    </template>
    <template v-if="type === 'RAW_SQL'">
      <div class="flex items-center">
        <File :size="16" class="mr-1" />
        {{ $t("changelist.change-source.raw-sql") }}
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { File, GitBranch, History } from "lucide-vue-next";
import { computed } from "vue";
import { Changelist_Change as Change } from "@/types/proto/v1/changelist_service";
import { getChangelistChangeSourceType } from "@/utils";

const props = defineProps<{
  change: Change;
}>();

const type = computed(() => {
  return getChangelistChangeSourceType(props.change);
});
</script>
