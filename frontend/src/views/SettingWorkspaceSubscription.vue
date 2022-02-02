<template>
  <div class="mx-auto">
    <div class="textinfolabel">
      {{ $t("subscription.description") }}
      <a class="text-accent" href="https://hub.bytebase.com" target="__blank">
        {{ $t("subscription.description-highlight") }}
      </a>
    </div>
    <dl class="text-left grid grid-cols-2 gap-x-6 my-5 sm:grid-cols-4">
      <div class="my-3">
        <dt class="text-gray-400">
          {{ $t("subscription.current") }}
        </dt>
        <dd class="text-indigo-600 mt-1 text-4xl">
          {{ currentPlan }}
        </dd>
      </div>
      <div class="my-3">
        <dt class="text-gray-400">
          {{ $t("subscription.instance-count") }}
        </dt>
        <dd class="mt-1 text-4xl">{{ instanceCount }}</dd>
      </div>
      <div class="my-3 col-span-2">
        <dt class="text-gray-400">
          {{ $t("subscription.expires-at") }}
        </dt>
        <dd class="mt-1 text-4xl">{{ expiresAt }}</dd>
      </div>
    </dl>
    <div class="w-full mt-5 flex flex-col">
      <textarea
        id="license"
        v-model="state.license"
        type="text"
        name="license"
        :placeholder="$t('subscription.sensitive-placeholder')"
        class="shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full sm:text-sm border-gray-300 rounded-md"
      />
      <button
        type="button"
        :class="[
          disabled ? 'cursor-not-allowed' : '',
          'btn-primary inline-flex justify-center ml-auto mt-3',
        ]"
        target="_blank"
        href="https://hub.bytebase.com"
        @click="uploadLicense"
      >
        {{ $t("subscription.upload-license") }}
      </button>
    </div>
    <div class="sm:flex sm:flex-col sm:align-center pt-5 mt-5 border-t">
      <div class="textinfolabel">
        {{ $t("subscription.plan-compare") }}
      </div>
      <PricingTable :subscription="subscription" />
    </div>
  </div>
</template>

<script lang="ts">
import { computed, reactive } from "vue";
import { useStore } from "vuex";
import { useI18n } from "vue-i18n";
import dayjs from "dayjs";
import PricingTable from "../components/PricingTable.vue";
import { PlanType, Subscription } from "../types";

interface LocalState {
  loading: boolean;
  license: string;
}

export default {
  name: "SettingWorkspaceSubscription",
  components: {
    PricingTable,
  },
  setup() {
    const store = useStore();
    const { t } = useI18n();

    const state = reactive<LocalState>({
      loading: false,
      license: "",
    });

    const disabled = computed((): boolean => {
      return state.loading || !state.license;
    });

    const uploadLicense = async () => {
      if (disabled.value) return;
      state.loading = true;

      try {
        await store.dispatch("subscription/patchSubscription", state.license);
      } finally {
        state.loading = false;
      }
    };

    const subscription = computed((): Subscription | undefined => {
      return store.getters["subscription/subscription"]();
    });

    const instanceCount = computed((): number => {
      return subscription.value?.instanceCount ?? 5;
    });

    const expiresAt = computed((): string => {
      const expiresTs = subscription.value?.expiresTs ?? 0;
      if (expiresTs <= 0) {
        return "n/a";
      }
      return dayjs(expiresTs * 1000).format("YYYY-MM-DD");
    });

    const currentPlan = computed((): string => {
      const plan = store.getters["subscription/currentPlan"]();
      switch (plan) {
        case PlanType.TEAM:
          return t("setting.plan.team");
        case PlanType.ENTERPRISE:
          return t("setting.plan.enterprise");
        default:
          return t("setting.plan.free");
      }
    });

    return {
      state,
      disabled,
      expiresAt,
      currentPlan,
      instanceCount,
      subscription,
      uploadLicense,
    };
  },
};
</script>
