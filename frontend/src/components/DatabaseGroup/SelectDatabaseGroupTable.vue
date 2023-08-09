<template>
  <BBGrid
    :column-list="COLUMN_LIST"
    :data-source="databaseGroupList"
    row-key="name"
    class="border"
    :row-clickable="true"
    @click-row="handleDatabaseGroupSelect"
  >
    <template #item="{ item: dbGroup }: { item: ComposedDatabaseGroup }">
      <div class="bb-grid-cell justify-center items-center">
        <NRadio
          :checked="state.selectedDatabaseGroupName === dbGroup.name"
          :value="dbGroup.name"
          name="database-group"
          @update:checked="handleDatabaseGroupSelect(dbGroup)"
        >
        </NRadio>
      </div>
      <div class="bb-grid-cell">
        {{ dbGroup.databasePlaceholder }}
      </div>
      <div class="bb-grid-cell">{{ dbGroup.project.title }}</div>
      <div class="bb-grid-cell">{{ dbGroup.environment.title }}</div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { NRadio } from "naive-ui";
import { ref, watch, reactive, computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBGridColumn } from "@/bbkit";
import { useDBGroupStore } from "@/store";
import { ComposedDatabaseGroup } from "@/types";
import { SchemaGroup } from "@/types/proto/v1/project_service";

interface LocalState {
  selectedDatabaseGroupName?: string;
}

const props = defineProps<{
  databaseGroupList: ComposedDatabaseGroup[];
  selectedDatabaseGroupName?: string;
}>();

const emit = defineEmits<{
  (event: "update", dbGroupName: string): void;
}>();

const { t } = useI18n();
const dbGroupStore = useDBGroupStore();
const state = reactive<LocalState>({
  selectedDatabaseGroupName: props.selectedDatabaseGroupName,
});
const schemaGroupListMap = ref<Map<string, SchemaGroup[]>>(new Map());

const COLUMN_LIST = computed(() => {
  const columns: BBGridColumn[] = [
    {
      title: "",
      width: "2rem",
    },
    { title: t("common.name"), width: "1fr" },
    {
      title: t("common.project"),
      width: "1fr",
    },
    {
      title: t("common.environment"),
      width: "1fr",
    },
  ];

  return columns;
});

const handleDatabaseGroupSelect = (dbGroup: ComposedDatabaseGroup) => {
  if (state.selectedDatabaseGroupName === dbGroup.name) {
    return;
  }

  state.selectedDatabaseGroupName = dbGroup.name;
  emit("update", state.selectedDatabaseGroupName || "");
};

watch(
  () => props,
  async () => {
    for (const dbGroup of props.databaseGroupList) {
      const schemaGroupList =
        await dbGroupStore.getOrFetchSchemaGroupListByDBGroupName(dbGroup.name);
      schemaGroupListMap.value.set(dbGroup.name, schemaGroupList);
    }
  },
  {
    immediate: true,
    deep: true,
  }
);
</script>
