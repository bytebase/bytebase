<template>
  <div
    v-if="show || instanceMissingLicense"
    :class="['text-accent cursor-pointer', customClass]"
    @click="state.showInstanceAssignmentDrawer = true"
  >
    <NTooltip :show-arrow="true">
      <template #trigger>
        <heroicons-outline:exclamation class="text-warning w-5 h-5" />
      </template>
      <span class="w-56 text-sm">
        {{
          $t("subscription.instance-assignment.missing-license-for-feature", {
            feature: $t(
              `subscription.features.${feature.split(".").join("-")}.title`
            ).toLowerCase(),
          })
        }}
      </span>
    </NTooltip>
  </div>
  <InstanceAssignment
    v-if="!hasFeature || show"
    :show="state.showInstanceAssignmentDrawer"
    @dismiss="state.showInstanceAssignmentDrawer = false"
  />
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import { reactive, PropType, computed } from "vue";
import { useSubscriptionV1Store } from "@/store";
import { FeatureType } from "@/types";
import { Instance } from "@/types/proto/v1/instance_service";

interface LocalState {
  showInstanceAssignmentDrawer: boolean;
}

const props = defineProps({
  show: {
    type: Boolean,
    default: false,
  },
  feature: {
    required: true,
    type: String as PropType<FeatureType>,
  },
  instance: {
    type: Object as PropType<Instance>,
    default: undefined,
  },
  customClass: {
    require: false,
    default: "",
    type: String,
  },
});

const state = reactive<LocalState>({
  showInstanceAssignmentDrawer: false,
});

const subscriptionStore = useSubscriptionV1Store();

const hasFeature = computed(() => {
  return subscriptionStore.hasInstanceFeature(props.feature, props.instance);
});

const instanceMissingLicense = computed(() => {
  return subscriptionStore.instanceMissingLicense(
    props.feature,
    props.instance
  );
});
</script>
