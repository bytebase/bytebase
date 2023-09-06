<template>
  <BBAttention
    v-if="
      instanceLimitFeature.has(feature) &&
      subscriptionV1Store.currentPlan !== PlanType.FREE
    "
    :class="customClass"
    :style="style ?? `INFO`"
    :title="$t(`subscription.features.${featureKey}.desc`)"
    :description="
      $t('subscription.instance-assignment.missing-license-attention')
    "
    :action-text="$t('subscription.instance-assignment.assign-license')"
    @click-action="onClick"
  />
  <InstanceAssignment
    :show="state.showInstanceAssignmentDrawer"
    @dismiss="state.showInstanceAssignmentDrawer = false"
  />
</template>

<script lang="ts" setup>
import { reactive } from "vue";
import { BBAttentionStyle } from "@/bbkit";
import { useSubscriptionV1Store } from "@/store";
import { FeatureType, instanceLimitFeature } from "@/types";
import { PlanType } from "@/types/proto/v1/subscription_service";

interface LocalState {
  showInstanceAssignmentDrawer: boolean;
}

const props = defineProps<{
  style?: BBAttentionStyle;
  feature: FeatureType;
  customClass?: string;
}>();

const state = reactive<LocalState>({
  showInstanceAssignmentDrawer: false,
});

const subscriptionV1Store = useSubscriptionV1Store();
const featureKey = props.feature.split(".").join("-");

const onClick = () => {
  state.showInstanceAssignmentDrawer = true;
};
</script>
