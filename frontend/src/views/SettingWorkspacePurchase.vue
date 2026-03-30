<template>
  <div class="w-full">
    <!-- Pending Payment Overlay -->
    <div
      v-if="state.pendingPayment"
      class="flex flex-col items-center justify-center py-16 gap-y-4"
    >
      <NSpin size="large" />
      <div class="text-lg font-medium">{{ $t("subscription.purchase.pending-title") }}</div>
      <div class="text-sm text-gray-500">{{ $t("subscription.purchase.pending-description") }}</div>
    </div>

    <template v-else>
      <template v-if="!subscriptionStore.isFreePlan && !subscriptionStore.isExpired">
        <!-- Active Subscription Management -->
        <div v-if="!subscriptionStore.isFreePlan && !subscriptionStore.isExpired">
          <div v-if="subscriptionStore.paymentInfo" class="space-y-2">
            <div class="text-lg font-medium">{{ $t("subscription.purchase.payment-info") }}</div>
            <div class="grid grid-cols-2 gap-4 text-sm">
              <div>
                <span class="text-gray-500">{{ $t("subscription.purchase.total-price") }}</span>
                <div class="flex items-center gap-x-2">
                  <div class="font-medium">
                    ${{ (Number(subscriptionStore.paymentInfo.totalPrice) / 100).toFixed(2) }}
                    {{ subscriptionStore.paymentInfo.currency.toUpperCase() }}
                  </div>
                  <NButton
                    v-if="subscriptionStore.paymentInfo.invoiceUrl"
                    text
                    type="primary"
                    tag="a"
                    :href="subscriptionStore.paymentInfo.invoiceUrl"
                    target="_blank"
                  >
                    {{ $t("subscription.purchase.manage-invoices") }}
                  </NButton>
                </div>
              </div>
              <div>
                <span class="text-gray-500">{{ $t("subscription.purchase.billing-period") }}</span>
                <div class="font-medium">
                  {{ subscriptionStore.paymentInfo.periodStart }} - {{ subscriptionStore.paymentInfo.periodEnd }}
                </div>
              </div>
            </div>
          </div>
          <!-- Cancellation pending (annual plan canceled, waiting for period end) -->
          <BBAttention v-if="subscriptionStore.paymentInfo?.cancelAtPeriodEnd" class="my-2" :type="'warning'">
            {{ $t("subscription.purchase.cancel-pending", { date: subscriptionStore.paymentInfo.periodEnd }) }}
          </BBAttention>
          <!-- Cancel button -->
          <div v-else-if="allowManage" class="mt-4">
            <NButton type="error" secondary :loading="state.canceling" :disabled="state.canceling" @click="handleCancel">
              {{ $t("subscription.purchase.cancel") }}
            </NButton>
          </div>
        </div>
        <NDivider />
      </template>

      <template v-if="planCards.length > 0">
        <!-- Plan Cards Grid -->
        <div class="grid gap-8 grid-cols-1 lg:grid-cols-3">
          <PlanCard
            v-for="card in planCards"
            :key="card.type"
            :title="card.title"
            :description="card.description"
            :features="card.features"
            :highlighted="isCurrentPlan(card.type)"
          >
            <template #pricing>
              <!-- Free -->
              <span v-if="card.type === PlanType.FREE" class="text-4xl font-extrabold">$0</span>
              <!-- Self-service plans: show unit price from API -->
              <template v-else-if="card.unitPrice > 0">
                <span class="text-4xl font-extrabold">${{ card.unitPrice / 100 }}</span>
                <span class="ml-2 text-gray-500">{{ $t("subscription.purchase.per-user-per-month") }}</span>
              </template>
              <!-- Enterprise / custom -->
              <span v-else class="text-3xl font-extrabold">{{ $t("subscription.purchase.custom") }}</span>
            </template>

            <template v-if="allowManage" #config>
              <!-- User counter for self-service plans with USER additional -->
              <div v-if="card.userAdditional" class="flex items-center">
                <span class="tracking-tight text-gray-900">{{ $t("subscription.purchase.seats") }}</span>
                <div class="ml-auto custom-number-input h-7 w-20">
                  <div class="flex flex-row h-full w-full rounded-lg relative bg-gray-200 text-gray-600 items-center">
                    <button
                      class="hover:text-gray-700 hover:bg-gray-300 h-full w-10 rounded-l cursor-pointer outline-none"
                      :disabled="state.seats <= (card.userAdditional.minimumCount || 1)"
                      @click="state.seats = Math.max(card.userAdditional.minimumCount || 1, state.seats - 1)"
                    >
                      <span class="m-auto font-thin">−</span>
                    </button>
                    <div class="w-full text-center text-sm">{{ state.seats }}</div>
                    <button
                      class="hover:text-gray-700 hover:bg-gray-300 h-full w-10 rounded-r cursor-pointer outline-none"
                      :disabled="card.userAdditional.maximumCount > 0 && state.seats >= card.userAdditional.maximumCount"
                      @click="state.seats++"
                    >
                      <span class="m-auto font-thin">+</span>
                    </button>
                  </div>
                </div>
              </div>
            </template>

            <template v-if="allowManage" #action>
              <!-- Free: Go to workspace -->
              <div v-if="card.type === PlanType.FREE" class="mt-3 w-full">
                <NButton size="large" type="primary" secondary block :disabled="isCurrentPlan(PlanType.FREE)">
                  {{ isCurrentPlan(PlanType.FREE) ? $t("subscription.current") : $t("subscription.purchase.go-to-workspace") }}
                </NButton>
              </div>

              <!-- Self-service: Terms + Subscribe -->
              <template v-else-if="card.selfServicePurchase">
                <div class="mt-3 flex items-start gap-x-2 text-sm text-gray-500">
                  <NCheckbox :checked="state.checkPolicy" @update:checked="(val: boolean) => (state.checkPolicy = val)" />
                  <span>
                    {{ $t("subscription.purchase.accept-terms-prefix") }}
                    <a href="https://www.bytebase.com/terms" target="_blank" class="underline hover:text-gray-700">{{ $t("subscription.purchase.terms-of-service") }}</a>
                    {{ $t("subscription.purchase.and") }}
                    <a href="https://www.bytebase.com/privacy" target="_blank" class="underline hover:text-gray-700">{{ $t("subscription.purchase.privacy-policy") }}</a>
                  </span>
                </div>
                <div class="mt-3 w-full">
                  <NButton
                    size="large"
                    type="primary"
                    block
                    :loading="state.loading"
                    :disabled="purchaseButtonDisabled(card)"
                    @click="handlePurchase(card)"
                  >
                    <span class="text-sm! font-semibold!">{{ purchaseButtonText(card) }}</span>
                  </NButton>
                </div>
              </template>

              <!-- Enterprise: Contact us -->
              <div v-else class="mt-3 w-full">
                <RequireEnterpriseButton size="large" type="primary" secondary block>
                  {{ $t("subscription.contact-us") }}
                </RequireEnterpriseButton>
              </div>
            </template>
          </PlanCard>
        </div>

        <!-- Footer links -->
        <div class="pt-4 pb-2 text-center">
          <a
            class="text-sm text-gray-500 hover:text-gray-700 underline"
            href="https://www.bytebase.com/pricing?source=console"
            target="_blank"
          >
            {{ $t("subscription.purchase.see-comparison") }}
          </a>
        </div>
      </template>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { NButton, NCheckbox, NDivider, NSpin, useDialog } from "naive-ui";
