<template>
  <div class="w-full h-full relative">
    <div
      v-if="state.loading"
      class="w-full h-full fixed md:absolute inset-0 flex justify-center items-center bg-white/50"
    >
      <NSpin />
    </div>
    <template v-else>
      <IssueDetail v-if="state.view === 'OLD'" :issue-slug="issueSlug" />
      <IssueDetailV1 v-if="state.view === 'NEW'" :issue-slug="issueSlug" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import { useRouter } from "vue-router";
import { useTitle } from "@vueuse/core";
import { useI18n } from "vue-i18n";
import { NSpin } from "naive-ui";

import { useActuatorV1Store, useIssueStore } from "@/store";
import { UNKNOWN_ID } from "@/types";
import { idFromSlug, isGrantRequestIssueType } from "@/utils";
import IssueDetail from "./IssueDetail.vue";
import IssueDetailV1 from "./IssueDetailV1.vue";

const props = defineProps({
  issueSlug: {
    required: true,
    type: String,
  },
});

const { t } = useI18n();
const router = useRouter();
const actuatorStore = useActuatorV1Store();

const state = reactive({
  view: "OLD" as "OLD" | "NEW",
  loading: true,
});
const enableNewIssueUI = computed(() => {
  return !!actuatorStore.serverInfo?.developmentUseV2Scheduler;
});

watch(
  [enableNewIssueUI, () => props.issueSlug],
  async ([enableNewIssueUI, slug]) => {
    if (!enableNewIssueUI) {
      state.view = "OLD";
      state.loading = false;
      return;
    }

    if (slug === "new") {
      state.view = "NEW";
      state.loading = false;
      return;
    } else {
      const legacyIssue = await useIssueStore().getOrFetchIssueById(
        idFromSlug(slug)
      );
      if (String(legacyIssue.id) === String(UNKNOWN_ID)) {
        router.replace({
          name: "error.404",
        });
        return;
      }
      if (isGrantRequestIssueType(legacyIssue.type)) {
        state.view = "OLD";
        state.loading = false;
        return;
      }
      state.view = "NEW";
      state.loading = false;
    }
  },
  { immediate: true }
);

useTitle(computed(() => t("common.loading")));
</script>
