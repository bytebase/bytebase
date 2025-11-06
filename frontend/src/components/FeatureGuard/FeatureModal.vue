<template>
  <BBModal v-if="open && feature" :title="title" @close="$emit('cancel')">
    <div class="min-w-0 md:min-w-[400px] max-w-4xl mb-2">
      <div class="flex items-start gap-x-2 mt-3">
        <div class="flex items-center">
          <heroicons-solid:lock-closed
            v-if="instanceMissingLicense"
            class="text-accent w-6 h-6"
          />
          <SparklesIcon v-else class="h-6 w-6 text-accent" />
        </div>
        <h3
          id="modal-headline"
          class="flex self-center text-lg leading-6 font-medium text-gray-900"
        >
          {{ $t(`dynamic.subscription.features.${featureKey}.title`) }}
        </h3>
      </div>
      <div class="mt-4">
        <p class="whitespace-pre-wrap">
          {{ $t(`dynamic.subscription.features.${featureKey}.desc`) }}
        </p>
      </div>
      <div class="mt-3">
        <p class="whitespace-pre-wrap">
          <i18n-t
            v-if="instanceMissingLicense"
            keypath="subscription.instance-assignment.missing-license-attention"
          />
          <i18n-t
            v-else-if="requiredPlan !== PlanType.FREE"
            keypath="subscription.required-plan-with-trial"
          >
            <template #requiredPlan>
              <span class="font-bold text-accent">
                {{
                  $t(
                    `subscription.plan.${PlanType[requiredPlan].toLowerCase()}.title`
                  )
                }}
              </span>
            </template>
            <template v-if="!hasPermission" #startTrial>
              {{ $t("subscription.contact-to-upgrade") }}
            </template>
            <template v-else #startTrial>
              {{
                $t("subscription.trial-for-days", {
                  days: subscriptionStore.trialingDays,
                })
              }}
            </template>
          </i18n-t>
          <i18n-t v-else keypath="subscription.trial-for-days">
            <template #days>
              {{ subscriptionStore.trialingDays }}
            </template>
          </i18n-t>
        </p>
      </div>
      <div class="mt-7 flex justify-end gap-x-2">
        <NButton
          v-if="!hasPermission"
          type="primary"
          @click.prevent="$emit('cancel')"
        >
          {{ $t("common.ok") }}
        </NButton>
        <template v-else-if="subscriptionStore.showTrial">
          <NButton type="primary" @click.prevent="trialSubscription">
            {{
              $t("subscription.request-n-days-trial", {
                days: subscriptionStore.trialingDays,
              })
            }}
          </NButton>
        </template>
        <NButton v-else type="primary" @click.prevent="ok">
          {{
            instanceMissingLicense
              ? $t("subscription.instance-assignment.assign-license")
              : $t("common.learn-more")
          }}
        </NButton>
      </div>
    </div>
  </BBModal>
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
import { SparklesIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBModal } from "@/bbkit";
import { useLanguage } from "@/composables/useLanguage";
import { useSubscriptionV1Store } from "@/store";
import { ENTERPRISE_INQUIRE_LINK } from "@/types";
import type {
  Instance,
  InstanceResource,
} from "@/types/proto-es/v1/instance_service_pb";
import {
  PlanFeature,
  PlanType,
} from "@/types/proto-es/v1/subscription_service_pb";
import { autoSubscriptionRoute, hasWorkspacePermissionV2 } from "@/utils";
import InstanceAssignment from "../InstanceAssignment.vue";
import WeChatQRModal from "../WeChatQRModal.vue";

interface LocalState {
  showInstanceAssignmentDrawer: boolean;
  showQRCodeModal: boolean;
}

const props = withDefaults(
  defineProps<{
    open: boolean;
    feature?: PlanFeature;
    instance?: Instance | InstanceResource;
  }>(),
  {
    feature: PlanFeature.FEATURE_UNSPECIFIED,
    instance: undefined,
  }
);

const state = reactive<LocalState>({
  showInstanceAssignmentDrawer: false,
  showQRCodeModal: false,
});

const emit = defineEmits(["cancel"]);
const { t } = useI18n();
const router = useRouter();
const { locale } = useLanguage();

const subscriptionStore = useSubscriptionV1Store();
const hasPermission = hasWorkspacePermissionV2("bb.settings.set");

const instanceMissingLicense = computed(() => {
  return subscriptionStore.instanceMissingLicense(
    props.feature,
    props.instance
  );
});

const title = computed(() => {
  if (instanceMissingLicense.value) {
    return t("subscription.instance-assignment.require-license");
  }
  return t("subscription.disabled-feature");
});

const ok = () => {
  if (instanceMissingLicense.value) {
    state.showInstanceAssignmentDrawer = true;
  } else {
    router.push(autoSubscriptionRoute(router));
  }
  emit("cancel");
};

const requiredPlan = computed(() =>
  subscriptionStore.getMinimumRequiredPlan(props.feature)
);

const featureKey = computed(() => {
  return PlanFeature[props.feature].split(".").join("-");
});

const trialSubscription = () => {
  if (locale.value === "zh-CN") {
    state.showQRCodeModal = true;
  } else {
    window.open(ENTERPRISE_INQUIRE_LINK, "_blank");
  }
};
</script>

<style scoped>
@media (min-width: 768px) {
  .md\:min-w-400 {
    min-width: 400px;
  }
}
</style>
