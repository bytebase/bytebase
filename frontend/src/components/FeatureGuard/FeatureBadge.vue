<template>
  <router-link
    v-if="!hasFeature"
    :to="`/setting/subscription${
      instanceMissingLicense ? '?manageLicense=1' : ''
    }`"
    exact-active-class=""
    class="tooltip-wrapper"
  >
    <heroicons-solid:lock-closed
      v-if="instanceMissingLicense"
      class="text-accent w-5 h-5"
    />
    <heroicons-solid:sparkles v-else class="text-accent w-5 h-5" />
    <span
      v-if="instanceMissingLicense"
      class="w-56 text-sm -translate-y-full -translate-x-1/3 tooltip"
    >
      {{ $t("subscription.instance-assignment.missing-license-attention") }}
    </span>
  </router-link>
</template>

<script lang="ts" setup>
import { PropType, computed } from "vue";
import { FeatureType } from "@/types";
import { useSubscriptionV1Store } from "@/store";
import { Instance } from "@/types/proto/v1/instance_service";

const props = defineProps({
  feature: {
    required: true,
    type: String as PropType<FeatureType>,
  },
  instance: {
    type: Object as PropType<Instance>,
    default: undefined,
  },
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
