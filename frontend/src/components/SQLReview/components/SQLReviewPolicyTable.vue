<template>
  <div>
    <div class="hidden md:grid">
      <SQLReviewPolicyDataTable
        :review-list="reviewList"
        :filter="filter"
        :allow-edit="true"
        @edit="handleClickEdit"
        @delete="handleClickDelete"
      />
    </div>

    <div
      class="flex flex-col md:hidden border px-2 pb-4 divide-y space-y-4 divide-block-border"
    >
      <div
        v-for="(review, i) in reviewList"
        :key="`${i}-${review.id}`"
        class="pt-4"
      >
        <div class="text-md">
          {{ review.name }}
        </div>
        <div class="space-y-2 space-x-2">
          <BBBadge
            v-for="resource in review.resources"
            :key="resource"
            :can-remove="false"
          >
            <Resource :show-prefix="true" :resource="resource" />
          </BBBadge>
          <BBBadge
            v-if="!review.enforce"
            :text="$t('common.disable')"
            :can-remove="false"
            :badge-style="'DISABLED'"
          />
        </div>
        <div class="flex items-center gap-x-2 mt-4">
          <NButton @click.prevent="handleClickEdit(review)">
            {{
              hasUpdatePolicyPermission ? $t("common.edit") : $t("common.view")
            }}
          </NButton>

          <BBButtonConfirm
            v-if="hasDeletePolicyPermission"
            :text="false"
            :disabled="!hasUpdatePolicyPermission"
            :type="'DELETE'"
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
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBBadge, BBButtonConfirm } from "@/bbkit";
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
