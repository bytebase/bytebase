<template>
  <div class="hidden xl:block">
    <table id="plans" class="w-full h-px table-fixed">
      <caption class="sr-only">
        Pricing plan comparison
      </caption>
      <thead>
        <tr>
          <th
            class="py-8 px-6 text-sm font-medium text-gray-900 text-left align-top hidden 2xl:block"
            scope="row"
          ></th>
          <td
            v-for="plan in plans"
            :key="plan.type"
            class="h-full pt-8 px-6 align-top"
          >
            <div class="flex-1">
              <img :src="plan.image" class="hidden lg:block p-5" />

              <div class="flex flex-row items-center h-10">
                <h3 class="text-xl font-semibold text-gray-900">
                  {{ $t(`subscription.plan.${plan.title}.title`) }}
                </h3>
                <span
                  v-if="plan.label"
                  class="ml-2 inline-flex items-center px-3 py-0.5 rounded-full text-sm font-sm bg-indigo-100 text-indigo-800"
                >
                  {{ plan.label }}
                </span>
              </div>

              <p class="text-gray-500 mb-10 h-12">
                {{ $t(`subscription.plan.${plan.title}.desc`) }}
              </p>

              <p
                class="mt-4 flex items-baseline text-gray-900 text-xl space-x-2"
              >
                <span v-if="plan.pricePrefix" class="text-3xl">
                  {{ plan.pricePrefix }}
                </span>
                <span
                  :class="[
                    'font-bold',
                    plan.type == PlanType.ENTERPRISE ? 'text-3xl' : 'text-4xl',
                  ]"
                >
                  {{ plan.pricing }}
                </span>
                {{ plan.priceSuffix }}
              </p>

              <div
                :class="[
                  'mt-2 text-gray-600 h-12',
                  plan.type == PlanType.TEAM ? 'font-bold' : '',
                ]"
              >
                {{ $t(`subscription.${plan.title}-price-intro`) }}
              </div>

              <button
                v-if="plan.buttonText"
                type="button"
                :class="[
                  plan.highlight
                    ? 'border-indigo-500  text-white  bg-indigo-500 hover:bg-indigo-600 hover:border-indigo-600'
                    : 'border-accent text-accent hover:bg-accent',
                  'mt-8 block w-full border rounded-md py-2 lg:py-4 text-sm md:text-xl font-semibold text-center hover:text-white whitespace-nowrap overflow-hidden',
                ]"
                @click="onButtonClick(plan)"
              >
                {{ plan.buttonText }}
              </button>
              <div
                v-if="plan.canTrial && subscriptionStore.canTrial"
                class="font-bold text-sm my-2 text-center underline cursor-pointer"
                @click="emit('on-trial')"
              >
                {{ $t("subscription.start-free-trial") }}
              </div>
            </div>
          </td>
        </tr>
      </thead>
    </table>
    <div class="px-4 py-8 text-right text-gray-500">
      <i18n-t keypath="subscription.announcement">
        <template #cancel>
          <a
            class="underline"
            href="https://bytebase.com/refund?source=console"
            target="_blank"
            >{{ $t("subscription.cancel") }}</a
          >
        </template>
      </i18n-t>
    </div>
    <table class="w-full table-fixed mb-16 border-l border-r border-b">
      <caption class="sr-only">
        Feature comparison
      </caption>
      <tbody class="border-t border-gray-200 divide-y divide-gray-200">
        <template v-for="section in FEATURE_SECTIONS" :key="section.type">
          <tr>
            <th
              class="bg-gray-50 py-3 pl-6 text-base font-medium text-gray-900 text-left"
              colspan="4"
              scope="colgroup"
            >
              {{ $t(`subscription.feature-sections.${section.type}.title`) }}
            </th>
          </tr>
          <tr
            v-for="feature in section.featureList"
            :key="feature"
            class="hover:bg-gray-50"
          >
            <th
              class="py-5 px-6 text-sm font-normal text-gray-500 text-left"
              scope="row"
            >
              {{
                $t(
                  `subscription.feature-sections.${section.type}.features.${feature}`
                )
              }}
            </th>
            <FeatureItem
              v-for="plan in plans"
              :key="plan.type"
              :plan="plan"
              :feature="feature"
            />
          </tr>
        </template>
      </tbody>
      <tfoot>
        <tr class="border-t border-gray-200">
          <th class="sr-only" scope="row">Choose your plan</th>
          <td v-for="plan in plans" :key="plan.type" class="py-5 px-6">
            <button
              v-if="plan.buttonText"
              class="block w-full bg-gray-800 border border-gray-800 rounded-md py-2 text-lg font-semibold text-white text-center hover:bg-gray-900"
              @click="onButtonClick(plan)"
            >
              {{ plan.buttonText }}
            </button>
          </td>
        </tr>
      </tfoot>
    </table>
  </div>
  <div class="xl:hidden">
    <div v-for="plan in plans" :key="plan.type" class="mb-16">
      <div class="flex flex-col items-center">
        <img :src="plan.image" class="block w-2/4" />

        <div class="flex flex-row items-center">
          <h3 class="text-2xl font-semibold text-gray-900">
            {{ $t(`subscription.plan.${plan.title}.title`) }}
          </h3>
          <span
            v-if="plan.label"
            class="ml-2 inline-flex items-center px-3 py-0.5 rounded-full text-sm font-sm bg-indigo-100 text-indigo-800"
          >
            {{ plan.label }}
          </span>
        </div>

        <p class="text-gray-500">
          {{ $t(`subscription.plan.${plan.title}.desc`) }}
        </p>

        <p class="mt-4 flex items-baseline text-gray-900 text-xl space-x-2">
          <span v-if="plan.pricePrefix" class="text-3xl">
            {{ plan.pricePrefix }}
          </span>
          <span
            :class="[
              'font-bold',
              plan.type == PlanType.ENTERPRISE ? 'text-3xl' : 'text-4xl',
            ]"
          >
            {{ plan.pricing }}
          </span>
          {{ plan.priceSuffix }}
        </p>

        <div
          :class="[
            'text-gray-600',
            plan.type == PlanType.TEAM ? 'font-bold' : '',
          ]"
        >
          {{ $t(`subscription.${plan.title}-price-intro`) }}
        </div>

        <button
          v-if="plan.buttonText"
          type="button"
          :class="[
            plan.highlight
              ? 'border-indigo-500  text-white  bg-indigo-500 hover:bg-indigo-600 hover:border-indigo-600'
              : 'border-accent text-accent hover:bg-accent',
            'mt-8 block w-full border rounded-md py-4 font-semibold text-center text-xl hover:text-white whitespace-nowrap overflow-hidden',
          ]"
          @click="onButtonClick(plan)"
        >
          {{ plan.buttonText }}
        </button>
        <div
          v-if="plan.canTrial && subscriptionStore.canTrial"
          class="font-bold text-sm my-2 text-center underline cursor-pointer"
          @click="emit('on-trial')"
        >
          {{ $t("subscription.start-free-trial") }}
        </div>

        <div v-if="plan.isAvailable" class="px-4 py-8 text-right text-gray-500">
          <i18n-t keypath="subscription.announcement">
            <template #cancel>
              <a
                class="underline"
                href="https://bytebase.com/refund?source=console"
                target="_blank"
                >{{ $t("subscription.cancel") }}</a
              >
            </template>
          </i18n-t>
        </div>
      </div>
      <table class="w-full h-px table-fixed mt-10">
        <caption class="sr-only">
          Feature comparison
        </caption>
        <tbody class="border border-gray-200 divide-y divide-gray-200">
          <template v-for="section in FEATURE_SECTIONS" :key="section.type">
            <tr>
              <th
                class="bg-gray-50 py-3 pl-6 text-base font-medium text-gray-900 text-left"
                scope="colgroup"
              >
                {{ $t(`subscription.feature-sections.${section.type}.title`) }}
              </th>
            </tr>
            <tr
              v-for="feature in section.featureList"
              :key="feature"
              class="hover:bg-gray-50"
            >
              <th
                class="py-5 px-6 text-sm font-normal text-gray-500 text-left"
                scope="row"
              >
                {{
                  $t(
                    `subscription.feature-sections.${section.type}.features.${feature}`
                  )
                }}
              </th>
              <FeatureItem :plan="plan" :feature="feature" class="w-3/4" />
            </tr>
          </template>
        </tbody>
      </table>
      <button
        v-if="plan.buttonText"
        type="button"
        :class="[
          plan.highlight
            ? 'border-indigo-500  text-white  bg-indigo-500 hover:bg-indigo-600 hover:border-indigo-600'
            : 'border-accent text-accent hover:bg-accent',
          'mt-8 block w-full border rounded-md py-4 text-lg font-semibold text-center hover:text-white whitespace-nowrap overflow-hidden',
        ]"
        @click="onButtonClick(plan)"
      >
        {{ plan.buttonText }}
      </button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { reactive, computed, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useSubscriptionV1Store } from "@/store";
