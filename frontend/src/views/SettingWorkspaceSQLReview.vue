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

    <SQLReviewPolicyTable />
  </div>
</template>

<script lang="ts" setup>
import { watchEffect } from "vue";
import SQLReviewPolicyTable from "@/components/SQLReview/SQLReviewPolicyTable.vue";
import { useSQLReviewStore } from "@/store";

const store = useSQLReviewStore();

watchEffect(() => {
  store.fetchReviewPolicyList();
});
</script>
