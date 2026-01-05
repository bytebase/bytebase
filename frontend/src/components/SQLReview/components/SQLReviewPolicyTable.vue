<template>
  <div>
    <div class="hidden lg:grid">
      <SQLReviewPolicyDataTable
        :review-list="reviewList"
        :filter="filter"
        :allow-edit="true"
        @edit="handleClickEdit"
        @delete="handleClickDelete"
      />
    </div>

    <div
      class="flex flex-col lg:hidden border px-2 divide-y divide-block-border"
    >
      <div
        v-for="(review, i) in reviewList"
        :key="`${i}-${review.id}`"
        class="py-4"
      >
        <div class="text-md">
          {{ review.name }}
        </div>
        <div class="flex flex-wrap mt-2 gap-2">
          <NTag
            v-for="resource in review.resources"
            size="small"
            type="primary"
            :key="resource"
          >
            <Resource :show-prefix="true" :resource="resource" />
          </NTag>
          <NTag
            v-if="!review.enforce"
            type="warning"
          >
            {{ $t('common.disable') }}
          </NTag>
        </div>
        <div class="flex items-center gap-x-2 mt-4">
          <NButton size="small" @click.prevent="handleClickEdit(review)">
            {{
              hasUpdatePolicyPermission ? $t("common.edit") : $t("common.view")
            }}
          </NButton>

          <BBButtonConfirm
            v-if="hasDeletePolicyPermission"
            :text="false"
            :disabled="!hasUpdatePolicyPermission"
            :type="'DELETE'"
            size="small"
            :hide-icon="true"
            :button-text="$t('common.delete')"
            :ok-text="$t('common.delete')"
            :confirm-title="$t('common.delete') + ` '${review.name}'?`"
            :require-confirm="true"
            @confirm="handleClickDelete(review)"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="tsx">
import { NButton, NTag } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBButtonConfirm } from "@/bbkit";
import Resource from "@/components/v2/ResourceOccupiedModal/Resource.vue";
import { WORKSPACE_ROUTE_SQL_REVIEW_DETAIL } from "@/router/dashboard/workspaceRoutes";
import { pushNotification, useSQLReviewStore } from "@/store";
import type { SQLReviewPolicy } from "@/types";
import { hasWorkspacePermissionV2, sqlReviewPolicySlug } from "@/utils";
import SQLReviewPolicyDataTable from "./SQLReviewPolicyDataTable.vue";

defineProps<{
  reviewList: SQLReviewPolicy[];
  filter: string;
}>();

const { t } = useI18n();
const router = useRouter();
const sqlReviewStore = useSQLReviewStore();

const hasUpdatePolicyPermission = computed(() => {
  return hasWorkspacePermissionV2("bb.reviewConfigs.update");
});

const hasDeletePolicyPermission = computed(() => {
  return hasWorkspacePermissionV2("bb.reviewConfigs.delete");
});

const handleClickEdit = (review: SQLReviewPolicy) => {
  router.push({
    name: WORKSPACE_ROUTE_SQL_REVIEW_DETAIL,
    params: {
      sqlReviewPolicySlug: sqlReviewPolicySlug(review),
    },
  });
};

const handleClickDelete = async (review: SQLReviewPolicy) => {
  await sqlReviewStore.removeReviewPolicy(review.id);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("sql-review.policy-removed"),
  });
};
</script>
