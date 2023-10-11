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
import { useRoute } from "vue-router";
import BranchCreateView from "@/components/Branch/BranchCreateView.vue";
import BranchDetailView from "@/components/Branch/BranchDetailView.vue";
import { useProjectV1Store, useSheetV1Store } from "@/store";
import { useSchemaDesignStore } from "@/store/modules/schemaDesign";
import { getProjectAndSheetId } from "@/store/modules/v1/common";

const { t } = useI18n();
const route = useRoute();
const sheetStore = useSheetV1Store();
const projectStore = useProjectV1Store();
const schemaDesignStore = useSchemaDesignStore();
const initialized = ref(false);

const isCreating = computed(() => route.params.branchName === "new");
const branchName = computed(() => {
  return `projects/${route.params.projectName}/schemaDesigns/${route.params.branchName}`;
});

watchEffect(async () => {
  if (!isCreating.value) {
    const sheetId = (route.params.branchName || "") as string;
    const sheet = await sheetStore.getOrFetchSheetByUID(`${sheetId}`);
    if (sheet) {
      const [projectName] = getProjectAndSheetId(sheet.name);
      await projectStore.getOrFetchProjectByName(`projects/${projectName}`);
      await schemaDesignStore.getOrFetchSchemaDesignByName(
        `projects/${projectName}/schemaDesigns/${sheetId}`
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
      return `${schemaDesign.title}`;
    }
  }
  return t("common.loading");
});
useTitle(documentTitle);
</script>
