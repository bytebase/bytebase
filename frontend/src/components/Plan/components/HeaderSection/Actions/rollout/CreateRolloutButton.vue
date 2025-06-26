<template>
  <NTooltip :disabled="errors.length === 0" placement="top">
    <template #trigger>
      <NButton
        type="primary"
        size="medium"
        tag="div"
        :disabled="errors.length > 0"
        @click="handleCreateRollout"
      >
        {{ $t("common.create") }} {{ $t("common.rollout") }}
      </NButton>
    </template>

    <template #default>
      <ErrorList :errors="errors" />
    </template>
  </NTooltip>
</template>

<script setup lang="ts">
import { NTooltip, NButton } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { ErrorList } from "@/components/Plan/components/common";
import { usePlanContext, useIssueReviewContext } from "@/components/Plan/logic";
import { useCurrentProjectV1 } from "@/store";
import { hasProjectPermissionV2 } from "@/utils";

const emit = defineEmits<{
  (event: "create-rollout"): void;
}>();

const { t } = useI18n();
const { project } = useCurrentProjectV1();
const { plan } = usePlanContext();
const reviewContext = useIssueReviewContext();

const errors = computed(() => {
  const errorList: string[] = [];

  if (!hasProjectPermissionV2(project.value, "bb.rollouts.create")) {
    errorList.push(t("common.missing-required-permission"));
  }

  if (plan.value.rollout) {
    errorList.push("Rollout already exists for this plan");
  }

  if (!reviewContext.done.value) {
    errorList.push("Issue must pass approval review before creating rollout");
  }

  return errorList;
});

const handleCreateRollout = () => {
  emit("create-rollout");
};
</script>
