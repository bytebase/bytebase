<template>
  <div class="w-full flex-1 flex flex-col">
    <SpecDetailView />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { usePlanContext } from "../../logic";
import { SpecDetailView } from "../index";
import { provideSelectedSpec } from "../SpecDetailView/context";

const props = withDefaults(
  defineProps<{
    initialSpecId?: string;
  }>(),
  { initialSpecId: "" }
);

const { plan } = usePlanContext();

// Injection-based spec selection
const selectedSpecId = ref<string>(props.initialSpecId);

watch(
  () => props.initialSpecId,
  (specId) => {
    if (specId) {
      selectedSpecId.value = specId;
      return;
    }
    if (!selectedSpecId.value && plan.value.specs.length > 0) {
      selectedSpecId.value = plan.value.specs[0].id;
    }
  },
  { immediate: true }
);

watch(
  () => plan.value.specs,
  (specs) => {
    if (specs.length === 0) {
      selectedSpecId.value = "";
      return;
    }

    const hasSelectedSpec = specs.some(
      (spec) => spec.id === selectedSpecId.value
    );
    if (!hasSelectedSpec) {
      selectedSpecId.value = props.initialSpecId || specs[0].id;
    }
  },
  { immediate: true }
);

const selectedSpecRef = computed(() => {
  const spec = plan.value.specs.find((s) => s.id === selectedSpecId.value);
  return spec ?? plan.value.specs[0];
});

const setSelectedSpecId = (specId: string) => {
  selectedSpecId.value = specId;
};
provideSelectedSpec(selectedSpecRef, setSelectedSpecId);
</script>
