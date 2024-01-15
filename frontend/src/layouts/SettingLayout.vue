<template>
  <div class="p-6">
    <router-view />
  </div>
</template>

<script lang="ts" setup>
import { onMounted } from "vue";
import { useRouter } from "vue-router";
import { useCurrentUserV1 } from "@/store";
import { hasWorkspacePermissionV2 } from "@/utils";

const router = useRouter();
const currentUser = useCurrentUserV1();

onMounted(() => {
  const hasPermission = hasWorkspacePermissionV2(
    currentUser.value,
    "bb.settings.set"
  );

  if (!hasPermission) {
    router.push({
      name: "error.403",
      replace: false,
    });
  }
});
</script>
