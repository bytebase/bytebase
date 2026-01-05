<template>
  <NoPermissionPlaceholder
    class="mx-6 my-2"
    :resources="resources"
    :permissions="permissions"
  >
    <template #action>
      <div class="mt-2">
        <NButton size="small" @click.prevent="goHome">
          <template #icon><ChevronLeftIcon /></template>
          {{ $t("error-page.go-back-home") }}
        </NButton>
      </div>
    </template>
  </NoPermissionPlaceholder>
</template>

<script lang="ts" setup>
import { ChevronLeftIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import NoPermissionPlaceholder from "@/components/misc/NoPermissionPlaceholder.vue";
import { WORKSPACE_ROOT_MODULE } from "@/router/dashboard/workspaceRoutes";
import type { Permission } from "@/types";

const route = useRoute();
const router = useRouter();

const permissions = computed<Permission[]>(() => {
  const permQuery = route.query.permissions;
  if (typeof permQuery === "string" && permQuery) {
    return permQuery.split(",").filter((r) => r) as Permission[];
  }
  return [];
});

const resources = computed(() => {
  const resources = route.query.resources;
  if (typeof resources === "string") {
    return resources.split(",").filter((r) => r);
  }
  return [];
});

const goHome = () => {
  router.push({ name: WORKSPACE_ROOT_MODULE });
};
</script>
