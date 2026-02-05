<template>
  <div class="w-full flex flex-col justify-start items-start gap-y-6">
    <SQLReviewForResource
      ref="sqlReviewForResourceRef"
      :resource="project.name"
    />

    <MaximumSQLResultSizeSetting
      ref="maximumSQLResultSizeSettingRef"
      :resource="project.name"
      :policy="policyPayload"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { SQLReviewForResource } from "@/components/SQLReview";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import MaximumSQLResultSizeSetting from "@/components/GeneralSetting/MaximumSQLResultSizeSetting.vue";
import {
  usePolicyByParentAndType,
  usePolicyV1Store,
} from "@/store";
import {
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";

const props = defineProps<{
  project: Project;
}>();

const sqlReviewForResourceRef =
  ref<InstanceType<typeof SQLReviewForResource>>();
const maximumSQLResultSizeSettingRef =
  ref<InstanceType<typeof MaximumSQLResultSizeSetting>>();

const { ready } = usePolicyByParentAndType(
  computed(() => ({
    parentPath: props.project.name,
    policyType: PolicyType.DATA_QUERY,
  }))
);

watch(
  () => ready.value,
  (ready) => {
    if (ready) {
      maximumSQLResultSizeSettingRef.value?.revert();
    }
  }
);

const policyV1Store = usePolicyV1Store();
const policyPayload = computed(() => {
  return policyV1Store.getQueryDataPolicyByParent(props.project.name);
});

const isDirty = computed(() =>
  sqlReviewForResourceRef.value?.isDirty || maximumSQLResultSizeSettingRef.value?.isDirty
);

const onUpdate = async () => {
  if (sqlReviewForResourceRef.value?.isDirty) {
    await sqlReviewForResourceRef.value.update();
  }
  if (maximumSQLResultSizeSettingRef.value?.isDirty) {
    await maximumSQLResultSizeSettingRef.value.update();
  }
};

const resetState = () => {
  sqlReviewForResourceRef.value?.revert();
  maximumSQLResultSizeSettingRef.value?.revert();
};

defineExpose({
  isDirty,
  update: onUpdate,
  revert: resetState,
});
</script>
