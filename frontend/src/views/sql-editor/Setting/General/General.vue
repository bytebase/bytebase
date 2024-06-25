<template>
  <div
    class="w-full flex flex-col gap-4 py-4 px-4 divide-y divide-block-border overflow-y-auto"
  >
    <DatabaseChangeModeSetting :allow-edit="allowEdit" />
    <AIAugmentationSetting :allow-edit="allowEdit" />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import {
  AIAugmentationSetting,
  DatabaseChangeModeSetting,
} from "@/components/GeneralSetting";
import { useCurrentUserV1 } from "@/store";
import { hasWorkspacePermissionV2 } from "@/utils";

const me = useCurrentUserV1();

const allowEdit = computed((): boolean => {
  return hasWorkspacePermissionV2(me.value, "bb.settings.set");
});
</script>
