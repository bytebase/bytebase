<template>
  <BBTable
    :column-list="columnList"
    :data-source="guideList"
    :show-header="true"
    :left-bordered="false"
    :right-bordered="false"
    @click-row="onRowClick"
  >
    <template #body="{ rowData: guide }">
      <BBTableCell class="w-32">
        {{ guide.name }}
      </BBTableCell>
      <BBTableCell class="w-16">
        <div class="flex gap-x-3">
          <BBBadge
            v-for="envId in guide.environmentList"
            :key="envId"
            :text="environmentNameFromId(envId)"
            :can-remove="false"
          />
        </div>
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ humanizeTs(guide.updatedTs) }}
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ humanizeTs(guide.createdTs) }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts" setup>
import { PropType } from "vue";
import { useRouter } from "vue-router";
import { BBTableColumn } from "../../bbkit/types";
import { environmentName, schemaGuideSlug } from "../../utils";
import { EnvironmentId, DatabaseSchemaGuide } from "../../types";
import { useI18n } from "vue-i18n";
import { useEnvironmentStore } from "@/store";

const props = defineProps({
  guideList: {
    required: true,
    type: Object as PropType<DatabaseSchemaGuide[]>,
  },
});

const { t } = useI18n();
const router = useRouter();

const columnList: BBTableColumn[] = [
  {
    title: t("common.name"),
  },
  {
    title: t("common.environment"),
  },
  {
    title: t("common.updated-at"),
  },
  {
    title: t("common.created-at"),
  },
];

const onRowClick = function (section: number, row: number) {
  const guide = props.guideList[row];
  router.push({
    name: "setting.workspace.database-review-guide.detail",
    params: {
      schemaGuideSlug: schemaGuideSlug(guide),
    },
  });
};

const environmentNameFromId = function (id: EnvironmentId) {
  const env = useEnvironmentStore().getEnvironmentById(id);

  return environmentName(env);
};
</script>
