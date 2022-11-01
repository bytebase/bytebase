<template>
  <BBAttention
    :class="customClass"
    :style="`WARN`"
    :title="$t(`subscription.features.${featureKey}.title`)"
    :description="description"
    :action-text="
      subscriptionStore.canTrial
        ? $t('subscription.start-n-days-trial', {
            days: subscriptionStore.trialingDays,
          })
        : $t('subscription.upgrade')
    "
    @click-action="onClick"
  />
</template>

<script lang="ts" setup>
import { PropType } from "vue";
import { FeatureType } from "../types";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useSubscriptionStore, pushNotification } from "@/store";

const props = defineProps({
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
});

const router = useRouter();
const { t } = useI18n();
const subscriptionStore = useSubscriptionStore();

const onClick = () => {
  if (subscriptionStore.canTrial) {
    subscriptionStore.trialSubscription().then(() => {
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.success"),
        description: t("subscription.successfully-start-trial", {
          days: subscriptionStore.trialingDays,
        }),
      });
    });
  } else {
    router.push({ name: "setting.workspace.subscription" });
  }
};

const featureKey = props.feature.split(".").join("-");
</script>
