<template>
  <div>
    <template v-if="type === 'CHANGE_HISTORY'">
      {{ $t("common.change-history") }}
    </template>
    <template v-if="type === 'BRANCH'">
      {{ $t("common.branch") }}
    </template>
    <template v-if="type === 'RAW_SQL'">
      {{ $t("changelist.change-source.raw-sql") }}
    </template>
  </div>
</template>

<script setup lang="ts">
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
