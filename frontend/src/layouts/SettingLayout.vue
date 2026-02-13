<template>
  <div class="px-4 min-h-full">
    <router-view :allow-edit="allowEdit" v-bind="$attrs" />
  </div>
</template>

<script lang="ts" setup>
import { computed, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { hasWorkspacePermissionV2, setDocumentTitle } from "@/utils";

const { t } = useI18n();
const route = useRoute();

const allowEdit = computed((): boolean => {
  return hasWorkspacePermissionV2("bb.settings.set");
});

watch(
  () => route.meta.title,
  () => {
    const pageTitle = route.meta.title ? route.meta.title(route) : undefined;
    if (pageTitle) {
      setDocumentTitle(pageTitle, t("common.settings"));
    }
  },
  { immediate: true }
);
</script>
