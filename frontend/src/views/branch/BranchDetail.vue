<template>
  <template v-if="ready">
    <template v-if="isCreating">
      <BranchCreateView />
    </template>
    <template v-else-if="branch">
      <BranchDetailView :branch="branch" />
    </template>
  </template>
</template>

<script lang="ts" setup>
import { useTitle } from "@vueuse/core";
import { computed, ref, watch } from "vue";
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
const branchName = ref<string>("");
const ready = ref<boolean>(false);

const isCreating = computed(() => route.params.branchName === "new");
const branch = computed(() => {
  return schemaDesignStore.getSchemaDesignByName(branchName.value);
});

watch(
  () => route.params,
  async () => {
    if (isCreating.value) {
      return;
    }

    // Prepare branch name from route params.
    const sheetId = (route.params.branchName as string) || "";
    if (!sheetId) {
      return;
    }
    const sheet = await sheetStore.getOrFetchSheetByUID(`${sheetId}`);
    if (sheet) {
      const [projectName] = getProjectAndSheetId(sheet.name);
      await projectStore.getOrFetchProjectByName(`projects/${projectName}`);
      branchName.value = `projects/${projectName}/schemaDesigns/${sheetId}`;
    }
  },
  {
    immediate: true,
    deep: true,
  }
);

watch(
  () => branchName.value,
  async () => {
    ready.value = false;
    if (isCreating.value || !branchName.value) {
      ready.value = true;
      return;
    }

    await schemaDesignStore.fetchSchemaDesignByName(
      branchName.value,
      false /* useCache */
    );
    ready.value = true;
  },
  {
    immediate: true,
  }
);

const documentTitle = computed(() => {
  if (isCreating.value) {
    return t("schema-designer.new-branch");
  } else {
    if (branch.value) {
      return branch.value.title;
    }
  }
  return t("common.loading");
});
useTitle(documentTitle);
</script>