import { Plan, PLANS, FEATURE_SECTIONS } from "@/types";
import { PlanType } from "@/types/proto/v1/subscription_service";
import FeatureItem from "./FeatureItem.vue";
import { LocalPlan } from "./types";

interface LocalState {
  isMonthly: boolean;
  instanceCount: number;
}

const emit = defineEmits(["on-trial"]);

const minimumInstanceCount = 5;

const { t } = useI18n();
const subscriptionStore = useSubscriptionV1Store();
const state = reactive<LocalState>({
  isMonthly: false,
  instanceCount:
    subscriptionStore.subscription?.instanceCount ?? minimumInstanceCount,
});
const enterprisePlanFormLink = "https://www.bytebase.com/contact-us";

watch(
  () => subscriptionStore.subscription,
  (val) => (state.instanceCount = val?.instanceCount ?? minimumInstanceCount)
);

const plans = computed((): LocalPlan[] => {
  return PLANS.map((plan) => ({
    ...plan,
    image: new URL(
      `../../assets/plan-${plan.title.toLowerCase()}.png`,
      import.meta.url
    ).href,
    buttonText: getButtonText(plan),
    highlight: plan.type === PlanType.TEAM,
    isAvailable: plan.type === PlanType.TEAM,
    label: t(`subscription.plan.${plan.title}.label`),
    pricing:
      plan.type === PlanType.ENTERPRISE
        ? t("subscription.contact-us")
        : `${plan.pricePerInstancePerMonth}`,
    pricePrefix: "",
    priceSuffix:
      plan.type === PlanType.TEAM
        ? t("subscription.price-unit-for-team")
        : plan.type === PlanType.ENTERPRISE
        ? ""
        : t("subscription.per-month"),
    // only support free trial for enterprise in console.
    canTrial: plan.type === PlanType.ENTERPRISE && plan.trialDays > 0,
  }));
});

