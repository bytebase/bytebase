<template>
  <div class="bg-white">
    <div class="mx-auto">
      <div class="sm:flex sm:flex-col sm:align-center">
        <h1 class="text-5xl font-extrabold text-gray-900 sm:text-center">
          Pricing Plans
        </h1>
        <p class="mt-5 text-xl text-gray-500 sm:text-center">
          During beta period, we offer a 25% discount for the Standard plan.
        </p>
        <PricingTable />
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import PricingTable from "../components/PricingTable.vue";

export default {
  name: "SettingWorkspacePlan",
  components: {
    PricingTable,
  },
};
</script>
