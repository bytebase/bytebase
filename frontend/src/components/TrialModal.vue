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
                $t(`subscription.plan.${planTypeToString(PlanType.TEAM)}.title`)
              }}
            </span>
          </template>
        </i18n-t>
      </p>
      <div class="mt-7 flex justify-end space-x-2">
        <button
          type="button"
          class="btn-normal"
          @click.prevent="$emit('cancel')"
        >
          {{ $t("common.dismiss") }}
        </button>

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
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useSubscriptionStore, pushNotification } from "@/store";
import { PlanType, planTypeToString } from "@/types";

const emit = defineEmits(["cancel"]);
const { t } = useI18n();
const router = useRouter();
const subscriptionStore = useSubscriptionStore();

const learnMore = () => {
  router.push({ name: "setting.workspace.subscription" });
};

const trialSubscription = () => {
  subscriptionStore.trialSubscription(PlanType.TEAM).then(() => {
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
