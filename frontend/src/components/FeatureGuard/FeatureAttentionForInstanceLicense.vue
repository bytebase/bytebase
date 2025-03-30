<template>
  <BBAttention
    v-if="
      existInstanceWithoutLicense &&
      instanceLimitFeature.has(feature) &&
      subscriptionV1Store.currentPlan !== PlanType.FREE
    "
    :class="customClass"
    :type="type ?? `info`"
    :title="$t(`dynamic.subscription.features.${featureKey}.desc`)"
    :description="
      $t('subscription.instance-assignment.missing-license-attention')
    "
    :action-text="
      canManageSubscription
        ? $t('subscription.instance-assignment.assign-license')
        : ''
    "
    @click="onClick"
  />
  <InstanceAssignment
    :show="state.showInstanceAssignmentDrawer"
    @dismiss="state.showInstanceAssignmentDrawer = false"
  />
</template>

<script lang="ts" setup>
import { reactive, computed } from "vue";
import { BBAttention } from "@/bbkit";
import { useSubscriptionV1Store, useActuatorV1Store } from "@/store";
import type { FeatureType } from "@/types";
import { instanceLimitFeature } from "@/types";
import { PlanType } from "@/types/proto/v1/subscription_service";
import { hasWorkspacePermissionV2 } from "@/utils";
import InstanceAssignment from "../InstanceAssignment.vue";

interface LocalState {
  showInstanceAssignmentDrawer: boolean;
}

const props = defineProps<{
  type?: "warning" | "info";
  feature: FeatureType;
  customClass?: string;
}>();

const state = reactive<LocalState>({
  showInstanceAssignmentDrawer: false,
});

const subscriptionV1Store = useSubscriptionV1Store();
const actuatorStore = useActuatorV1Store();
const featureKey = props.feature.split(".").join("-");

const onClick = () => {
  state.showInstanceAssignmentDrawer = true;
};

const canManageSubscription = computed((): boolean => {
  return hasWorkspacePermissionV2("bb.instances.update");
});

const existInstanceWithoutLicense = computed(() => {
  return (
    actuatorStore.totalInstanceCount > actuatorStore.activatedInstanceCount
  );
});
</script>
