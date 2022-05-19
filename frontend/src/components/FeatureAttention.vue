<template>
  <BBAttention
    :class="customClass"
    :style="`WARN`"
    :title="$t(titleKey)"
    :description="description"
    :action-text="$t('subscription.upgrade')"
    @click-action="redirect"
  />
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { FeatureType } from "../types";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";

export default defineComponent({
  props: {
    feature: {
      required: true,
      type: String as PropType<FeatureType>,
    },
    description: {
      require: false,
      default: "",
      type: String,
    },
    customClass: {
      require: false,
      default: "",
      type: String,
    },
  },
  setup(props) {
    const { t } = useI18n();
    const router = useRouter();

    const redirect = () => {
      router.push({ name: "setting.workspace.subscription" });
    };

    const featureKey = props.feature.split(".").join("-");

    return {
      redirect,
      titleKey: `subscription.features.${featureKey}.title`,
    };
  },
});
</script>
