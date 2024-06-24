<template>
  <div class="mx-auto space-y-4">
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

    <div class="space-y-6">
      <SQLReviewPolicyTable
        :review-list="reviewConfigWithResourceList"
        :filter="searchText"
      />
      <div v-if="reviewConfigWithoutResourceList.length > 0">
        <h2 class="mb-3">{{ $t("sql-review.no-linked-resources") }}</h2>
        <SQLReviewPolicyTable
          :review-list="reviewConfigWithoutResourceList"
          :filter="searchText"
        />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { watchEffect, ref, computed } from "vue";
import { useRouter } from "vue-router";
import { WORKSPACE_ROUTE_SQL_REVIEW_CREATE } from "@/router/dashboard/workspaceRoutes";
import {
  useSQLReviewStore,
  useCurrentUserV1,
  useEnvironmentV1List,
} from "@/store";
import { environmentNamePrefix } from "@/store/modules/v1/common";
import type { SQLReviewPolicy } from "@/types";
import { hasWorkspacePermissionV2 } from "@/utils";

const router = useRouter();
const sqlReviewStore = useSQLReviewStore();
const searchText = ref("");
const currentUserV1 = useCurrentUserV1();
const environmentList = useEnvironmentV1List();

watchEffect(() => {
  sqlReviewStore.fetchReviewPolicyList();
});

const hasCreatePolicyPermission = computed(() => {
  return hasWorkspacePermissionV2(currentUserV1.value, "bb.policies.create");
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

const reviewConfigWithResourceList = computed(() => {
  const reviewConfigWithEnvironmentList = environmentList.value
    .map<SQLReviewPolicy>((environment) => {
      const review = sqlReviewStore.getReviewPolicyByResouce(environment.name);

      return (
        review ?? {
          id: "",
          enforce: false,
          name: "",
          ruleList: [],
          resources: [environment.name],
        }
      );
    })
    .filter((review) => {
      if (!searchText.value) {
        return true;
      }
      return review.name.toLowerCase().includes(searchText.value.toLowerCase());
    });

  return [
    ...reviewConfigWithEnvironmentList,
    ...filteredReviewConfigList.value.filter((review) => {
      return (
        review.resources.length > 0 &&
        !review.resources.some((resource) =>
          resource.startsWith(environmentNamePrefix)
        )
      );
    }),
  ];
});

const reviewConfigWithoutResourceList = computed(() => {
  return filteredReviewConfigList.value.filter((review) => {
    return review.resources.length === 0;
  });
});
</script>
