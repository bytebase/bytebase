<template>
  <div class="hidden md:block">
    <table id="plans" class="w-full h-px table-fixed mb-16">
      <caption class="sr-only">
        Pricing plan comparison
      </caption>
      <thead>
        <tr>
          <th
            class="py-8 px-6 text-sm font-medium text-gray-900 text-left align-top"
            scope="row"
          ></th>
          <td
            v-for="plan in plans"
            :key="plan.type"
            class="h-full py-8 px-6 align-top"
          >
            <div class="flex-1">
              <img :src="plan.image" class="hidden lg:block p-5" />
              <h3 class="text-2xl font-semibold text-gray-900">
                {{ plan.title }}
              </h3>
              <p class="text-gray-500 mb-10">{{ plan.description }}</p>

              <p class="mt-4 flex items-baseline text-gray-900">
                <span class="text-4xl">
                  {{ plan.price }}
                </span>
              </p>

              <div
                :class="[
                  isAvailableToPurchase(plan) ? '' : 'opacity-0 disabled',
                  'flex justify-center items-center mt-5',
                ]"
              >
                <div>
                  Instances<br />
                  ${{ instancePricePerMonth }}/instance/month
                </div>
                <Counter
                  class="ml-auto"
                  :count="state.instanceCount"
                  :minimum="minimumInstanceCount"
                  @on-change="(val) => (state.instanceCount = val)"
                />
              </div>

              <button
                type="button"
                :class="[
                  plan.highlight
                    ? 'border-indigo-500  text-white  bg-indigo-500 hover:bg-indigo-600 hover:border-indigo-600'
                    : 'border-accent text-accent hover:bg-accent',
                  'mt-8 block w-full border rounded-md py-2 lg:py-4 text-sm lg:text-xl font-semibold text-center hover:text-white whitespace-nowrap overflow-hidden',
                ]"
                @click="onButtonClick(plan)"
              >
                {{ plan.buttonText }}
              </button>
            </div>
          </td>
        </tr>
      </thead>
      <tbody class="border-t border-gray-200 divide-y divide-gray-200">
        <template v-for="section in sections" :key="section.id">
          <tr>
            <th
              class="bg-gray-50 py-3 pl-6 text-sm font-medium text-gray-900 text-left"
              colspan="4"
              scope="colgroup"
            >
              {{ section.id }}
            </th>
          </tr>
          <tr
            v-for="feature in section.features"
            :key="feature"
            class="hover:bg-gray-50"
          >
            <th
              class="py-5 px-6 text-sm font-normal text-gray-500 text-left"
              scope="row"
            >
              {{ feature }}
            </th>
            <td v-for="plan in plans" :key="plan.type" class="py-5 px-6">
              <template v-if="getFeature(plan, feature)">
                <span
                  v-if="getFeature(plan, feature)?.content"
                  class="block text-sm text-gray-700"
                  >{{ getFeature(plan, feature)?.content }}</span
                >
                <heroicons-solid:check v-else class="w-5 h-5 text-green-500" />
              </template>
              <template v-else>
                <heroicons-solid:minus class="w-5 h-5 text-gray-500" />
              </template>
            </td>
          </tr>
        </template>
      </tbody>
      <tfoot>
        <tr class="border-t border-gray-200">
          <th class="sr-only" scope="row">Choose your plan</th>
          <td v-for="plan in plans" :key="plan.type" class="pt-5 px-6">
            <a
              v-if="!plan.isFreePlan"
              href="https://hub.bytebase.com"
              target="_blank"
              class="block w-full py-4 bg-gray-800 border border-gray-800 rounded-md py-2 text-sm font-semibold text-white text-center hover:bg-gray-900"
            >
              Buy {{ plan.title }} Plan
            </a>
          </td>
        </tr>
      </tfoot>
    </table>
  </div>
</template>

<script lang="ts">
import { reactive, computed, watch, PropType } from "vue";
import {
  Plan,
  Subscription,
  PlanType,
  FEATURE_SECTIONS,
  FREE_PLAN,
  TEAM_PLAN,
  ENTERPRISE_PLAN,
} from "../types";

interface LocalState {
  isMonthly: boolean;
  instanceCount: number;
}

interface LocalPlan extends Plan {
  image: string;
  price: string;
  buttonText: string;
  highlight: boolean;
  isFreePlan: boolean;
}

const minimumInstanceCount = 5;
const instancePricePerMonth = 29;

export default {
  name: "PricingTable",
  props: {
    subscription: {
      required: false,
      default: undefined,
      type: Object as PropType<Subscription>,
    },
  },
  setup(props) {
    const state = reactive<LocalState>({
      isMonthly: false,
      instanceCount: props.subscription?.instanceCount ?? minimumInstanceCount,
    });

    watch(
      () => props.subscription,
      (val) =>
        (state.instanceCount = val?.instanceCount ?? minimumInstanceCount)
    );

    const instancePricePerYear = computed((): number => {
      return (
        (state.instanceCount - minimumInstanceCount) *
        instancePricePerMonth *
        12
      );
    });

    const getPlanPrice = (plan: Plan): number => {
      if (plan.type !== PlanType.TEAM) return plan.unitPrice;
      return plan.unitPrice + instancePricePerYear.value;
    };

    const plans = computed((): LocalPlan[] => {
      return [FREE_PLAN, TEAM_PLAN, ENTERPRISE_PLAN].map((plan) => ({
        ...plan,
        image: new URL(
          `../assets/plan-${plan.title.toLowerCase()}.png`,
          import.meta.url
        ).href,
        price:
          plan.type === PlanType.ENTERPRISE
            ? "Contact us"
            : `$${getPlanPrice(plan)}/year`,
        buttonText: getButtonText(plan),
        highlight: plan.type === PlanType.TEAM,
        isFreePlan: plan.type === PlanType.FREE,
      }));
    });

    const getFeature = (plan: Plan, feature: string) => {
      return plan.features.find((f) => f.id === feature);
    };

    const getButtonText = (plan: Plan): string => {
      if (plan.type === PlanType.FREE) return "Deploy";
      if (plan.type === PlanType.ENTERPRISE) return "Contact us";
      if (plan.type === props.subscription?.plan) return "Current plan";
      if (plan.trialDays && plan.trialPrice) {
        return `Start trial with $${plan.trialPrice} for ${plan.trialDays} days`;
      }
      return "Subscribe now";
    };

    const onButtonClick = (plan: Plan) => {
      if (plan.type === PlanType.TEAM) {
        window.open("https://hub.bytebase.com/", "__blank");
      } else if (plan.type === PlanType.ENTERPRISE) {
        window.open(
          "mailto:support@bytebase.com?subject=Request for enterprise plan"
        );
      } else {
        window.open("https://docs.bytebase.com/", "__blank");
      }
    };

    const isAvailableToPurchase = (plan: Plan): boolean => {
      return plan.type === PlanType.TEAM;
    };

    return {
      state,
      plans,
      sections: FEATURE_SECTIONS,
      getFeature,
      onButtonClick,
      minimumInstanceCount,
      instancePricePerMonth,
      isAvailableToPurchase,
    };
  },
};
</script>
