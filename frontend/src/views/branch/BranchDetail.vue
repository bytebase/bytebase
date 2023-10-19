<template>
  <template v-if="data.ready">
    <template v-if="isCreating">
      <BranchCreateView />
    </template>
    <template v-else-if="data.branch">
      <BranchDetailView :branch="data.branch" />
    </template>
  </template>
</template>

<script lang="ts" setup>
import { asyncComputed, useTitle } from "@vueuse/core";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import BranchCreateView from "@/components/Branch/BranchCreateView.vue";
import BranchDetailView from "@/components/Branch/BranchDetailView.vue";
import { useProjectV1Store, useSheetV1Store } from "@/store";
import { useSchemaDesignStore } from "@/store/modules/schemaDesign";
import { getProjectAndSheetId } from "@/store/modules/v1/common";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";

const { t } = useI18n();
const route = useRoute();
const sheetStore = useSheetV1Store();
const projectStore = useProjectV1Store();
const schemaDesignStore = useSchemaDesignStore();

const isCreating = computed(() => route.params.branchName === "new");

const data = asyncComputed<{ ready: boolean; branch?: SchemaDesign }>(
  async () => {
    if (isCreating.value) {
      return { ready: true };
    } else {
      const sheetId = (route.params.branchName as string) || "";
      if (!sheetId) {
        return { ready: false };
      }
      const sheet = await sheetStore.getOrFetchSheetByUID(`${sheetId}`);
      if (sheet) {
        const [projectName] = getProjectAndSheetId(sheet.name);
        await projectStore.getOrFetchProjectByName(`projects/${projectName}`);
        const branch = await schemaDesignStore.fetchSchemaDesignByName(
          `projects/${projectName}/schemaDesigns/${sheetId}`,
          true /* useCache */
        );
        return { ready: true, branch };
      }
    }

    return { ready: false };
  },
  { ready: false }
);

const documentTitle = computed(() => {
  if (isCreating.value) {
    return t("schema-designer.new-branch");
  } else {
    if (data.value.branch) {
      return data.value.branch.title;
    }
  }
  return t("common.loading");
});
useTitle(documentTitle);
</script>
