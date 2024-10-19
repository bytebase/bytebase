<template>
  <NTooltip :animated="false" :delay="250">
    <template #trigger>
      <heroicons-outline:eye-slash
        class="w-[12px] h-[12px] -mb-[2px]"
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
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { useRouter } from "vue-router";
import { WORKSPACE_ROUTE_DATA_MASKING } from "@/router/dashboard/workspaceRoutes";
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
    name: WORKSPACE_ROUTE_DATA_MASKING,
    hash: "#sensitive-column-list",
  });
  window.open(url.href, "_BLANK");
};
</script>
