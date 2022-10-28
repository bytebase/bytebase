<template>
  <div class="mx-auto">
    <div class="textinfolabel">
      {{ $t("sql-review.description") }}
      <a
        href="https://www.bytebase.com/docs/sql-review/review-rules/overview"
        target="_blank"
        class="normal-link inline-flex flex-row items-center"
      >
        {{ $t("common.learn-more") }}
        <heroicons-outline:external-link class="w-4 h-4" />
      </a>
    </div>
    <FeatureAttention
      v-if="!hasSQLReviewPolicyFeature"
      custom-class="mt-5"
      feature="bb.feature.sql-review"
      :description="$t('subscription.features.bb-feature-sql-review.desc')"
    />
    <div v-if="store.reviewPolicyList.length > 0" class="space-y-6 my-5">
      <div v-if="hasPermission" class="flex items-center justify-end">
        <button
          type="button"
          class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
          @click.prevent="goToCreationView"
        >
          {{ $t("sql-review.create-policy") }}
        </button>
      </div>
      <template v-for="review in store.reviewPolicyList" :key="review.id">
        <SQLReviewCard :review-policy="review" @click="onClick" />
      </template>
    </div>
    <template v-else>
      <SQLReviewEmptyView @create="goToCreationView" />
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed, watchEffect } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import {
  pushNotification,
  useSQLReviewStore,
  useCurrentUser,
  featureToRef,
} from "@/store";
import { SQLReviewPolicy } from "@/types/sqlReview";
import { hasWorkspacePermission, sqlReviewPolicySlug } from "@/utils";

const router = useRouter();
const store = useSQLReviewStore();
const ROUTE_NAME = "setting.workspace.sql-review";
const currentUser = useCurrentUser();
const { t } = useI18n();

watchEffect(() => {
  store.fetchReviewPolicyList();
});

const hasPermission = computed(() => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-sql-review-policy",
    currentUser.value.role
  );
});

const hasSQLReviewPolicyFeature = featureToRef("bb.feature.sql-review");

const goToCreationView = () => {
  if (hasPermission.value) {
    router.push({
      name: `${ROUTE_NAME}.create`,
    });
  } else {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("sql-review.no-permission"),
    });
  }
};

const onClick = (review: SQLReviewPolicy) => {
  router.push({
    name: `${ROUTE_NAME}.detail`,
    params: {
      sqlReviewPolicySlug: sqlReviewPolicySlug(review),
    },
  });
};
</script>
