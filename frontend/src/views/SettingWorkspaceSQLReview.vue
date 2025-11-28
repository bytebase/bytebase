<template>
  <div class="mx-auto flex flex-col gap-y-4">
    <div class="textinfolabel">
      {{ $t("sql-review.description") }}
      <LearnMoreLink
        url="https://docs.bytebase.com/sql-review/review-rules?source=console"
      />
    </div>

    <div class="flex justify-end items-center gap-x-2">
      <SearchBox
        v-model:value="searchText"
        style="max-width: 100%"
        :autofocus="true"
        :placeholder="$t('common.filter-by-name')"
      />
      <NButton
        v-if="hasCreatePolicyPermission"
        type="primary"
        @click="createSQLReview"
      >
        <template #icon>
          <PlusIcon class="h-4 w-4" />
        </template>
        {{ $t("sql-review.create-policy") }}
      </NButton>
    </div>

    <SQLReviewPolicyTable
      v-if="sqlReviewStore.reviewPolicyList.length > 0"
      :review-list="filteredReviewConfigList"
      :filter="searchText"
    />
    <NEmpty v-else class="py-12 border rounded-sm">
      <template #extra>
        <NButton
          v-if="hasCreatePolicyPermission"
          :size="'small'"
          type="primary"
          @click="createSQLReview"
        >
          <template #icon>
            <PlusIcon class="h-4 w-4" />
          </template>
          {{ $t("sql-review.create-policy") }}
        </NButton>
      </template>
    </NEmpty>
  </div>
</template>

<script lang="ts" setup>
import { PlusIcon } from "lucide-vue-next";
import { NButton, NEmpty } from "naive-ui";
import { computed, ref, watchEffect } from "vue";
import { useRouter } from "vue-router";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import SQLReviewPolicyTable from "@/components/SQLReview/components/SQLReviewPolicyTable.vue";
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
  return hasWorkspacePermissionV2("bb.reviewConfigs.create");
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
