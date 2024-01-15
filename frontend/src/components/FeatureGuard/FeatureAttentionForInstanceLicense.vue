<template>
  <BBAttention
    v-if="
      existInstanceWithoutLicense &&
      instanceLimitFeature.has(feature) &&
      subscriptionV1Store.currentPlan !== PlanType.FREE
    "
    :class="customClass"
    :type="type ?? `info`"
    :title="$t(`subscription.features.${featureKey}.desc`)"
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
import {
  useSubscriptionV1Store,
  useCurrentUserV1,
  useInstanceV1List,
} from "@/store";
import { FeatureType, instanceLimitFeature } from "@/types";
import { PlanType } from "@/types/proto/v1/subscription_service";
import { hasWorkspacePermissionV2 } from "@/utils";

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
const featureKey = props.feature.split(".").join("-");
const currentUserV1 = useCurrentUserV1();
const { instanceList } = useInstanceV1List(false /* !showDeleted */);

const onClick = () => {
  state.showInstanceAssignmentDrawer = true;
};

const canManageSubscription = computed((): boolean => {
  return hasWorkspacePermissionV2(currentUserV1.value, "bb.instances.update");
});

const existInstanceWithoutLicense = computed(() => {
  return instanceList.value.some((ins) => !ins.activation);
});
</script>
