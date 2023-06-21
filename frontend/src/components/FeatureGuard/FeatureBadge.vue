<template>
  <router-link
    v-if="!hasFeature"
    :to="`/setting/subscription${
      instanceMissingLicense ? '?manageLicense=1' : ''
    }`"
    exact-active-class=""
  >
    <heroicons-solid:sparkles class="text-accent w-5 h-5" />
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
