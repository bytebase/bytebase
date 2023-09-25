<template>
  <div class="flex flex-row items-center gap-2">
    <slot name="result" :advices="advices" :is-running="isRunning" />

    <NPopover :disabled="tooltipDisabled" to="body">
      <template #trigger>
        <NButton
          style="--n-padding: 0 14px 0 10px"
          :disabled="buttonDisabled"
          :style="buttonStyle"
          tag="div"
          v-bind="buttonProps"
          @click="runChecks"
        >
          <template #icon>
            <BBSpin v-if="isRunning" class="w-4 h-4" />
            <heroicons-outline:play v-else class="w-4 h-4" />
          </template>
          <template #default>
            <template v-if="isRunning">
              {{ $t("task.checking") }}
            </template>
            <template v-else>
              {{ $t("task.run-checks") }}
            </template>
          </template>
        </NButton>
      </template>
      <template #default>
        <template v-if="noReviewPolicyTips">
          <i18n-t :keypath="noReviewPolicyTips" tag="div">
            <template #environment>
              <span>{{ database.effectiveEnvironmentEntity.title }}</span>
            </template>
            <template #link>
              <router-link
                v-if="hasManageSQLReviewPolicyPermission"
                :to="`/setting/sql-review`"
                class="ml-1 normal-link underline"
              >
                {{ $t("common.go-to-configure") }}
              </router-link>
            </template>
          </i18n-t>
        </template>
        <ErrorList v-else :errors="errors ?? []" />
      </template>
    </NPopover>
  </div>
</template>

<script lang="ts" setup>
import { ButtonProps, NButton, NPopover } from "naive-ui";
import { CSSProperties, computed, ref } from "vue";
import { sqlServiceClient } from "@/grpcweb";
import { usePolicyByParentAndType } from "@/store";
import { ComposedDatabase } from "@/types";
import { PolicyType } from "@/types/proto/v1/org_policy_service";
import { Advice } from "@/types/proto/v1/sql_service";
import { useWorkspacePermissionV1 } from "@/utils";
import ErrorList from "../misc/ErrorList.vue";

const props = defineProps<{
  statement: string;
  database: ComposedDatabase;
  buttonProps?: ButtonProps;
  buttonStyle?: string | CSSProperties;
  errors?: string[];
}>();

const isRunning = ref(false);
const advices = ref<Advice[]>();

const reviewPolicy = usePolicyByParentAndType(
  computed(() => ({
    parentPath: props.database.effectiveEnvironment,
    policyType: PolicyType.SQL_REVIEW,
  }))
);

const hasManageSQLReviewPolicyPermission = useWorkspacePermissionV1(
  "bb.permission.workspace.manage-sql-review-policy"
);

const noReviewPolicyTips = computed(() => {
  if (!reviewPolicy.value?.sqlReviewPolicy) {
    if (hasManageSQLReviewPolicyPermission.value) {
      return "issue.sql-check.no-configured-sql-review-policy.admin";
    } else {
      return "issue.sql-check.no-configured-sql-review-policy.developer";
    }
  }
  return "";
});

const buttonDisabled = computed(() => {
  if (noReviewPolicyTips.value) return true;
  if (!props.statement) return true;
  return props.errors && props.errors.length > 0;
});
const tooltipDisabled = computed(() => {
  return !buttonDisabled.value;
});

const runChecks = async () => {
  const { statement, database } = props;
  isRunning.value = true;
  if (!advices.value) {
    advices.value = [];
  }
  try {
    const result = await sqlServiceClient.check({
      statement,
      database: database.name,
    });
    advices.value = result.advices;
  } finally {
    isRunning.value = false;
  }
};
</script>
