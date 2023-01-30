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
                <span v-if="plan.pricePrefix" class="text-4xl">
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
                v-if="plan.trialDays"
                class="font-bold text-sm my-2 text-center"
              >
                {{ $t("subscription.free-trial") }}
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
    <table class="w-full h-px table-fixed mb-16">
      <caption class="sr-only">
        Feature comparison
      </caption>
      <tbody class="border-t border-gray-200 divide-y divide-gray-200">
        <template v-for="section in FEATURE_SECTIONS" :key="section.type">
          <tr>
            <th
              class="bg-gray-50 py-3 pl-6 text-sm font-medium text-gray-900 text-left"
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
          <td v-for="plan in plans" :key="plan.type" class="pt-5 px-6">
            <button
              v-if="!plan.isFreePlan"
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
          <span v-if="plan.pricePrefix" class="text-4xl">
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
        <div v-if="plan.trialDays" class="font-bold text-sm my-2 text-center">
          {{ $t("subscription.free-trial") }}
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
        <tbody class="border-t border-gray-200 divide-y divide-gray-200">
          <template v-for="section in FEATURE_SECTIONS" :key="section.type">
            <tr>
              <th
                class="bg-gray-50 py-3 pl-6 text-sm font-medium text-gray-900 text-left"
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
        v-if="!plan.isFreePlan"
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
import { Plan, PlanType, PLANS, FEATURE_SECTIONS } from "@/types";
import { useSubscriptionStore } from "@/store";
import { LocalPlan } from "./types";
import FeatureItem from "./FeatureItem.vue";

interface LocalState {
  isMonthly: boolean;
  instanceCount: number;
}

const emit = defineEmits(["on-trial"]);

const minimumInstanceCount = 5;

const { t } = useI18n();
const subscriptionStore = useSubscriptionStore();
const state = reactive<LocalState>({
  isMonthly: false,
  instanceCount:
    subscriptionStore.subscription?.instanceCount ?? minimumInstanceCount,
});

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
    isFreePlan: plan.type === PlanType.FREE,
    label: t(`subscription.plan.${plan.title}.label`),
    pricing:
      plan.type === PlanType.ENTERPRISE
        ? t("subscription.contact-us")
        : `$${plan.unitPrice}`,
    pricePrefix:
      plan.type === PlanType.TEAM ? t("subscription.start-from") : "",
    priceSuffix:
      plan.type === PlanType.TEAM
        ? t("subscription.price-unit-for-team")
        : plan.type === PlanType.ENTERPRISE
        ? ""
        : t("subscription.per-month"),
  }));
});

const getButtonText = (plan: Plan): string => {
  if (plan.type === PlanType.FREE) return t("subscription.deploy");
  if (plan.type === PlanType.ENTERPRISE) return t("subscription.contact-us");

  if (subscriptionStore.isTrialing) return t("subscription.subscribe");
  if (plan.type === subscriptionStore.subscription?.plan)
    return t("subscription.upgrade");
  if (plan.trialDays) return t("subscription.start-trial");

  return t("subscription.subscribe");
};

const onButtonClick = (plan: Plan) => {
  if (plan.type === PlanType.TEAM) {
    if (subscriptionStore.canTrial && !subscriptionStore.isTrialing) {
      emit("on-trial");
      return;
    }
    window.open(
      "https://bytebase.com/pricing?source=console.subscription",
      "__blank"
    );
  } else if (plan.type === PlanType.ENTERPRISE) {
    window.open(
      "mailto:support@bytebase.com?subject=Request for enterprise plan"
    );
  } else {
    window.open("https://bytebase.com/docs?source=console", "_self");
  }
};
</script>
