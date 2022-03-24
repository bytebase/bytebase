<template>
  <router-link v-if="!hasFeature" to="/setting/subscription" exact-active-class>
    <heroicons-solid:sparkles class="w-5 h-5"/>
  </router-link>
</template>

<script lang="ts">
import { PropType, computed } from "vue";
import { useStore } from "vuex";
import { FeatureType } from "../types";

export default {
  props: {
    feature: {
      required: true,
      type: String as PropType<FeatureType>,
    },
  },
  setup(props) {
    const store = useStore();

    const hasFeature = computed(() =>
      store.getters["subscription/feature"](props.feature)
    );

    return {
      hasFeature,
    };
  },
};
</script>
