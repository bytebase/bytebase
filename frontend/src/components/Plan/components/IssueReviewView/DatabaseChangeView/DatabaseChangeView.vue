<template>
  <div class="w-full flex flex-col">
    <!-- Spec Tabs with content (read-only on Issue page) -->
    <SpecTabs v-model:selected-spec-id="selectedSpecId">
      <div v-if="selectedSpec" class="flex flex-col gap-2">
        <TargetListSection />
        <div class="h-72 flex flex-col">
          <StatementSection header-variant="section" />
        </div>
        <OptionsSection />
      </div>
    </SpecTabs>

  </div>
</template>

<script setup lang="ts">
import { computed, provide, ref, watch } from "vue";
import { usePlanContext } from "../../../logic";
import {
  FORCE_READONLY_KEY,
  provideSelectedSpec,
} from "../../SpecDetailView/context";
import TargetListSection from "../../SpecDetailView/TargetListSection.vue";
import StatementSection from "../../StatementSection/StatementSection.vue";
import OptionsSection from "./OptionsSection.vue";
import SpecTabs from "./SpecTabs.vue";

const { plan } = usePlanContext();

// Issue page shows changes read-only — editing happens on Plan Detail Page
provide(FORCE_READONLY_KEY, true);

// Local state for selected spec (not route-based)
const selectedSpecId = ref<string>(plan.value.specs[0]?.id ?? "");

const selectedSpec = computed(() => {
  return plan.value.specs.find((spec) => spec.id === selectedSpecId.value);
});

provideSelectedSpec(
  computed(() => {
    const spec = selectedSpec.value;
    if (!spec) {
      throw new Error("No spec selected");
    }
    return spec;
  })
);

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
