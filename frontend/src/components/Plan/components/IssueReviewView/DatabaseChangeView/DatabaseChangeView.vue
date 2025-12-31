<template>
  <div class="w-full flex flex-col">
    <!-- Spec Tabs with integrated content -->
    <SpecTabs v-model:selected-spec-id="selectedSpecId">
      <div v-if="selectedSpec" class="flex flex-col gap-2">
        <TargetListSection />
        <div class="h-72 flex flex-col">
          <StatementSection />
        </div>
        <OptionsSection />
      </div>
    </SpecTabs>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { usePlanContext } from "../../../logic";
import { provideSelectedSpec } from "../../SpecDetailView/context";
import TargetListSection from "../../SpecDetailView/TargetListSection.vue";
import StatementSection from "../../StatementSection/StatementSection.vue";
import OptionsSection from "./OptionsSection.vue";
import SpecTabs from "./SpecTabs.vue";

const { plan } = usePlanContext();

// Local state for selected spec (not route-based)
const selectedSpecId = ref<string>(plan.value.specs[0]?.id ?? "");

// Computed selected spec
const selectedSpec = computed(() => {
  return plan.value.specs.find((spec) => spec.id === selectedSpecId.value);
});

// Provide selected spec for child components
provideSelectedSpec(
  computed(() => {
    const spec = selectedSpec.value;
    if (!spec) {
      throw new Error("No spec selected");
    }
    return spec;
  })
);

// Update selected spec when specs change
watch(
  () => plan.value.specs,
  (specs) => {
    if (specs.length > 0 && !specs.find((s) => s.id === selectedSpecId.value)) {
      selectedSpecId.value = specs[0].id;
    }
  },
  { immediate: true }
);
</script>