import { computed, onMounted, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { BBAttention } from "@/bbkit";
import type { PlanFeatureItem } from "@/components/PlanCard.vue";
import PlanCard from "@/components/PlanCard.vue";
import RequireEnterpriseButton from "@/components/RequireEnterpriseButton.vue";
import { pushNotification, useSubscriptionV1Store } from "@/store";
import type { PurchasePlanAdditional } from "@/types/proto-es/v1/subscription_service_pb";
import {
  BillingInterval,
  PlanType,
  PurchaseDiscount_Type,
  PurchasePlanAdditional_Type,
} from "@/types/proto-es/v1/subscription_service_pb";

const props = defineProps<{
  allowManage: boolean;
}>();

const { t } = useI18n();
const $dialog = useDialog();
const route = useRoute();
const router = useRouter();
const subscriptionStore = useSubscriptionV1Store();

interface LocalState {
  seats: number;
  loading: boolean;
  canceling: boolean;
  checkPolicy: boolean;
  pendingPayment: boolean;
}

const state = reactive<LocalState>({
  seats: 1,
  loading: false,
  canceling: false,
  checkPolicy: false,
  pendingPayment: false,
});

// Sync state from subscription when it changes (e.g., after fetch or webhook update).
watchEffect(() => {
  const sub = subscriptionStore.subscription;
  if (sub && !subscriptionStore.isFreePlan && !subscriptionStore.isExpired) {
    state.seats = Math.max(1, sub.seats);
    state.checkPolicy = true;
  }
});

const isCurrentPlan = (plan: PlanType): boolean => {
  return subscriptionStore.currentPlan === plan && !subscriptionStore.isExpired;
};

// Feature keys for i18n (display-only, not from API).
const featureKeys: Record<number, string[]> = {
  [PlanType.FREE]: [
    "limits",
    "dcm",
    "git",
    "sql-review",
    "backup",
    "sql-editor",
    "schema",
    "iam",
    "support",
  ],
  [PlanType.TEAM]: [
    "limits",
    "everything",
    "batch-query",
    "read-only",
    "query-policy",
    "sso",
    "groups",
    "db-groups",
    "audit-log",
    "support",
  ],
  [PlanType.ENTERPRISE]: [
    "limits",
    "everything",
    "custom-limits",
    "approval",
    "audit-log",
    "sso",
    "2fa",
    "masking",
    "roles",
    "scim",
    "secret",
    "branding",
    "support",
  ],
};

const planPrefix: Record<number, string> = {
  [PlanType.FREE]: "free",
  [PlanType.TEAM]: "pro",
  [PlanType.ENTERPRISE]: "enterprise",
};

const planDescription = (type: PlanType): string => {
  switch (type) {
    case PlanType.FREE:
      return t("subscription.purchase.free-description");
    case PlanType.TEAM:
      return t("subscription.purchase.pro-description");
    case PlanType.ENTERPRISE:
      return t("subscription.purchase.enterprise-description");
    default:
      return "";
  }
};

const planTitle = (type: PlanType): string => {
  switch (type) {
    case PlanType.FREE:
      return t("subscription.plan.free.title");
    case PlanType.TEAM:
      return t("subscription.plan.team.title");
    case PlanType.ENTERPRISE:
      return t("subscription.plan.enterprise.title");
    default:
      return "";
  }
};

const getFeaturesForPlan = (type: PlanType): PlanFeatureItem[] =>
  (featureKeys[type] || []).map((key: string) => ({
    text: t(
      `dynamic.subscription.purchase.features.${planPrefix[type]}.${key}`
    ),
    bold: key === "everything",
  }));

interface PlanCardData {
  type: PlanType;
  title: string;
  description: string;
  features: PlanFeatureItem[];
  selfServicePurchase: boolean;
  unitPrice: number; // cents per user per month
  userAdditional: PurchasePlanAdditional | undefined;
  billingIntervals: BillingInterval[];
  discountDescription?: string; // from billing method discount (e.g., "first month 90% off")
}

const planCards = computed((): PlanCardData[] => {
  const apiPlans = subscriptionStore.purchasePlans;
  if (apiPlans.length === 0) {
    return [];
  }

  // Always show FREE (not from API).
  const cards: PlanCardData[] = [
    {
      type: PlanType.FREE,
      title: planTitle(PlanType.FREE),
      description: planDescription(PlanType.FREE),
      features: getFeaturesForPlan(PlanType.FREE),
      selfServicePurchase: false,
      unitPrice: 0,
      userAdditional: undefined,
      billingIntervals: [],
    },
  ];

  // Add API-driven plans.
  for (const plan of apiPlans) {
    const userAdditional = plan.additionals.find(
      (a) => a.type === PurchasePlanAdditional_Type.USER
    );
    // Find the first billing method with a discount description.
    const discount = plan.billingMethods.find((bm) => bm.discount)?.discount;
    let discountDescription = "";
    if (discount) {
      switch (discount.type) {
        case PurchaseDiscount_Type.PERCENTAGE_OFF:
          discountDescription = t(
            "subscription.purchase.discount.percentage-off",
            { value: discount.value }
          );
          break;
        case PurchaseDiscount_Type.FIXED_MONTH_OFF:
          discountDescription = t("subscription.purchase.discount.month-off", {
            value: discount.value,
          });
          break;
        case PurchaseDiscount_Type.FIXED_PRICE_OFF:
          discountDescription = t("subscription.purchase.discount.price-off", {
            value: discount.value,
          });
          break;
        default:
          break;
      }
    }
    cards.push({
      type: plan.type,
      title: planTitle(plan.type),
      description: planDescription(plan.type),
      features: getFeaturesForPlan(plan.type),
      selfServicePurchase: plan.selfServicePurchase,
      unitPrice: userAdditional?.unitPrice ?? 0,
      userAdditional,
      billingIntervals: plan.billingMethods.map((bm) => bm.interval),
      discountDescription: discountDescription,
    });
  }

  return cards;
});

const isPlanConfigChanged = (card: PlanCardData): boolean => {
  if (!isCurrentPlan(card.type)) return false;
  const currentSeats = subscriptionStore.subscription?.seats ?? 0;
  return state.seats !== currentSeats;
};

const purchaseButtonDisabled = (card: PlanCardData): boolean => {
  if (state.loading) return true;
  if (!state.checkPolicy) return true;
  // Current plan with no config change — disabled.
  if (isCurrentPlan(card.type) && !isPlanConfigChanged(card)) return true;
  return false;
};

const purchaseButtonText = (card: PlanCardData): string => {
  if (isPlanConfigChanged(card)) {
    return t("subscription.purchase.update");
  }
  if (isCurrentPlan(card.type)) {
    return t("subscription.current");
  }
  const label = t("subscription.purchase.subscribe");
  if (card.discountDescription) {
    return `${label} (${card.discountDescription})`;
  }
  return label;
};

const handlePurchase = async (card: PlanCardData) => {
  if (purchaseButtonDisabled(card)) return;

  const interval = card.billingIntervals[0] ?? BillingInterval.MONTH;
  state.loading = true;
  try {
    let paymentUrl: string;
    if (isPlanConfigChanged(card)) {
      // Update existing subscription.
      paymentUrl = await subscriptionStore.updatePurchase(
        card.type,
        interval,
        state.seats,
        "" // etag is optional
      );
    } else {
      // New purchase.
      paymentUrl = await subscriptionStore.createPurchase(
        card.type,
        interval,
        state.seats
      );
    }
    if (paymentUrl) {
      window.location.href = paymentUrl;
    } else {
      // Direct update succeeded — old subscription canceled, new one created.
      // Poll without updating the store to avoid UI flashing (PAID → FREE → PAID).
      state.pendingPayment = true;
      const maxAttempts = 30;
      for (let i = 0; i < maxAttempts; i++) {
        const sub = await subscriptionStore.fetchSubscription(false);
        if (sub && sub.plan !== PlanType.FREE && sub.seats === state.seats) {
          // New subscription confirmed — now update the store.
          subscriptionStore.setSubscription(sub);
          break;
        }
        await new Promise((resolve) => setTimeout(resolve, 2000));
      }
      state.pendingPayment = false;
    }
  } catch (e) {
    console.error(e);
  } finally {
    state.loading = false;
  }
};

const handleCancel = () => {
  $dialog.warning({
    title: t("subscription.purchase.cancel"),
    content: t("subscription.purchase.cancel-confirm"),
    positiveText: t("subscription.purchase.cancel"),
    negativeText: t("common.cancel"),
    onPositiveClick: async () => {
      state.canceling = true;
      try {
        await subscriptionStore.cancelPurchase();
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("subscription.purchase.canceled"),
        });
      } catch (e) {
        console.error(e);
      } finally {
        state.canceling = false;
      }
    },
  });
};

onMounted(async () => {
  // Fetch plans from API.
  await subscriptionStore.fetchPurchasePlans();

  // Poll for subscription activation after Stripe Checkout redirect.
  const sessionId = route.query.session_id as string | undefined;
  if (sessionId && subscriptionStore.isFreePlan && props.allowManage) {
    // Verify the session with Stripe before polling.
    try {
      const status = await subscriptionStore.verifyCheckoutSession(sessionId);
      if (status === "complete") {
        state.pendingPayment = true;
        const maxAttempts = 30;
        for (let i = 0; i < maxAttempts; i++) {
          await subscriptionStore.fetchSubscription();
          if (!subscriptionStore.isFreePlan) {
            break;
          }
          await new Promise((resolve) => setTimeout(resolve, 2000));
        }
        state.pendingPayment = false;
      }
    } catch (e) {
      console.error("failed to verify checkout session", e);
    }
  }
  // Clean up query params to prevent re-polling on refresh.
  router.replace({ query: {} });

  // Fetch payment info for active subscriptions.
  if (
    !subscriptionStore.isFreePlan &&
    !subscriptionStore.isExpired &&
    props.allowManage
  ) {
    await subscriptionStore.fetchPaymentInfo();
  }
});
</script>
