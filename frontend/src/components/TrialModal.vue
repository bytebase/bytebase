<template>
  <BBModal
    :title="
      $t('subscription.start-n-days-trial', {
        days: subscriptionStore.trialingDays,
      })
    "
    @close="$emit('cancel')"
  >
    <div class="min-w-0 md:min-w-400">
      <p class="whitespace-pre-wrap">
        <i18n-t keypath="subscription.trial-for-plan">
          <template #days>
            {{ subscriptionStore.trialingDays }}
          </template>
          <template #plan>
            <span class="font-bold text-accent">
              {{
                $t(
                  `subscription.plan.${planTypeToString(
                    PlanType.ENTERPRISE
                  )}.title`
                )
              }}
            </span>
          </template>
        </i18n-t>
      </p>
      <div class="mt-7 flex justify-end space-x-2">
        <button
          v-if="subscriptionStore.canTrial"
          type="button"
          class="btn-primary"
          @click.prevent="trialSubscription"
        >
          {{
            $t("subscription.start-n-days-trial", {
              days: subscriptionStore.trialingDays,
            })
          }}
        </button>
        <button
          v-else
          type="button"
          class="btn-primary"
          @click.prevent="learnMore"
        >
          {{ $t("common.learn-more") }}
        </button>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useSubscriptionV1Store, pushNotification } from "@/store";
import { planTypeToString } from "@/types";
import { PlanType } from "@/types/proto/v1/subscription_service";

const emit = defineEmits(["cancel"]);
const { t } = useI18n();
const router = useRouter();
const subscriptionStore = useSubscriptionV1Store();

const learnMore = () => {
  router.push({ name: "setting.workspace.subscription" });
};

const trialSubscription = () => {
  subscriptionStore.trialSubscription(PlanType.ENTERPRISE).then(() => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.success"),
      description: t("subscription.successfully-start-trial", {
        days: subscriptionStore.trialingDays,
      }),
    });
    emit("cancel");
  });
};
</script>
