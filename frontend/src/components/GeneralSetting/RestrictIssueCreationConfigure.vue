<template>
  <div>
    <div class="flex items-center gap-x-2">
      <Switch
        :value="restrictIssueCreationForSQLReview"
        :text="true"
        :disabled="!allowEdit"
        @update:value="handleRestrictIssueCreationForSQLReviewToggle"
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
import { computed, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import {
  hasFeature,
  pushNotification,
  useCurrentUserV1,
  usePolicyV1Store,
} from "@/store";
import {
  PolicyResourceType,
  PolicyType,
} from "@/types/proto/v1/org_policy_service";
import { hasWorkspacePermissionV2 } from "@/utils";

const props = defineProps<{
  resource: string;
  allowEdit: boolean;
}>();

const { t } = useI18n();
const me = useCurrentUserV1();
const policyV1Store = usePolicyV1Store();

const allowEdit = computed(() => {
  return (
    props.allowEdit &&
    hasWorkspacePermissionV2(me.value, "bb.policies.update") &&
    hasFeature("bb.feature.access-control")
  );
});

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
