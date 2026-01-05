<template>
  <div class="px-4 min-h-full">
    <router-view :allow-edit="allowEdit" v-bind="$attrs" />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { hasWorkspacePermissionV2 } from "@/utils";

const allowEdit = computed((): boolean => {
  return hasWorkspacePermissionV2("bb.settings.set");
});
</script>
