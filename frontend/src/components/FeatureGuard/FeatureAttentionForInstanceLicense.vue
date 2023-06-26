<template>
  <BBAttention
    v-if="instanceLimitFeature.has(feature)"
    :class="customClass"
    :style="`INFO`"
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
import { reactive, PropType } from "vue";
import { FeatureType, instanceLimitFeature } from "@/types";

interface LocalState {
  showInstanceAssignmentDrawer: boolean;
}

const props = defineProps({
  feature: {
    required: true,
    type: String as PropType<FeatureType>,
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
const featureKey = props.feature.split(".").join("-");

const onClick = () => {
  state.showInstanceAssignmentDrawer = true;
};
</script>