const getButtonText = (plan: Plan): string => {
  switch (plan.type) {
    case PlanType.FREE:
      return "";
    case PlanType.TEAM:
      if (subscriptionStore.currentPlan === PlanType.FREE) {
        return t("subscription.button.upgrade");
      }
      if (subscriptionStore.currentPlan === PlanType.TEAM) {
        return t("subscription.button.view-subscription");
      }
      if (subscriptionStore.currentPlan === PlanType.ENTERPRISE) {
        return "";
      }
      break;
    case PlanType.ENTERPRISE:
      if (subscriptionStore.currentPlan === PlanType.ENTERPRISE) {
        if (subscriptionStore.isTrialing) {
          return t("subscription.button.contact-us");
        }
        return t("subscription.button.view-subscription");
      } else {
        return t("subscription.button.contact-us");
      }
  }
  return "";
};

const onButtonClick = (plan: Plan) => {
  switch (plan.type) {
    case PlanType.TEAM:
      if (subscriptionStore.currentPlan === PlanType.FREE) {
        window.open(
          "https://hub.bytebase.com/workspace?source=console.subscription",
          "__blank"
        );
      } else if (subscriptionStore.currentPlan === PlanType.TEAM) {
        window.open(
          "https://hub.bytebase.com/subscription?source=console.subscription",
          "__blank"
        );
      }
      return;
    case PlanType.ENTERPRISE:
      if (subscriptionStore.currentPlan === PlanType.ENTERPRISE) {
        if (subscriptionStore.isTrialing) {
          window.open(enterprisePlanFormLink, "__blank");
        } else {
          window.open(
            "https://hub.bytebase.com/subscription?source=console.subscription",
            "__blank"
          );
        }
      } else {
        window.open(enterprisePlanFormLink, "__blank");
      }
  }
};
</script>
