<template>
  <div>
    <NTooltip placement="top-start" :disabled="allowEdit">
      <template #trigger>
        <label
          class="flex items-center gap-x-2"
          :class="[allowEdit ? 'cursor-pointer' : 'cursor-not-allowed']"
        >
          <NCheckbox
            :disabled="!allowEdit"
            :checked="restrictIssueCreationForSQLReview"
            :label="
              $t(
                'settings.general.workspace.restrict-issue-creation-for-sql-review.title'
              )
            "
            @update:checked="handleRestrictIssueCreationForSQLReviewToggle"
          />
        </label>
      </template>
      <span class="text-sm text-gray-400 -translate-y-2">
        {{ $t("settings.general.workspace.only-admin-can-edit") }}
      </span>
    </NTooltip>
    <div class="mb-3 text-sm text-gray-400">
      {{
        $t(
          "settings.general.workspace.restrict-issue-creation-for-sql-review.description"
        )
      }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { NTooltip, NCheckbox } from "naive-ui";
import { computed, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification, usePolicyV1Store } from "@/store";
import {
  PolicyResourceType,
  PolicyType,
} from "@/types/proto/v1/org_policy_service";

const props = defineProps<{
  resource: string;
  allowEdit: boolean;
}>();

const { t } = useI18n();
const policyV1Store = usePolicyV1Store();

watchEffect(async () => {
  await policyV1Store.getOrFetchPolicyByParentAndType({
    parentPath: props.resource,
    policyType: PolicyType.RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW,
  });
});

const restrictIssueCreationForSQLReview = computed((): boolean => {
  return (
    policyV1Store.getPolicyByParentAndType({
      parentPath: props.resource,
      policyType: PolicyType.RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW,
    })?.restrictIssueCreationForSqlReviewPolicy?.disallow ?? false
  );
});

const handleRestrictIssueCreationForSQLReviewToggle = async (on: boolean) => {
  await policyV1Store.createPolicy(props.resource, {
    type: PolicyType.RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW,
    resourceType: PolicyResourceType.PROJECT,
    restrictIssueCreationForSqlReviewPolicy: {
      disallow: on,
    },
  });

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.general.workspace.config-updated"),
  });
};
</script>
