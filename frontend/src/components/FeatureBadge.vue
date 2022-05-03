<template>
  <router-link v-if="!hasFeature" to="/setting/subscription" exact-active-class>
    <heroicons-solid:sparkles class="w-5 h-5" />
  </router-link>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { FeatureType } from "../types";
import { featureToRef } from "@/store";

export default defineComponent({
  props: {
    feature: {
      required: true,
      type: String as PropType<FeatureType>,
    },
  },
  setup(props) {
    const hasFeature = featureToRef(props.feature);

    return {
      hasFeature,
    };
  },
});
</script>
