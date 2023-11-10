<template>
  <div class="w-full mt-4 space-y-4">
    <div class="flex items-center justify-end">
      <NButton
        type="primary"
        :disabled="!hasPermission || !hasSensitiveDataFeature"
        @click="state.showCreateDrawer = true"
      >
        {{ $t("settings.sensitive-data.algorithms.add") }}
      </NButton>
    </div>
    <div class="space-y-5 divide-y-2 pb-10 divide-gray-100">
      <MaskingAlgorithmsTable
        :readonly="!hasPermission || !hasSensitiveDataFeature"
        :row-clickable="false"
        @on-edit="onEdit"
      />
    </div>
  </div>
  <MaskingAlgorithmsCreateDrawer
    :show="state.showCreateDrawer"
    :algorithm="state.pendingEditData"
    :readonly="!hasPermission || !hasSensitiveDataFeature"
    @dismiss="onDrawerDismiss"
  />
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive } from "vue";
import { featureToRef, useCurrentUserV1 } from "@/store";
import { MaskingAlgorithmSetting_Algorithm } from "@/types/proto/v1/setting_service";
import { hasWorkspacePermissionV1 } from "@/utils";

interface LocalState {
  showCreateDrawer: boolean;
  pendingEditData: MaskingAlgorithmSetting_Algorithm;
}

const state = reactive<LocalState>({
  showCreateDrawer: false,
  pendingEditData: MaskingAlgorithmSetting_Algorithm.fromPartial({
    id: uuidv4(),
  }),
});

const currentUserV1 = useCurrentUserV1();
const hasPermission = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-sensitive-data",
    currentUserV1.value.userRole
  );
});
const hasSensitiveDataFeature = featureToRef("bb.feature.sensitive-data");

const onDrawerDismiss = () => {
  state.showCreateDrawer = false;
};

const onEdit = (data: MaskingAlgorithmSetting_Algorithm) => {
  state.pendingEditData = data;
  state.showCreateDrawer = true;
};
</script>
