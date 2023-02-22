<template>
  <div class="mx-auto">
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
    <FeatureAttention
      v-if="!hasSQLReviewPolicyFeature"
      custom-class="mt-5"
      feature="bb.feature.sql-review"
      :description="$t('subscription.features.bb-feature-sql-review.desc')"
    />
    <SQLReviewPolicyTable class="my-5" />
  </div>
</template>

<script lang="ts" setup>
import { watchEffect } from "vue";
import { useSQLReviewStore, featureToRef } from "@/store";
import SQLReviewPolicyTable from "@/components/SQLReview/SQLReviewPolicyTable.vue";

const store = useSQLReviewStore();

watchEffect(() => {
  store.fetchReviewPolicyList();
});

const hasSQLReviewPolicyFeature = featureToRef("bb.feature.sql-review");
</script>
