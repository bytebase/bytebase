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
      <span v-if="subscriptionStore.canTrial" class="ml-1">
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
      <div v-if="subscriptionStore.currentPlan === PlanType.FREE" class="my-3">
        <dt class="text-gray-400">
          {{ $t("subscription.instance-count") }}
        </dt>
        <dd
          class="mt-1 text-4xl flex items-center gap-x-2 cursor-pointer group"
        >
          <span class="group-hover:underline">{{
            subscriptionStore.instanceCountLimit
          }}</span>
        </dd>
      </div>
      <div v-else class="my-3">
        <dt class="text-gray-400">
          {{ $t("subscription.instance-assignment.used-and-total-license") }}
        </dt>
        <dd
          class="mt-1 text-4xl flex items-center gap-x-2 cursor-pointer group"
          @click="state.showInstanceAssignmentDrawer = true"
        >
          <span class="group-hover:underline">{{ activateLicenseCount }}</span>
          <span class="text-xl">/</span>
          <span class="group-hover:underline">{{ totalLicenseCount }}</span>
          <heroicons-outline:pencil class="h-6 w-6" />
        </dd>
      </div>
      <div v-if="!subscriptionStore.isFreePlan" class="my-3">
        <dt class="text-gray-400">
          {{ $t("subscription.expires-at") }}
        </dt>
        <dd class="mt-1 text-4xl">{{ expireAt || "n/a" }}</dd>
      </div>
      <div
        v-if="subscriptionStore.canTrial && canManageSubscription"
        class="my-3"
      >
        <dt class="text-gray-400">
          {{ $t("subscription.try-for-free") }}
        </dt>

        <dd class="mt-1">
          <button
            type="button"
            class="btn-primary inline-flex justify-center ml-auto"
            @click="state.showTrialModal = true"
          >
            {{
              $t("subscription.enterprise-free-trial", {
                days: subscriptionStore.trialingDays,
              })
            }}
          </button>
        </dd>
      </div>
      <div
        v-if="
          subscriptionStore.isTrialing &&
          subscriptionStore.currentPlan == PlanType.ENTERPRISE
        "
        class="my-3"
      >
        <dt class="text-gray-400">
          {{ $t("subscription.inquire-enterprise-plan") }}
        </dt>

        <dd class="mt-1">
          <a
            type="button"
            class="btn-primary inline-flex justify-center ml-auto"
            target="_blank"
            href="https://www.bytebase.com/contact-us"
          >
            {{ $t("subscription.contact-us") }}
          </a>
        </dd>
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

  <InstanceAssignment
    :show="state.showInstanceAssignmentDrawer"
    @dismiss="state.showInstanceAssignmentDrawer = false"
  />
</template>

<script lang="ts" setup>
import { computed, reactive, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import PricingTable from "../components/PricingTable/";
import { PlanType } from "@/types/proto/v1/subscription_service";
import {
  pushNotification,
  useCurrentUserV1,
  useInstanceV1Store,
  useSubscriptionV1Store,
} from "@/store";
import { storeToRefs } from "pinia";
import { hasWorkspacePermissionV1 } from "@/utils";

interface LocalState {
  loading: boolean;
  license: string;
  showTrialModal: boolean;
  showInstanceAssignmentDrawer: boolean;
}

const subscriptionStore = useSubscriptionV1Store();
const instanceV1Store = useInstanceV1Store();
const { t } = useI18n();
const currentUserV1 = useCurrentUserV1();

const state = reactive<LocalState>({
  loading: false,
  license: "",
  showTrialModal: false,
  showInstanceAssignmentDrawer: false,
});

onMounted(() => {
  const params = new URLSearchParams(window.location.search);
  if (params.get("manageLicense")) {
    state.showInstanceAssignmentDrawer = true;
  }
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

const { expireAt, isTrialing, isExpired, instanceLicenseCount } =
  storeToRefs(subscriptionStore);

const totalLicenseCount = computed((): string => {
  if (instanceLicenseCount.value === Number.MAX_VALUE) {
    return t("subscription.unlimited");
  }
  return `${instanceLicenseCount.value}`;
});

const activateLicenseCount = computed((): string => {
  return `${instanceV1Store.activateInstanceCount}`;
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

const openTrialModal = () => {
  state.showTrialModal = true;
};

const canManageSubscription = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-subscription",
    currentUserV1.value.userRole
  );
});
</script>
