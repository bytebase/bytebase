<template>
  <div
    class="w-full flex flex-col py-4 px-4 divide-y divide-block-border overflow-y-auto"
  >
    <DatabaseChangeModeSetting :allow-edit="allowEdit" class="pb-0" />

    <div class="pt-6 pb-0 lg:flex">
      <div class="text-left lg:w-1/4">
        <div class="flex items-center space-x-2">
          <h1 class="text-2xl font-bold">
            {{ $t("settings.general.workspace.security") }}
          </h1>
        </div>
      </div>
      <div class="flex-1 lg:px-4 gap">
        <MaximumSQLResultSizeSetting
          :allow-edit="allowEdit"
          :show-update-button="true"
        />
      </div>
    </div>
    <AIAugmentationSetting :allow-edit="allowEdit" class="pb-0" />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import {
  AIAugmentationSetting,
  DatabaseChangeModeSetting,
  MaximumSQLResultSizeSetting,
} from "@/components/GeneralSetting";
import { useCurrentUserV1 } from "@/store";
import { hasWorkspacePermissionV2 } from "@/utils";

const me = useCurrentUserV1();

const allowEdit = computed((): boolean => {
  return hasWorkspacePermissionV2(me.value, "bb.settings.set");
});
</script>
