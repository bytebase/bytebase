<template>
  <BBModal
    :title="overrideTitle ?? $t('task.check-result.title-general')"
    :show-close="!confirm"
    :close-on-esc="!confirm"
    class="!w-[56rem]"
    header-class="whitespace-pre-wrap break-all gap-x-1"
    container-class="!pt-0 -mt-px"
    @close="$emit('close')"
  >
    <SQLCheckDetail :database="database" :advices="advices" />

    <div
      v-if="confirm"
      class="flex flex-row justify-end items-center gap-x-3 mt-4"
    >
      <NButton @click="confirm.resolve(false)">
        {{ $t("issue.sql-check.back-to-edit") }}
      </NButton>
      <NButton
        v-if="!restrictIssueCreationForSQLReview"
        type="primary"
        @click="confirm.resolve(true)"
      >
        {{ $t("issue.sql-check.continue-anyway") }}
      </NButton>
    </div>
  </BBModal>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { computed, onMounted } from "vue";
import { usePolicyV1Store } from "@/store";
import { ComposedDatabase } from "@/types";
import { Advice } from "@/types/proto/v1/sql_service";
import { Defer } from "@/utils";
import SQLCheckDetail from "./SQLCheckDetail.vue";

defineProps<{
  database: ComposedDatabase;
  advices: Advice[];
  overrideTitle?: string;
  confirm?: Defer<boolean>;
}>();

defineEmits<{
  (event: "close"): void;
}>();

const policyV1Store = usePolicyV1Store();

const restrictIssueCreationForSQLReview = computed((): boolean => {
  return (
    policyV1Store.getPolicyByName(
      "policies/RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW"
    )?.restrictIssueCreationForSqlReviewPolicy?.disallow ?? false
  );
});

onMounted(async () => {
  await prepareOrgPolicy();
});

const prepareOrgPolicy = async () => {
  await policyV1Store.getOrFetchPolicyByName(
    "policies/RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW"
  );
};
</script>
