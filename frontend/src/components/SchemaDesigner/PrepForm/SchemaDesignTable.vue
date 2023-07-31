<template>
  <div ref="wrapper" rule="database-table" v-bind="$attrs">
    <BBGrid
      class="border"
      :show-placeholder="true"
      :column-list="COLUMN_LIST"
      :data-source="schemaDesigns"
      @click-row="clickSchemaDesign"
    >
      <template #item="{ item: schemaDesign }: { item: SchemaDesign }">
        <div class="bb-grid-cell">
          <div class="flex flex-row justify-start items-center">
            <EngineIcon
              class="mr-2"
              :engine="getFormatedValue(schemaDesign).engine"
            />
            <span>{{ schemaDesign.title }}</span>
          </div>
        </div>
        <div class="bb-grid-cell">
          {{ getFormatedValue(schemaDesign).project }}
        </div>
        <div class="bb-grid-cell">
          {{ getFormatedValue(schemaDesign).creator }}
        </div>
        <div class="bb-grid-cell">
          {{ getFormatedValue(schemaDesign).updater }}
        </div>
        <div class="bb-grid-cell">
          <HumanizeTs
            :ts="(schemaDesign.updateTime?.getTime() ?? 0) / 1000"
            class="ml-1"
          />
        </div>
      </template>
    </BBGrid>
  </div>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBGridColumn } from "@/bbkit";
import { getProjectAndSchemaDesignSheetId } from "@/store/modules/v1/common";
import { useProjectV1Store, useUserStore } from "@/store";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";
import { projectV1Name } from "@/utils";

const emit = defineEmits<{
  (event: "click", schemaDesign: SchemaDesign): void;
}>();

defineProps<{
  schemaDesigns: SchemaDesign[];
}>();

const { t } = useI18n();
const projectV1Store = useProjectV1Store();

const COLUMN_LIST = computed(() => {
  const columns: BBGridColumn[] = [
    { title: t("common.name"), width: "1fr" },
    {
      title: t("common.project"),
      width: "1fr",
    },
    { title: t("common.creator"), width: "minmax(auto, 10rem)" },
    { title: t("common.updater"), width: "minmax(auto, 10rem)" },
    { title: t("common.updated-at"), width: "minmax(auto, 10rem)" },
  ];

  return columns;
});

const getFormatedValue = (schemaDesign: SchemaDesign) => {
  const [projectName] = getProjectAndSchemaDesignSheetId(schemaDesign.name);
  const project = projectV1Store.getProjectByName(`projects/${projectName}`);

  return {
    name: schemaDesign.title,
    project: projectV1Name(project),
    engine: schemaDesign.engine,
    creator:
      useUserStore().getUserByIdentifier(schemaDesign.creator)?.title ??
      schemaDesign.creator,
    updater:
      useUserStore().getUserByIdentifier(schemaDesign.updater)?.title ??
      schemaDesign.updater,
    updated: dayjs
      .duration((schemaDesign.updateTime ?? new Date()).getTime() - Date.now())
      .humanize(true),
  };
};

const clickSchemaDesign = (schemaDesign: SchemaDesign) => {
  emit("click", schemaDesign);
};
</script>
