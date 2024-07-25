<template>
  <slot />
</template>

<script lang="ts" setup>
import { useKBarHandler } from "@bytebase/vue-kbar";
import { defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { watch, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { useRecentVisit } from "@/router/useRecentVisit";

const handler = useKBarHandler();
const route = useRoute();
const router = useRouter();
const { recentVisit } = useRecentVisit();
const { t } = useI18n();

watch(
  () => route.fullPath,
  () => {
    // force hide kbar when page navigated
    handler.value.hide();
  }
);

const actions = computed(() => {
  return recentVisit.value
    .slice(1) // The first item is current page, just skip it.
    .filter(({ title, path }) => title && path)
    .map(({ title, path }, index) =>
      defineAction({
        // here `id` looks like "bb.recent_visited.1"
        id: `bb.recently_visited.${index + 1}`,
        section: t("kbar.recently-visited"),
        name: title,
        subtitle: path,
        shortcut: ["g", `${index + 1}`],
        keywords: "recently visited",
        perform: () => router.push({ path }),
      })
    );
});

// prepend recent visit actions to kbar
useRegisterActions(actions, true);
</script>
