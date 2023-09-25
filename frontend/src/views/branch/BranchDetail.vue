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
import { useTitle } from "@vueuse/core";
import { computed, ref, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import BranchCreateView from "@/components/Branch/BranchCreateView.vue";
import BranchDetailView from "@/components/Branch/BranchDetailView.vue";
import { useProjectV1Store, useSheetV1Store } from "@/store";
import { useSchemaDesignStore } from "@/store/modules/schemaDesign";
import { getProjectAndSheetId } from "@/store/modules/v1/common";
import { idFromSlug } from "@/utils";

const props = defineProps({
  branchSlug: {
    required: true,
    type: String,
  },
});

const { t } = useI18n();
const sheetStore = useSheetV1Store();
const projectStore = useProjectV1Store();
const schemaDesignStore = useSchemaDesignStore();
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

watchEffect(async () => {
  if (!isCreating.value) {
    const sheetId = idFromSlug(props.branchSlug);
    const sheet = await sheetStore.getOrFetchSheetByUID(`${sheetId}`);
    if (sheet) {
      const [projectName] = getProjectAndSheetId(sheet.name);
      const project = await projectStore.getOrFetchProjectByName(
        `projects/${projectName}`
      );
      await schemaDesignStore.getOrFetchSchemaDesignByName(
        `projects/${project}/schemaDesigns/${sheetId}`
      );
    }
  }
  initialized.value = true;
});

const documentTitle = computed(() => {
  if (isCreating.value) {
    return t("schema-designer.new-branch");
  } else {
    if (branchName.value) {
      const schemaDesign = schemaDesignStore.getSchemaDesignByName(
        branchName.value
      );
      return `${schemaDesign.title} - ${t("common.branch")}`;
    }
  }
  return t("common.loading");
});
useTitle(documentTitle);
</script>
