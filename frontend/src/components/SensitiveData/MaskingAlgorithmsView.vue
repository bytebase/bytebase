<template>
  <div class="w-full space-y-4">
    <div class="flex items-center justify-end">
      <NButton
        type="primary"
        :disabled="!hasPermission || !hasSensitiveDataFeature"
        @click="onCreate"
      >
        {{ $t("common.add") }}
      </NButton>
    </div>
    <div class="space-y-5 divide-y-2 pb-10 divide-gray-100">
      <MaskingAlgorithmsTable
        :readonly="!hasPermission || !hasSensitiveDataFeature"
        :row-clickable="false"
        @edit="onEdit"
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
import { MaskingAlgorithmSetting_Algorithm as Algorithm } from "@/types/proto/v1/setting_service";
import { hasWorkspacePermissionV1 } from "@/utils";
import MaskingAlgorithmsCreateDrawer from "./components/MaskingAlgorithmsCreateDrawer.vue";
import MaskingAlgorithmsTable from "./components/MaskingAlgorithmsTable.vue";

interface LocalState {
  showCreateDrawer: boolean;
  pendingEditData: Algorithm;
}

const state = reactive<LocalState>({
  showCreateDrawer: false,
  pendingEditData: Algorithm.fromPartial({
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

const onCreate = () => {
  state.pendingEditData = Algorithm.fromPartial({
    id: uuidv4(),
  });
  state.showCreateDrawer = true;
};

const onDrawerDismiss = () => {
  state.showCreateDrawer = false;
};

const onEdit = (data: Algorithm) => {
  state.pendingEditData = data;
  state.showCreateDrawer = true;
};
</script>
