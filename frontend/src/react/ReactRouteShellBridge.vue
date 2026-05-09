<template>
  <ReactPageMount
    :page="page"
    :page-props="{
      ...pageProps,
      routeKey: route.fullPath,
      onReady: handleReady,
    }"
  />

  <teleport v-if="targets.content" :to="targets.content">
    <router-view
      v-if="routerViewName"
      :name="routerViewName"
      v-bind="targets.routeProps ?? {}"
    />
    <router-view v-else v-bind="targets.routeProps ?? {}" />
  </teleport>

  <teleport v-if="targets.quickstart" :to="targets.quickstart">
    <ReactPageMount page="Quickstart" container-class="w-full" />
  </teleport>
</template>

<script lang="ts" setup>
import { computed, shallowRef } from "vue";
import { useRoute } from "vue-router";
import { provideBodyLayoutContext } from "@/layouts/common";
import type { ReactRouteShellTargets } from "@/react/dashboard-shell";
import ReactPageMount from "@/react/ReactPageMount.vue";

defineProps<{
  page: string;
  pageProps?: Record<string, unknown>;
  routerViewName?: string;
}>();

const route = useRoute();
const targets = shallowRef<ReactRouteShellTargets>({
  content: null,
});
const mainContainerRef = computed(
  () => targets.value.mainContainer ?? undefined
);

const handleReady = (nextTargets: ReactRouteShellTargets | null) => {
  targets.value = nextTargets ?? { content: null };
};

provideBodyLayoutContext({
  mainContainerRef,
});
</script>
