<template>
  <BBGrid
    class="border"
    :show-placeholder="true"
    :column-list="COLUMN_LIST"
    :data-source="branches"
    :ready="ready"
    v-bind="$attrs"
    @click-row="clickBranch"
  >
    <template #item="{ item: branch }: { item: Branch }">
      <div v-if="!hideProjectColumn" class="bb-grid-cell">
        {{ projectV1Name(getFormattedValue(branch).project) }}
      </div>
      <div class="bb-grid-cell">
        <NPerformantEllipsis :line-clamp="1">{{
          branch.title
        }}</NPerformantEllipsis>
      </div>
      <div class="bb-grid-cell">
        <NPerformantEllipsis :line-clamp="1">{{
          getFormattedValue(branch).parentBranch
        }}</NPerformantEllipsis>
      </div>
      <div class="bb-grid-cell">
        <DatabaseInfo :database="getFormattedValue(branch).database" />
      </div>
      <div class="bb-grid-cell">
        <NPerformantEllipsis>
          <span class="text-gray-400">
            {{ getFormattedValue(branch).updatedTimeStr }}</span
          >
        </NPerformantEllipsis>
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { NPerformantEllipsis } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBGridColumn } from "@/bbkit";
import DatabaseInfo from "@/components/DatabaseInfo.vue";
import { useDatabaseV1Store, useProjectV1Store, useUserStore } from "@/store";
import {
  getProjectAndSchemaDesignSheetId,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import { Branch } from "@/types/proto/v1/branch_service";
import { projectV1Name } from "@/utils";

const emit = defineEmits<{
  (event: "click", branch: Branch): void;
}>();

const props = defineProps<{
  branches: Branch[];
  hideProjectColumn?: boolean;
  ready?: boolean;
}>();

const { t } = useI18n();
const userV1Store = useUserStore();
const projectV1Store = useProjectV1Store();
const databaseV1Store = useDatabaseV1Store();

const COLUMN_LIST = computed(() => {
  const columns: BBGridColumn[] = [
    { title: t("database.branch"), width: "minmax(auto, 0.5fr)" },
    { title: t("schema-designer.parent-branch"), width: "minmax(auto, 0.5fr)" },
    { title: t("common.database"), width: "1fr" },
    { title: "", width: "1fr" },
  ];
  if (!props.hideProjectColumn) {
    columns.unshift({
      title: t("common.project"),
      width: "minmax(auto, 0.5fr)",
    });
  }

  return columns;
});

const getFormattedValue = (branch: Branch) => {
  const [projectName] = getProjectAndSchemaDesignSheetId(branch.name);
  const project = projectV1Store.getProjectByName(
    `${projectNamePrefix}${projectName}`
  );
  let parentBranch = "";
  if (branch.parentBranch !== "") {
    const parentSchemaDesign = props.branches.find(
      (br) => br.name === branch.parentBranch
    );
    if (parentSchemaDesign) {
      parentBranch = parentSchemaDesign.title;
    }
  }

  const updater = userV1Store.getUserByEmail(branch.updater.split("/")[1]);
  const updatedTimeStr = t("schema-designer.message.updated-time-by-user", {
    time: dayjs
      .duration((branch.updateTime ?? new Date()).getTime() - Date.now())
      .humanize(true),
    user: updater?.title,
  });

  return {
    name: branch.title,
    project: project,
    database: databaseV1Store.getDatabaseByName(branch.baselineDatabase),
    parentBranch: parentBranch,
    updatedTimeStr,
  };
};

const clickBranch = (branch: Branch) => {
  emit("click", branch);
};
</script>
