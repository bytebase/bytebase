<template>
  <NTooltip :animated="false" :delay="250">
    <template #trigger>
      <EyeOffIcon
        class="w-3 h-3 -mb-0.5"
        :class="[clickable && 'cursor-pointer']"
        v-bind="$attrs"
        @click="handleClick"
      />
    </template>

    <span class="whitespace-nowrap">
      {{ $t("sensitive-data.self") }}
    </span>
  </NTooltip>
</template>

<script lang="ts" setup>
import { EyeOffIcon } from "lucide-vue-next";
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { useRouter } from "vue-router";
import { WORKSPACE_ROUTE_GLOBAL_MASKING } from "@/router/dashboard/workspaceRoutes";
import { hasWorkspacePermissionV2 } from "@/utils";

const router = useRouter();

const clickable = computed(() => {
  return hasWorkspacePermissionV2("bb.policies.update");
});

const handleClick = () => {
  if (!clickable.value) {
    return;
  }
  const url = router.resolve({
    name: WORKSPACE_ROUTE_GLOBAL_MASKING,
  });
  window.open(url.href, "_BLANK");
};
</script>
