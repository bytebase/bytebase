<template>
  <div class="flex flex-row justify-start items-center opacity-80">
    <span class="mr-4 shrink-0"
      >{{ $t("schema-designer.baseline-version") }}:</span
    >
    <DatabaseInfo
      class="flex-nowrap mr-4 shrink-0"
      :database="baselineDatabase"
    />
    <div v-if="!isLoadingChangeHistory" class="shrink-0 flex-nowrap">
      <NTooltip v-if="changeHistory" trigger="hover">
        <template #trigger>@{{ changeHistory.version }}</template>
        <div class="w-full flex flex-row justify-start items-center">
          <span class="block pr-2 w-full max-w-[32rem] truncate">
            {{ changeHistory.version }} -
            {{ changeHistory.description }}
          </span>
          <span class="opacity-60">
            {{ humanizeDate(changeHistory.updateTime) }}
          </span>
        </div>
      </NTooltip>
      <div v-else>
        {{ "Previously latest schema" }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { asyncComputed } from "@vueuse/core";
import { computed, ref } from "vue";
import DatabaseInfo from "@/components/DatabaseInfo.vue";
import { useChangeHistoryStore, useDatabaseV1Store } from "@/store";
import { UNKNOWN_ID } from "@/types";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";

const props = defineProps<{
  branch: SchemaDesign;
}>();

const isLoadingChangeHistory = ref(false);

const baselineDatabase = computed(() => {
  return useDatabaseV1Store().getDatabaseByName(props.branch.baselineDatabase);
});

const changeHistoryName = computed(() => {
  const { branch } = props;
  if (!branch) return undefined;
  const id = branch.baselineChangeHistoryId;
  if (!id || id === String(UNKNOWN_ID)) return undefined;
  const name = `${baselineDatabase.value.name}/changeHistories/${id}`;
  return name;
});

const changeHistory = asyncComputed(
  async () => {
    const name = changeHistoryName.value;
    if (!name) return undefined;
    return await useChangeHistoryStore().getOrFetchChangeHistoryByName(name);
  },
  undefined,
  {
    evaluating: isLoadingChangeHistory,
  }
);
</script>
