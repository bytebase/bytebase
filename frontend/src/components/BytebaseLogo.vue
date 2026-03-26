<template>
  <div
    class="shrink-0 max-w-44 flex items-center overflow-hidden"
    @click="recordRedirect"
  >
    <!-- The active-class and exact-active-class need to set as ""
     to avoid unexpected route-link class override -->
    <component
      :is="component"
      :to="{
        name: redirect,
      }"
      class="h-full w-full select-none flex flex-row justify-center items-center"
      active-class=""
      exact-active-class=""
    >
      <img
        v-if="workspaceStore.currentWorkspace?.logo"
        :src="workspaceStore.currentWorkspace?.logo"
        alt="branding logo"
        class="h-full object-contain"
      />

      <img
        v-else
        class="h-8 md:h-10 w-auto object-contain"
        src="@/assets/logo-full.svg"
        alt="Bytebase"
      />
    </component>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useRouter } from "vue-router";
import { useRecentVisit } from "@/router/useRecentVisit";
import { useWorkspaceV1Store } from "@/store";

const props = withDefaults(
  defineProps<{
    redirect?: string;
  }>(),
  {
    redirect: "",
  }
);

const component = computed(() => (props.redirect ? "router-link" : "span"));
const { record } = useRecentVisit();
const router = useRouter();
const workspaceStore = useWorkspaceV1Store();

const recordRedirect = () => {
  if (!props.redirect) {
    return;
  }
  const route = router.resolve({
    name: props.redirect,
  });
  record(route.fullPath);
};
</script>
