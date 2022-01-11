<template>
  <div class="sm:flex sm:flex-col sm:align-center">
    <div
      class="relative self-center mt-6 bg-gray-100 rounded-lg p-0.5 flex sm:mt-8"
    >
      <button
        type="button"
        class="relative w-1/2 rounded-md py-2 text-sm font-medium text-gray-700 whitespace-nowrap focus:outline-none focus:ring-2 focus:ring-transparent focus:z-10 sm:w-auto sm:px-8"
        :class="state.isMonthly ? '' : 'bg-white border-gray-200 shadow-sm'"
        @click.prevent="state.isMonthly = false"
      >
        Yearly billing
      </button>
      <button
        type="button"
        class="ml-0.5 relative w-1/2 rounded-md py-2 text-sm font-medium text-gray-700 whitespace-nowrap focus:outline-none focus:ring-2 focus:ring-transparent focus:z-10 sm:w-auto sm:px-8"
        :class="state.isMonthly ? 'bg-white border-gray-200 shadow-sm' : ''"
        @click.prevent="state.isMonthly = true"
      >
        Monthly billing
      </button>
    </div>
  </div>
  <div class="mt-12 grid grid-cols-3 gap-6">
    <div
      class="border border-gray-200 rounded-lg shadow-sm divide-y divide-block-border"
    >
      <div class="px-6 pt-6 pb-4">
        <h2 class="text-4xl text-center leading-6 font-bold text-gray-900">
          Free
        </h2>
        <a
          href="#"
          class="mt-8 block w-full border border-accent rounded-md py-4 text-2xl font-semibold text-accent text-center hover:bg-accent hover:text-white"
          >Deploy</a
        >
        <div class="mt-6 flex justify-center flex-row">
          <span class="text-4xl font-extrabold text-gray-900">$0</span>
          <div class="ml-2 flex flex-col">
            <span class="text-sm font-medium text-gray-500 whitespace-nowrap"
              >per user</span
            >
            <span class="text-sm font-medium text-gray-500 whitespace-nowrap"
              >per month</span
            >
          </div>
        </div>
        <div class="mt-8 text-sm text-center">Community support</div>
      </div>
      <div class="pt-6 pb-8 px-6">
        <h3 class="text-xs font-medium text-gray-900 tracking-wide uppercase">
          What's included
        </h3>
        <ul class="mt-6 space-y-4">
          <li class="flex space-x-3">
            <!-- Heroicon name: solid/check -->
            <heroicons-solid:check
              class="flex-shrink-0 h-5 w-5 text-green-500"
            />
            <span class="text-sm text-main whitespace-nowrap"
              >100 issues (excluding archived)</span
            >
          </li>

          <li class="flex space-x-3">
            <!-- Heroicon name: solid/check -->
            <heroicons-solid:check
              class="flex-shrink-0 h-5 w-5 text-green-500"
            />
            <span class="text-sm text-main">Unlimited members</span>
          </li>

          <li class="flex space-x-3">
            <!-- Heroicon name: solid/check -->
            <heroicons-solid:check
              class="flex-shrink-0 h-5 w-5 text-green-500"
            />
            <span class="text-sm text-main">Unlimited environments</span>
          </li>

          <li class="flex space-x-3">
            <!-- Heroicon name: solid/check -->
            <heroicons-solid:check
              class="flex-shrink-0 h-5 w-5 text-green-500"
            />
            <span class="text-sm text-main">Unlimited instances</span>
          </li>

          <li class="flex space-x-3">
            <!-- Heroicon name: solid/check -->
            <heroicons-solid:check
              class="flex-shrink-0 h-5 w-5 text-green-500"
            />
            <span class="text-sm text-main">Unlimited data sources</span>
          </li>

          <li class="flex space-x-3">
            <!-- Heroicon name: solid/check -->
            <heroicons-solid:check
              class="flex-shrink-0 h-5 w-5 text-green-500"
            />
            <span class="text-sm text-main">Slack integration</span>
          </li>

          <li class="flex space-x-3">
            <span class="ml-8 text-sm text-gray-500">DBA and Owner roles</span>
          </li>

          <li class="flex space-x-3">
            <span class="ml-8 text-sm text-gray-500">SSO</span>
          </li>

          <li class="flex space-x-3">
            <span class="ml-8 text-sm text-gray-500">Custom branding</span>
          </li>

          <li class="flex space-x-3">
            <span class="ml-8 text-sm text-gray-500">Audit log</span>
          </li>

          <li class="flex space-x-3">
            <span class="ml-8 text-sm text-gray-500">Roadmap sharing</span>
          </li>
        </ul>
      </div>
    </div>

    <div
      class="border border-gray-200 rounded-lg shadow-sm divide-y divide-block-border"
    >
      <div class="px-6 pt-6 pb-4">
        <h2 class="text-4xl text-center leading-6 font-bold text-gray-900">
          Standard
          <span
            class="ml-1 align-top inline-flex items-center px-1.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800"
          >
            Beta
          </span>
        </h2>
        <a
          href="#"
          class="mt-8 block w-full border border-green-500 rounded-md py-4 text-2xl font-semibold text-white text-center bg-green-500 hover:bg-green-600 hover:border-green-600 hover:text-white whitespace-nowrap"
          >Buy now</a
        >
        <div class="mt-6 flex justify-center flex-row">
          <span class="text-4xl font-medium text-gray-900 line-through">{{
            state.isMonthly ? "$16" : "$12"
          }}</span>
          <span class="ml-1 text-4xl font-extrabold text-gray-900">{{
            state.isMonthly ? "$12" : "$9"
          }}</span>
          <div class="ml-2 flex flex-col">
            <span class="text-sm font-medium text-gray-500 whitespace-nowrap"
              >per user</span
            >
            <span class="text-sm font-medium text-gray-500 whitespace-nowrap"
              >per month</span
            >
          </div>
        </div>

        <div class="mt-8 text-sm text-center">Priority support</div>
      </div>
      <div class="pt-6 pb-8 px-6">
        <h3 class="text-xs font-medium text-gray-900 tracking-wide uppercase">
          What's included
        </h3>
        <ul class="mt-6 space-y-4">
          <li class="flex space-x-3">
            <!-- Heroicon name: solid/check -->
            <heroicons-solid:check
              class="flex-shrink-0 h-5 w-5 text-green-500"
            />
            <span class="text-sm text-main">Unlimited issues</span>
          </li>

          <li class="flex space-x-3">
            <!-- Heroicon name: solid/check -->
            <heroicons-solid:check
              class="flex-shrink-0 h-5 w-5 text-green-500"
            />
            <span class="text-sm text-main">Unlimited members</span>
          </li>

          <li class="flex space-x-3">
            <!-- Heroicon name: solid/check -->
            <heroicons-solid:check
              class="flex-shrink-0 h-5 w-5 text-green-500"
            />
            <span class="text-sm text-main">Unlimited environments</span>
          </li>

          <li class="flex space-x-3">
            <!-- Heroicon name: solid/check -->
            <heroicons-solid:check
              class="flex-shrink-0 h-5 w-5 text-green-500"
            />
            <span class="text-sm text-main">Unlimited instances </span>
          </li>

          <li class="flex space-x-3">
            <!-- Heroicon name: solid/check -->
            <heroicons-solid:check
              class="flex-shrink-0 h-5 w-5 text-green-500"
            />
            <span class="text-sm text-main">Unlimited data sources</span>
          </li>

          <li class="flex space-x-3">
            <!-- Heroicon name: solid/check -->
            <heroicons-solid:check
              class="flex-shrink-0 h-5 w-5 text-green-500"
            />
            <span class="text-sm text-main">Slack integration</span>
          </li>

          <li class="flex space-x-3">
            <!-- Heroicon name: solid/check -->
            <heroicons-solid:check
              class="flex-shrink-0 h-5 w-5 text-green-500"
            />
            <span class="text-sm text-main">DBA and Owner roles</span>
          </li>

          <li class="flex space-x-3">
            <span class="ml-8 text-sm text-gray-500">SSO</span>
          </li>

          <li class="flex space-x-3">
            <span class="ml-8 text-sm text-gray-500">Custom branding</span>
          </li>

          <li class="flex space-x-3">
            <span class="ml-8 text-sm text-gray-500">Audit log</span>
          </li>

          <li class="flex space-x-3">
            <span class="ml-8 text-sm text-gray-500">Roadmap sharing</span>
          </li>
        </ul>
      </div>
    </div>

    <div
      class="border border-gray-200 rounded-lg shadow-sm divide-y divide-block-border"
    >
      <div class="px-6 pt-6 pb-4">
        <h2 class="text-4xl text-center leading-6 font-bold text-gray-900">
          Premium
        </h2>
        <div
          class="mt-8 block w-full border border-gray-800 rounded-md py-4 text-2xl font-semibold text-gray-800 text-center whitespace-nowrap"
        >
          Coming soon
        </div>
        <div class="mt-6 flex justify-center flex-row">
          <span class="ml-1 text-4xl font-extrabold text-gray-900">{{
            state.isMonthly ? "$48" : "$36"
          }}</span>
          <div class="ml-2 flex flex-col">
            <span class="text-sm font-medium text-gray-500 whitespace-nowrap"
              >per user</span
            >
            <span class="text-sm font-medium text-gray-500 whitespace-nowrap"
              >per month</span
            >
          </div>
        </div>

        <div class="mt-8 text-sm text-center">Priority support</div>
      </div>
      <div class="pt-6 pb-8 px-6">
        <h3 class="text-xs font-medium text-gray-900 tracking-wide uppercase">
          What's included
        </h3>
        <ul class="mt-6 space-y-4">
          <li class="flex space-x-3">
            <!-- Heroicon name: solid/check -->
            <heroicons-solid:check
              class="flex-shrink-0 h-5 w-5 text-green-500"
            />
            <span class="text-sm text-main">Unlimited issues </span>
          </li>

          <li class="flex space-x-3">
            <!-- Heroicon name: solid/check -->
            <heroicons-solid:check
              class="flex-shrink-0 h-5 w-5 text-green-500"
            />
            <span class="text-sm text-main">Unlimited members</span>
          </li>

          <li class="flex space-x-3">
            <!-- Heroicon name: solid/check -->
            <heroicons-solid:check
              class="flex-shrink-0 h-5 w-5 text-green-500"
            />
            <span class="text-sm text-main">Unlimited environments</span>
          </li>

          <li class="flex space-x-3">
            <!-- Heroicon name: solid/check -->
            <heroicons-solid:check
              class="flex-shrink-0 h-5 w-5 text-green-500"
            />
            <span class="text-sm text-main">Unlimited instances</span>
          </li>

          <li class="flex space-x-3">
            <!-- Heroicon name: solid/check -->
            <heroicons-solid:check
              class="flex-shrink-0 h-5 w-5 text-green-500"
            />
            <span class="text-sm text-main">Unlimited data sources</span>
          </li>

          <li class="flex space-x-3">
            <!-- Heroicon name: solid/check -->
            <heroicons-solid:check
              class="flex-shrink-0 h-5 w-5 text-green-500"
            />
            <span class="text-sm text-main">Slack integration</span>
          </li>

          <li class="flex space-x-3">
            <!-- Heroicon name: solid/check -->
            <heroicons-solid:check
              class="flex-shrink-0 h-5 w-5 text-green-500"
            />
            <span class="text-sm text-main">DBA and Owner roles</span>
          </li>

          <li class="flex space-x-3">
            <!-- Heroicon name: solid/check -->
            <heroicons-solid:check
              class="flex-shrink-0 h-5 w-5 text-green-500"
            />
            <span class="text-sm text-main">SSO</span>
          </li>

          <li class="flex space-x-3">
            <!-- Heroicon name: solid/check -->
            <heroicons-solid:check
              class="flex-shrink-0 h-5 w-5 text-green-500"
            />
            <span class="text-sm text-main">Custom branding</span>
          </li>

          <li class="flex space-x-3">
            <!-- Heroicon name: solid/check -->
            <heroicons-solid:check
              class="flex-shrink-0 h-5 w-5 text-green-500"
            />
            <span class="text-sm text-main">Audit log</span>
          </li>

          <li class="flex space-x-3">
            <!-- Heroicon name: solid/check -->
            <heroicons-solid:check
              class="flex-shrink-0 h-5 w-5 text-green-500"
            />
            <span class="text-sm text-main">Roadmap sharing</span>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { reactive } from "vue";

interface LocalState {
  isMonthly: boolean;
}

export default {
  name: "PricingTable",
  props: {},
  setup() {
    const state = reactive<LocalState>({
      isMonthly: false,
    });

    return {
      state,
    };
  },
};
</script>
