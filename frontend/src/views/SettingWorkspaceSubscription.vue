<template>
  <div class="mx-auto">
    <div class="textinfolabel">
      {{ $t("subscription.description") }}
      <a
        class="text-accent"
        href="https://hub.bytebase.com/subscription?source=console.subscription"
        target="__blank"
      >
        {{ $t("subscription.purchase-license") }}
      </a>
      <span v-if="canTrial" class="ml-1">
        {{ $t("common.or") }}
        <span class="text-accent cursor-pointer" @click="openTrialModal">
          {{ $t("subscription.plan.try") }}
        </span>
      </span>
    </div>
    <dl class="text-left grid grid-cols-2 gap-x-6 my-5 xl:grid-cols-4">
      <div class="my-3">
        <dt class="flex text-gray-400">
          {{ $t("subscription.current") }}
          <span
            v-if="isExpired"
            class="ml-2 inline-flex items-center px-3 py-0.5 rounded-full text-base font-sm bg-red-100 text-red-800 h-6"
          >
            {{ $t("subscription.expired") }}
          </span>
          <span
            v-else-if="isTrialing"
            class="ml-2 inline-flex items-center px-3 py-0.5 rounded-full text-base font-sm bg-indigo-100 text-indigo-800 h-6"
          >
            {{ $t("subscription.trialing") }}
          </span>
        </dt>
        <dd class="text-indigo-600 mt-1 text-4xl">
          <div>
            {{ currentPlan }}
          </div>
        </dd>
      </div>
      <div class="my-3">
        <dt class="text-gray-400">
          {{ $t("subscription.instance-count") }}
        </dt>
        <dd class="mt-1 text-4xl">{{ instanceCount }}</dd>
      </div>
      <div class="my-3">
        <dt class="text-gray-400">
          {{ $t("subscription.seat-count") }}
        </dt>
        <dd class="mt-1 text-4xl">{{ seatCount }}</dd>
      </div>
      <div class="my-3">
        <dt class="text-gray-400">
          {{ $t("subscription.expires-at") }}
        </dt>
        <dd class="mt-1 text-4xl">{{ expireAt || "n/a" }}</dd>
      </div>
    </dl>
    <div v-if="canManageSubscription" class="w-full mt-5 flex flex-col">
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
        class="btn-primary inline-flex justify-center ml-auto mt-3"
        :disabled="disabled"
        target="_blank"
        @click="uploadLicense"
      >
        {{ $t("subscription.upload-license") }}
      </button>
    </div>
    <div class="sm:flex sm:flex-col sm:align-center pt-5 mt-5 border-t">
      <div class="textinfolabel">
        {{ $t("subscription.plan-compare") }}
      </div>
      <PricingTable @on-trial="openTrialModal" />
    </div>
    <TrialModal
      v-if="state.showTrialModal"
      @cancel="state.showTrialModal = false"
    />
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, reactive } from "vue";
import { useI18n } from "vue-i18n";
import PricingTable from "../components/PricingTable/";
import { PlanType } from "../types";
import {
  pushNotification,
  useCurrentUser,
  useSubscriptionStore,
} from "@/store";
import { storeToRefs } from "pinia";
import { hasWorkspacePermission } from "@/utils";

interface LocalState {
  loading: boolean;
  license: string;
  showTrialModal: boolean;
}

export default defineComponent({
  name: "SettingWorkspaceSubscription",
  components: {
    PricingTable,
  },
  setup() {
    const subscriptionStore = useSubscriptionStore();
    const { t } = useI18n();
    const currentUser = useCurrentUser();

    const state = reactive<LocalState>({
      loading: false,
      license: "",
      showTrialModal: false,
    });

    const disabled = computed((): boolean => {
      return state.loading || !state.license;
    });

    const uploadLicense = async () => {
      if (disabled.value) return;
      state.loading = true;

      try {
        await subscriptionStore.patchSubscription(state.license);
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("subscription.update.success.title"),
          description: t("subscription.update.success.description"),
        });
      } catch {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("subscription.update.failure.title"),
          description: t("subscription.update.failure.description"),
        });
      } finally {
        state.loading = false;
        state.license = "";
      }
    };

    const { subscription, expireAt, isTrialing, isExpired } =
      storeToRefs(subscriptionStore);

    const instanceCount = computed((): string => {
      const count = subscription.value?.instanceCount ?? 5;
      if (count > 0) {
        return `${count}`;
      }
      return t("subscription.unlimited");
    });

    const seatCount = computed((): string => {
      const seat = subscription.value?.seat ?? 2;
      if (seat > 0) {
        return `${seat}`;
      }
      return t("subscription.unlimited");
    });

    const currentPlan = computed((): string => {
      const plan = subscriptionStore.currentPlan;
      switch (plan) {
        case PlanType.TEAM:
          return t("subscription.plan.team.title");
        case PlanType.ENTERPRISE:
          return t("subscription.plan.enterprise.title");
        default:
          return t("subscription.plan.free.title");
      }
    });

    const canTrial = computed((): boolean => {
      return subscriptionStore.canTrial;
    });

    const openTrialModal = () => {
      state.showTrialModal = true;
    };

    const canManageSubscription = computed((): boolean => {
      return hasWorkspacePermission(
        "bb.permission.workspace.manage-subscription",
        currentUser.value.role
      );
    });

    return {
      state,
      disabled,
      canTrial,
      expireAt,
      isTrialing,
      isExpired,
      seatCount,
      currentPlan,
      instanceCount,
      uploadLicense,
      openTrialModal,
      canManageSubscription,
    };
  },
});
</script>
