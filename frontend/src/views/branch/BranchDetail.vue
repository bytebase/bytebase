<template>
  <template v-if="initialized">
    <template v-if="isCreating">
      <BranchCreateView />
    </template>
    <template v-else-if="branchName">
      <BranchDetailView :schema-design-name="branchName" />
    </template>
  </template>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from "vue";
import BranchCreateView from "@/components/Branch/BranchCreateView.vue";
import BranchDetailView from "@/components/Branch/BranchDetailView.vue";
import { useProjectV1Store, useSheetV1Store } from "@/store";
import { getProjectAndSheetId } from "@/store/modules/v1/common";
import { idFromSlug } from "@/utils";

const props = defineProps({
  branchSlug: {
    required: true,
    type: String,
  },
});

const sheetStore = useSheetV1Store();
const projectStore = useProjectV1Store();
const initialized = ref(false);

const isCreating = computed(() => props.branchSlug === "new");
const branchName = computed(() => {
  const sheetId = idFromSlug(props.branchSlug);
  const sheet = sheetStore.getSheetByUID(`${sheetId}`);
  if (!sheet) {
    return undefined;
  }
  const [project] = getProjectAndSheetId(sheet.name);
  return `projects/${project}/schemaDesigns/${sheetId}`;
});

onMounted(async () => {
  if (!isCreating.value) {
    const sheetId = idFromSlug(props.branchSlug);
    const sheet = await sheetStore.getOrFetchSheetByUID(`${sheetId}`);
    if (sheet) {
      const [project] = getProjectAndSheetId(sheet.name);
      await projectStore.getOrFetchProjectByName(`projects/${project}`);
    }
  }
  initialized.value = true;
});
</script>
