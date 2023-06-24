<template>
  <BBAttention
    v-if="instanceLimitFeature.has(feature)"
    :class="customClass"
    :style="`INFO`"
    :title="$t(`subscription.features.${featureKey}.title`)"
    :description="descriptionText"
    :action-text="$t('subscription.instance-assignment.assign-license')"
    @click-action="onClick"
  />
  <InstanceAssignment
    :show="state.showInstanceAssignmentDrawer"
    @dismiss="state.showInstanceAssignmentDrawer = false"
  />
</template>

<script lang="ts" setup>
import { reactive, PropType, computed } from "vue";
import { FeatureType, instanceLimitFeature } from "@/types";
import { useI18n } from "vue-i18n";

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

const { t } = useI18n();
const state = reactive<LocalState>({
  showInstanceAssignmentDrawer: false,
});
const featureKey = props.feature.split(".").join("-");

const onClick = () => {
  state.showInstanceAssignmentDrawer = true;
};

const descriptionText = computed(() => {
  const attention = t(
    "subscription.instance-assignment.missing-license-attention"
  );
  const description = t(`subscription.features.${featureKey}.desc`);

  return `${description} ${attention}`;
});
</script>
