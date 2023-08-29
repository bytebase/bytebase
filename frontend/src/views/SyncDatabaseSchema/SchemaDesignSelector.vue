<template>
  <div class="w-full" v-bind="$attrs">
    <div
      class="w-full mb-4 flex flex-row justify-between items-center space-x-2"
    >
      <span>
        {{ $t("database.select-branch") }}
      </span>
      <div>
        <NButton @click="state.showCreatePanel = true">
          <heroicons-solid:plus class="w-4 h-auto mr-0.5" />
          <span>{{ $t("database.new-branch") }}</span>
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
          {{ projectV1Name(getFormatedValue(schemaDesign).project) }}
        </div>
        <div class="bb-grid-cell">
          {{ schemaDesign.title }}
        </div>
        <div class="bb-grid-cell">
          {{ getFormatedValue(schemaDesign).parentBranch }}
        </div>
        <div class="bb-grid-cell">
          <DatabaseInfo :database="getFormatedValue(schemaDesign).database" />
        </div>
        <div class="bb-grid-cell">
          <span class="text-gray-400">{{
            getFormatedValue(schemaDesign).updatedTimeStr
          }}</span>
        </div>

        <div class="bb-grid-cell">
          <NButton
            size="small"
            @click.stop="handleViewSchemaDesign(schemaDesign)"
          >
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
    :view-mode="true"
    @dismiss="state.showEditPanel = false"
  />
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { NButton } from "naive-ui";
import { NRadio } from "naive-ui";
import { computed, ref, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { BBGridColumn } from "@/bbkit";
import DatabaseInfo from "@/components/DatabaseInfo.vue";
import CreateSchemaDesignPanel from "@/components/SchemaDesigner/CreateSchemaDesignPanel.vue";
import EditSchemaDesignPanel from "@/components/SchemaDesigner/EditSchemaDesignPanel.vue";
import { useDatabaseV1Store, useProjectV1Store, useUserStore } from "@/store";
import {
  useSchemaDesignList,
  useSchemaDesignStore,
} from "@/store/modules/schemaDesign";
import { getProjectAndSchemaDesignSheetId } from "@/store/modules/v1/common";
import {
  SchemaDesign,
  SchemaDesign_Type,
} from "@/types/proto/v1/schema_design_service";
import { projectV1Name } from "@/utils";

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
const userV1Store = useUserStore();
const projectV1Store = useProjectV1Store();
const databaseV1Store = useDatabaseV1Store();
const schemaDesignStore = useSchemaDesignStore();
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
    {
      title: t("common.project"),
      width: "minmax(auto, 0.5fr)",
    },
    { title: t("database.branch"), width: "minmax(auto, 0.5fr)" },
    { title: t("schema-designer.parent-branch"), width: "minmax(auto, 0.5fr)" },
    { title: t("common.database"), width: "1fr" },
    { title: "", width: "1fr" },
    { title: "", width: "5rem" },
  ];

  return columns;
});

const getFormatedValue = (schemaDesign: SchemaDesign) => {
  const [projectName] = getProjectAndSchemaDesignSheetId(schemaDesign.name);
  const project = projectV1Store.getProjectByName(`projects/${projectName}`);
  let parentBranch = "";
  if (schemaDesign.type === SchemaDesign_Type.PERSONAL_DRAFT) {
    const parentSchemaDesign = schemaDesignStore.getSchemaDesignByName(
      schemaDesign.baselineSheetName
    );
    if (parentSchemaDesign) {
      parentBranch = parentSchemaDesign.title;
    }
  }

  const updater = userV1Store.getUserByEmail(
    schemaDesign.updater.split("/")[1]
  );
  const updatedTimeStr = t("schema-designer.message.updated-time-by-user", {
    time: dayjs
      .duration((schemaDesign.updateTime ?? new Date()).getTime() - Date.now())
      .humanize(true),
    user: updater?.title,
  });

  return {
    name: schemaDesign.title,
    project: project,
    database: databaseV1Store.getDatabaseByName(schemaDesign.baselineDatabase),
    parentBranch: parentBranch,
    updatedTimeStr,
  };
};

const clickSchemaDesign = (schemaDesign: SchemaDesign) => {
  selectedSchemaDesign.value = schemaDesign;
  emit("select", schemaDesign);
};

const handleViewSchemaDesign = (schemaDesign: SchemaDesign) => {
  clickSchemaDesign(schemaDesign);
  state.showEditPanel = true;
};
</script>
