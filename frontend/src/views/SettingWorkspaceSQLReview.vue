<template>
  <div class="mx-auto space-y-4">
    <div class="textinfolabel">
      {{ $t("sql-review.description") }}
      <a
        href="https://www.bytebase.com/docs/sql-review/review-rules"
        target="_blank"
        class="normal-link inline-flex flex-row items-center"
      >
        {{ $t("common.learn-more") }}
        <heroicons-outline:external-link class="w-4 h-4" />
      </a>
    </div>

    <div class="flex justify-end items-center space-x-2">
      <SearchBox
        v-model:value="searchText"
        :autofocus="true"
        :placeholder="$t('sql-review.search-by-name')"
      />
      <NButton
        v-if="hasCreatePolicyPermission"
        type="primary"
        @click="createSQLReview"
      >
        {{ $t("common.add") }}
      </NButton>
    </div>

    <SQLReviewPolicyTable
      v-if="sqlReviewStore.reviewPolicyList.length > 0"
      :review-list="filteredReviewConfigList"
      :filter="searchText"
    />
    <NoDataPlaceholder v-else>
      <template #default>
        <NButton
          :size="'small'"
          type="primary"
          :disabled="!hasCreatePolicyPermission"
          @click="createSQLReview"
        >
          {{ $t("sql-review.create-policy") }}
        </NButton>
      </template>
    </NoDataPlaceholder>
  </div>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { watchEffect, ref, computed } from "vue";
import { useRouter } from "vue-router";
import SQLReviewPolicyTable from "@/components/SQLReview/components/SQLReviewPolicyTable.vue";
import NoDataPlaceholder from "@/components/misc/NoDataPlaceholder.vue";
import { SearchBox } from "@/components/v2";
import { WORKSPACE_ROUTE_SQL_REVIEW_CREATE } from "@/router/dashboard/workspaceRoutes";
import { useSQLReviewStore } from "@/store";
import { hasWorkspacePermissionV2 } from "@/utils";

const router = useRouter();
const sqlReviewStore = useSQLReviewStore();
const searchText = ref("");

watchEffect(() => {
  sqlReviewStore.fetchReviewPolicyList();
});

const hasCreatePolicyPermission = computed(() => {
  return hasWorkspacePermissionV2("bb.policies.create");
});

const createSQLReview = () => {
  router.push({
    name: WORKSPACE_ROUTE_SQL_REVIEW_CREATE,
  });
};

const filteredReviewConfigList = computed(() => {
  if (!searchText.value) {
    return sqlReviewStore.reviewPolicyList;
  }
  return sqlReviewStore.reviewPolicyList.filter((config) => {
    return config.name.toLowerCase().includes(searchText.value.toLowerCase());
  });
});
</script>
