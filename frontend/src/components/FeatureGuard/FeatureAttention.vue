<template>
  <BBAttention
    v-if="!hasFeature"
    :class="customClass"
    type="warning"
    :title="$t(`dynamic.subscription.features.${featureKey}.title`)"
    :description="descriptionText"
    :action-text="actionText"
    @click="onClick"
  />
  <WeChatQRModal
    v-if="state.showQRCodeModal"
    :title="$t('subscription.request-with-qr')"
    @close="state.showQRCodeModal = false"
  />
  <InstanceAssignment
    :show="state.showInstanceAssignmentDrawer"
    @dismiss="state.showInstanceAssignmentDrawer = false"
  />
</template>

<script lang="ts" setup>
import type { PropType } from "vue";
import { reactive, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBAttention } from "@/bbkit";
import { useLanguage } from "@/composables/useLanguage";
import { useSubscriptionV1Store } from "@/store";
import type { FeatureType } from "@/types";
import { planTypeToString, ENTERPRISE_INQUIRE_LINK } from "@/types";
import type {
  Instance,
  InstanceResource,
} from "@/types/proto/api/v1alpha/instance_service";
import { autoSubscriptionRoute, hasWorkspacePermissionV2 } from "@/utils";
import InstanceAssignment from "../InstanceAssignment.vue";
import WeChatQRModal from "../WeChatQRModal.vue";

interface LocalState {
  showInstanceAssignmentDrawer: boolean;
  showQRCodeModal: boolean;
}

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
  instance: {
    type: Object as PropType<Instance | InstanceResource>,
    default: undefined,
  },
});

const router = useRouter();
const { t } = useI18n();
const { locale } = useLanguage();

const subscriptionStore = useSubscriptionV1Store();
const state = reactive<LocalState>({
  showInstanceAssignmentDrawer: false,
  showQRCodeModal: false,
});

const hasPermission = computed(() =>
  hasWorkspacePermissionV2("bb.settings.set")
);

const canManageInstanceLicense = computed((): boolean => {
  return hasWorkspacePermissionV2("bb.instances.update");
});

const hasFeature = computed(() => {
  return subscriptionStore.hasInstanceFeature(props.feature, props.instance);
});

const instanceMissingLicense = computed(() => {
  return subscriptionStore.instanceMissingLicense(
    props.feature,
    props.instance
  );
});

const actionText = computed(() => {
  if (instanceMissingLicense.value) {
    if (!canManageInstanceLicense.value) {
      return "";
    }
    return t("subscription.instance-assignment.assign-license");
  }
  if (!hasPermission.value) {
    return "";
  }
  if (!subscriptionStore.canTrial) {
    return t("subscription.upgrade");
  }
  if (subscriptionStore.canUpgradeTrial) {
    return t("subscription.upgrade-trial-button");
  }
  return t("subscription.request-n-days-trial", {
    days: subscriptionStore.trialingDays,
  });
});

const featureKey = props.feature.split(".").join("-");

const descriptionText = computed(() => {
  let description = props.description;
  if (!description) {
    description = t(`dynamic.subscription.features.${featureKey}.desc`);
  }

  if (instanceMissingLicense.value) {
    const attention = t(
      "subscription.instance-assignment.missing-license-attention"
    );
    return `${description}\n${attention}`;
  }

  const startTrial = subscriptionStore.canUpgradeTrial
    ? t("subscription.upgrade-trial")
    : subscriptionStore.isTrialing
      ? ""
      : t("subscription.trial-for-days", {
          days: subscriptionStore.trialingDays,
        });
  if (!Array.isArray(subscriptionStore.featureMatrix.get(props.feature))) {
    return `${description}\n${startTrial}`;
  }

  const requiredPlan = subscriptionStore.getMinimumRequiredPlan(props.feature);
  const trialText = t("subscription.required-plan-with-trial", {
    requiredPlan: t(
      `subscription.plan.${planTypeToString(requiredPlan)}.title`
    ),
    startTrial: startTrial,
  });

  return `${description}\n${trialText}`;
});

const onClick = () => {
  if (instanceMissingLicense.value) {
    state.showInstanceAssignmentDrawer = true;
  } else if (subscriptionStore.canTrial) {
    if (locale.value === "zh-CN") {
      state.showQRCodeModal = true;
    } else {
      window.open(ENTERPRISE_INQUIRE_LINK, "_blank");
    }
  } else {
    router.push(autoSubscriptionRoute(router));
  }
};
</script>
