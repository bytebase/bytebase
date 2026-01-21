<template>
  <div id="security" class="py-6 lg:flex gap-y-4 lg:gap-y-0">
    <div class="text-left lg:w-1/4">
      <div class="flex items-center gap-x-2">
        <h1 class="text-2xl font-bold">
          {{ title }}
        </h1>
      </div>
    </div>

    <div class="flex-1 lg:px-4 flex flex-col gap-y-6">
      <QueryDataPolicySetting
        ref="queryDataPolicySettingRef"
        resource=""
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import QueryDataPolicySetting from "./QueryDataPolicySetting.vue";

const props = defineProps<{
  title: string;
}>();

const queryDataPolicySettingRef =
  ref<InstanceType<typeof QueryDataPolicySetting>>();

const isDirty = computed(() => {
  return queryDataPolicySettingRef.value?.isDirty;
});

const onUpdate = async () => {
  await queryDataPolicySettingRef.value?.update();
};

defineExpose({
  isDirty,
  update: onUpdate,
  title: props.title,
  revert: () => {
    queryDataPolicySettingRef.value?.revert();
  },
});
</script>
