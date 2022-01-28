<template>
  <BBModal
    :title="$t('subscription.disabled-feature')"
    @close="$emit('cancel')"
  >
    <div>
      <div class="flex items-start">
        <div
          class="mx-auto flex-shrink-0 flex items-center justify-center h-12 w-12 rounded-full bg-yellow-100 sm:mx-0 sm:h-10 sm:w-10"
        >
          <!-- Heroicons name: outline/exclamation -->
          <heroicons-outline:exclamation class="h-6 w-6 text-yellow-600" />
        </div>
        <h3
          id="modal-headline"
          class="ml-4 flex self-center text-lg leading-6 font-medium text-gray-900"
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
        <p class="whitespace-pre-wrap">{{ $t("subscription.trial") }}*</p>
        <p class="text-gray-500 whitespace-pre-wrap text-sm">
          {{ $t("subscription.trial-comment") }}
        </p>
      </div>
      <div class="mt-7 flex justify-end">
        <button
          type="button"
          class="btn-normal mt-3 px-4 py-2 sm:mt-0 sm:w-auto"
          @click.prevent="$emit('cancel')"
        >
          {{ $t("common.dismiss") }}
        </button>
        <button
          type="button"
          class="sm:ml-3 inline-flex justify-center w-full rounded-md border border-transparent shadow-sm px-4 py-2 bg-error text-base font-medium text-white hover:bg-error-hover focus:outline-none focus-visible:ring-2 focus:ring-offset-2 focus:ring-red-500 sm:w-auto sm:text-sm btn-primary"
          @click.prevent="ok"
        >
          {{ $t("common.learn-more") }}
        </button>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts">
import { PropType } from "vue";
import { FeatureType } from "../types";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";

export default {
  props: {
    feature: {
      required: true,
      type: String as PropType<FeatureType>,
    },
  },
  emits: ["cancel"],
  setup(props, { emit }) {
    const { t } = useI18n();
    const router = useRouter();

    const cancel = () => {
      emit("cancel");
    };

    const ok = () => {
      router.push({ name: "setting.workspace.subscription" });
    };

    const featureKey = props.feature.split(".").join("-");

    return {
      ok,
      featureDesc: t(`subscription.features.${featureKey}.desc`),
      featureTitle: t(`subscription.features.${featureKey}.title`),
    };
  },
};
</script>
