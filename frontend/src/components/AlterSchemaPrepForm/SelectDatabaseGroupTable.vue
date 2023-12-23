<template>
  <BBGrid
    :column-list="COLUMN_LIST"
    :data-source="databaseGroupList"
    row-key="name"
    class="border"
    :row-clickable="true"
    :show-placeholder="true"
    @click-row="handleDatabaseGroupSelect"
  >
    <template #item="{ item: dbGroup }: { item: ComposedDatabaseGroup }">
      <div
        v-if="showSelection"
        class="bb-grid-cell justify-center items-center"
      >
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
      <div class="bb-grid-cell">
        <ProjectCol :project="dbGroup.project" :show-tenant-icon="true" />
      </div>
      <div class="bb-grid-cell">
        <EnvironmentV1Name :environment="dbGroup.environment" :link="false" />
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { NRadio } from "naive-ui";
import { reactive, computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBGridColumn } from "@/bbkit";
import { ComposedDatabaseGroup } from "@/types";

interface LocalState {
  selectedDatabaseGroupName?: string;
}

const props = defineProps<{
  databaseGroupList: ComposedDatabaseGroup[];
  selectedDatabaseGroupName?: string;
  showSelection?: boolean;
}>();

const emit = defineEmits<{
  (event: "update", dbGroupName: string): void;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  selectedDatabaseGroupName: props.selectedDatabaseGroupName,
});

const COLUMN_LIST = computed(() => {
  const columns: BBGridColumn[] = [
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

  if (props.showSelection) {
    columns.unshift({
      title: "",
      width: "2rem",
    });
  }

  return columns;
});

const handleDatabaseGroupSelect = (dbGroup: ComposedDatabaseGroup) => {
  if (state.selectedDatabaseGroupName === dbGroup.name) {
    return;
  }

  state.selectedDatabaseGroupName = dbGroup.name;
  emit("update", state.selectedDatabaseGroupName || "");
};
</script>
