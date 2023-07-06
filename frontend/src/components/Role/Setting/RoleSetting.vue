<template>
  <div class="space-y-4 pb-4">
    <div class="flex items-center justify-between gap-x-6">
      <div class="flex-1 textinfolabel">
        {{ $t("role.setting.description") }}
      </div>
      <div>
        <NButton type="primary" :disabled="!allowAdmin" @click="addRole">
          {{ $t("role.setting.add") }}
        </NButton>
      </div>
    </div>
    <div>
      <RoleTable
        :role-list="filteredRoleList"
        :show-placeholder="state.ready"
        @select-role="selectRole($event, 'EDIT')"
      />

      <div
        v-if="!state.ready"
        class="relative flex flex-col h-[8rem] items-center justify-center"
      >
        <BBSpin />
      </div>
    </div>

    <RolePanel
      :role="state.detail.role"
      :mode="state.detail.mode"
      @close="state.detail.role = undefined"
    />

    <FeatureModal
      feature="bb.feature.custom-role"
      :open="showFeatureModal"
      @cancel="showFeatureModal = false"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, ref } from "vue";
import { NButton } from "naive-ui";

import { RoleTable, RolePanel } from "./components";
import { featureToRef, useRoleStore } from "@/store";
import { Role } from "@/types/proto/v1/role_service";
import { useWorkspacePermissionV1 } from "@/utils";
import { provideCustomRoleSettingContext } from "./context";

type LocalState = {
  ready: boolean;
  detail: {
    role: Role | undefined;
    mode: "ADD" | "EDIT";
  };
  filter: {
    keyword: string;
  };
};

const store = useRoleStore();
const state = reactive<LocalState>({
  ready: false,
  detail: {
    role: undefined,
    mode: "ADD",
  },
  filter: {
    keyword: "",
  },
});

const hasCustomRoleFeature = featureToRef("bb.feature.custom-role");
const showFeatureModal = ref(false);

const allowAdmin = useWorkspacePermissionV1(
  "bb.permission.workspace.manage-general"
);

const filteredRoleList = computed(() => {
  const keyword = state.filter.keyword.trim().toLowerCase();
  if (!keyword) return store.roleList;
  return store.roleList.filter((role) => {
    return (
      role.name.toLowerCase().includes(keyword) ||
      role.description.toLowerCase().includes(keyword)
    );
  });
});

const addRole = () => {
  selectRole(Role.fromJSON({}), "ADD");
};

const selectRole = (role: Role | undefined, mode?: "ADD" | "EDIT") => {
  state.detail.role = role;
  if (mode) {
    state.detail.mode = mode;
  }
};

const prepare = async () => {
  try {
    await store.fetchRoleList();
  } finally {
    state.ready = true;
  }
};
onMounted(prepare);

provideCustomRoleSettingContext({
  hasCustomRoleFeature,
  showFeatureModal,
});
</script>
