<template>
  <div class="w-full" v-bind="$attrs">
    <div
      class="w-full mb-4 flex flex-row justify-between items-center space-x-2"
    >
      <span>
        {{ $t("schema-designer.select-design") }}
      </span>
      <div>
        <NButton @click="state.showCreatePanel = true">
          <heroicons-solid:plus class="w-4 h-auto mr-0.5" />
          <span>{{ $t("schema-designer.new-design") }}</span>
        </NButton>
      </div>
    </div>
    <BBGrid
      class="border"
      :show-placeholder="true"
      :column-list="COLUMN_LIST"
      :data-source="schemaDesignList"
      @click-row="clickSchemaDesign"
    >
      <template #item="{ item: schemaDesign }: { item: SchemaDesign }">
        <div class="bb-grid-cell">
          <NRadio :checked="schemaDesign.name === selectedSchemaDesign?.name" />
        </div>
        <div class="bb-grid-cell">
          {{ schemaDesign.title }}
        </div>
        <div class="bb-grid-cell">
          {{ getFormatedValue(schemaDesign).project }}
        </div>
        <div class="bb-grid-cell">
          {{ engineNameV1(getFormatedValue(schemaDesign).engine) }}
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
        <div class="bb-grid-cell">
          <NButton size="small" @click="state.showEditPanel = true">
            {{ $t("common.view") }}
          </NButton>
        </div>
      </template>
    </BBGrid>
  </div>

  <CreateSchemaDesignPanel
    v-if="state.showCreatePanel"
    @dismiss="state.showCreatePanel = false"
    @created="
      (schemaDesign) => {
        state.showCreatePanel = false;
        clickSchemaDesign(schemaDesign);
      }
    "
  />

  <EditSchemaDesignPanel
    v-if="state.showEditPanel && selectedSchemaDesign"
    :schema-design-name="selectedSchemaDesign.name"
    @dismiss="state.showEditPanel = false"
  />
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { computed, ref, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { NButton } from "naive-ui";
import { BBGridColumn } from "@/bbkit";
import { getProjectAndSchemaDesignSheetId } from "@/store/modules/v1/common";
import { useProjectV1Store, useUserStore } from "@/store";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";
import { engineNameV1, projectV1Name } from "@/utils";
import { useSchemaDesignList } from "@/store/modules/schemaDesign";
import { NRadio } from "naive-ui";
import CreateSchemaDesignPanel from "@/components/SchemaDesigner/CreateSchemaDesignPanel.vue";
import EditSchemaDesignPanel from "@/components/SchemaDesigner/EditSchemaDesignPanel.vue";

interface LocalState {
  showCreatePanel: boolean;
  showEditPanel: boolean;
}

const emit = defineEmits<{
  (event: "select", schemaDesign: SchemaDesign): void;
}>();

const props = defineProps<{
  selectedSchemaDesign?: SchemaDesign;
}>();

const { t } = useI18n();
const projectV1Store = useProjectV1Store();
const { schemaDesignList } = useSchemaDesignList();
const state = reactive<LocalState>({
  showCreatePanel: false,
  showEditPanel: false,
});
const selectedSchemaDesign = ref<SchemaDesign | undefined>(
  props.selectedSchemaDesign
);

const COLUMN_LIST = computed(() => {
  const columns: BBGridColumn[] = [
    { title: "", width: "3rem" },
    { title: t("common.name"), width: "1fr" },
    {
      title: t("common.project"),
      width: "1fr",
    },
    {
      title: t("database.engine"),
      width: "1fr",
    },
    { title: t("common.creator"), width: "1fr" },
    { title: t("common.updater"), width: "1fr" },
    { title: t("common.updated-at"), width: "1fr" },
    { title: "", width: "5rem" },
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
  selectedSchemaDesign.value = schemaDesign;
  emit("select", schemaDesign);
};
</script>
