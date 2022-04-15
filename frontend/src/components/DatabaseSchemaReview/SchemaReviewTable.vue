<template>
  <BBTable
    :column-list="columnList"
    :data-source="reviewList"
    :show-header="true"
    :left-bordered="false"
    :right-bordered="false"
    @click-row="onRowClick"
  >
    <template #body="{ rowData: review }">
      <BBTableCell class="w-32">
        {{ review.name }}
      </BBTableCell>
      <BBTableCell class="w-16">
        <div class="flex gap-x-2">
          <BBBadge
            v-for="envId in review.environmentList"
            :key="envId"
            :text="envStore.getEnvironmentNameById(envId)"
            :can-remove="false"
          />
        </div>
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ humanizeTs(review.updatedTs) }}
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ humanizeTs(review.createdTs) }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts" setup>
import { PropType } from "vue";
import { useRouter } from "vue-router";
import { BBTableColumn } from "../../bbkit/types";
import { schemaReviewSlug } from "../../utils";
import { DatabaseSchemaReview } from "../../types";
import { useI18n } from "vue-i18n";
import { useEnvironmentStore } from "@/store";

const props = defineProps({
  reviewList: {
    required: true,
    type: Object as PropType<DatabaseSchemaReview[]>,
  },
});

const { t } = useI18n();
const router = useRouter();
const envStore = useEnvironmentStore();

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
  const review = props.reviewList[row];
  router.push({
    name: "setting.workspace.schame-review.detail",
    params: {
      schemaReviewSlug: schemaReviewSlug(review),
    },
  });
};
</script>
