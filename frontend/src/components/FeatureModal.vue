<template>
  <BBModal
    :title="$t('subscription.disabled-feature')"
    @close="$emit('cancel')"
  >
    <div class="min-w-0 md:min-w-400">
      <div class="flex items-start space-x-2">
        <div class="flex items-center">
          <heroicons-solid:sparkles class="h-6 w-6 text-accent" />
        </div>
        <h3
          id="modal-headline"
          class="flex self-center text-lg leading-6 font-medium text-gray-900"
        >
          {{ featureTitle }}
        </h3>
      </div>
      <div class="mt-5">
        <p class="whitespace-pre-wrap">
          {{ featureDesc }}
        </p>
      </div>
      <div class="mt-3">
        <p class="whitespace-pre-wrap">
          {{ $t("subscription.trial-with-plan", { plan: requiredPlan }) }}*
        </p>
      </div>
      <div class="mt-7 flex justify-end space-x-2">
        <button
          type="button"
          class="btn-normal"
          @click.prevent="$emit('cancel')"
        >
          {{ $t("common.dismiss") }}
        </button>
        <button type="button" class="btn-primary" @click.prevent="ok">
          {{ $t("common.learn-more") }}
        </button>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useSubscriptionStore } from "@/store";
import { FeatureType, planTypeToString } from "@/types";

export default defineComponent({
  props: {
    feature: {
      required: true,
      type: String as PropType<FeatureType>,
    },
  },
  emits: ["cancel"],
  setup(props) {
    const { t } = useI18n();
    const router = useRouter();

    const ok = () => {
      router.push({ name: "setting.workspace.subscription" });
    };

    const requiredPlan = useSubscriptionStore().getMinimumRequiredPlan(
      props.feature
    );

    const featureKey = props.feature.split(".").join("-");

    return {
      ok,
      requiredPlan: t(
        `subscription.plan.${planTypeToString(requiredPlan)}.title`
      ),
      featureDesc: t(`subscription.features.${featureKey}.desc`),
      featureTitle: t(`subscription.features.${featureKey}.title`),
    };
  },
});
</script>

<style scoped>
@media (min-width: 768px) {
  .md\:min-w-400 {
    min-width: 400px;
  }
}
</style>
