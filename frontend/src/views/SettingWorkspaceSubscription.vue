<template>
  <div class="bg-white">
    <div class="mx-auto">
      <div class="sm:flex sm:flex-col sm:align-center">
        <!-- TODO: implement license upload -->
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
