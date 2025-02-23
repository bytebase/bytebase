<template>
  <div>
    <div class="flex items-center gap-x-2">
      <Switch
        v-model:value="restrictIssueCreationForSQLReview"
        :text="true"
        :disabled="!allowEdit"
      />
      <span class="textlabel">
        {{
          $t(
            "settings.general.workspace.restrict-issue-creation-for-sql-review.title"
          )
        }}
      </span>
      <FeatureBadge feature="bb.feature.access-control" />
    </div>
    <div class="mt-1 mb-3 text-sm text-gray-400">
      {{
        $t(
          "settings.general.workspace.restrict-issue-creation-for-sql-review.description"
        )
      }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import {
  hasFeature,
  usePolicyV1Store,
  usePolicyByParentAndType,
} from "@/store";
import {
  PolicyResourceType,
  PolicyType,
} from "@/types/proto/v1/org_policy_service";
import { hasWorkspacePermissionV2 } from "@/utils";
import { FeatureBadge } from "../FeatureGuard";
import { Switch } from "../v2";

const props = defineProps<{
  resource: string;
  allowEdit: boolean;
}>();

const policyV1Store = usePolicyV1Store();

const allowEdit = computed(() => {
  return (
    props.allowEdit &&
    hasWorkspacePermissionV2("bb.policies.update") &&
    hasFeature("bb.feature.access-control")
  );
});

const { policy, ready } = usePolicyByParentAndType(
  computed(() => ({
    parentPath: props.resource,
    policyType: PolicyType.RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW,
  }))
);

const restrictIssueCreationForSQLReview = ref<boolean>(false);

const resetState = () => {
  restrictIssueCreationForSQLReview.value =
    policy.value?.restrictIssueCreationForSqlReviewPolicy?.disallow ?? false;
};

watch(
  () => ready.value,
  () => {
    if (ready.value) {
      resetState();
    }
  }
);

const handleRestrictIssueCreationForSQLReviewToggle = async () => {
  await policyV1Store.upsertPolicy({
    parentPath: props.resource,
    policy: {
      type: PolicyType.RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW,
      resourceType: PolicyResourceType.PROJECT,
      restrictIssueCreationForSqlReviewPolicy: {
        disallow: restrictIssueCreationForSQLReview.value,
      },
    },
  });
};

defineExpose({
  isDirty: computed(
    () =>
      restrictIssueCreationForSQLReview.value !==
      (policy.value?.restrictIssueCreationForSqlReviewPolicy?.disallow ?? false)
  ),
  update: handleRestrictIssueCreationForSQLReviewToggle,
  revert: resetState,
});
</script>
