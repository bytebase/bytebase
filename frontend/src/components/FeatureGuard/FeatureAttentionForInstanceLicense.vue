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
</template>

<script lang="ts" setup>
import { PropType, computed } from "vue";
import { FeatureType, instanceLimitFeature } from "@/types";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";

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

const router = useRouter();
const { t } = useI18n();

const featureKey = props.feature.split(".").join("-");

const onClick = () => {
  router.push({
    name: "setting.workspace.subscription",
    query: {
      manageLicense: 1,
    },
  });
};

const descriptionText = computed(() => {
  const attention = t(
    "subscription.instance-assignment.missing-license-attention"
  );
  const description = t(`subscription.features.${featureKey}.desc`);

  return `${description} ${attention}`;
});
</script>
