<template>
  <div class="flex flex-col gap-y-4">
    <div class="textinfolabel">
      {{ $t("role.setting.description") }}
      <LearnMoreLink
        url="https://docs.bytebase.com/administration/roles?source=console"
      />
    </div>
    <div class="w-full flex flex-row justify-end items-center">
      <PermissionGuardWrapper
        v-slot="slotProps"
        :permissions="['bb.roles.create']"
      >
        <NButton
          type="primary"
          class="capitalize"
          :disabled="slotProps.disabled"
          @click="addRole"
        >
          <template #icon>
            <PlusIcon class="h-4 w-4" />
            <FeatureBadge
              :feature="PlanFeature.FEATURE_CUSTOM_ROLES"
              class="mr-1 text-white"
            />
          </template>
          {{ $t("role.setting.add") }}
        </NButton>
      </PermissionGuardWrapper>
    </div>
    <RoleDataTable
      :role-list="filteredRoleList"
      :show-placeholder="state.ready"
      @select-role="selectRole($event, 'EDIT')"
    />

    <div
      v-if="!state.ready"
      class="relative flex flex-col h-32 items-center justify-center"
    >
      <BBSpin />
    </div>

    <RolePanel
      :role="state.detail.role"
      :mode="state.detail.mode"
      @close="state.detail.role = undefined"
    />

    <FeatureModal
      :feature="PlanFeature.FEATURE_CUSTOM_ROLES"
      :open="showFeatureModal"
      @cancel="showFeatureModal = false"
    />
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { sortBy } from "lodash-es";
import { PlusIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, onMounted, reactive, ref } from "vue";
import { BBSpin } from "@/bbkit";
import { FeatureBadge, FeatureModal } from "@/components/FeatureGuard";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { featureToRef, useRoleStore } from "@/store";
import { PRESET_ROLES } from "@/types";
import type { Role } from "@/types/proto-es/v1/role_service_pb";
import { RoleSchema } from "@/types/proto-es/v1/role_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { RoleDataTable, RolePanel } from "./Setting/components";
import { provideCustomRoleSettingContext } from "./Setting/context";

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

const roleStore = useRoleStore();
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

const hasCustomRoleFeature = featureToRef(PlanFeature.FEATURE_CUSTOM_ROLES);
const showFeatureModal = ref(false);

const filteredRoleList = computed(() => {
  const sortedRoles = sortBy(roleStore.roleList, (role) => {
    return PRESET_ROLES.includes(role.name)
      ? PRESET_ROLES.indexOf(role.name)
      : roleStore.roleList.length;
  });
  const keyword = state.filter.keyword.trim().toLowerCase();
  if (!keyword) return sortedRoles;
  return sortedRoles.filter((role) => {
    return (
      role.name.toLowerCase().includes(keyword) ||
      role.description.toLowerCase().includes(keyword)
    );
  });
});

const addRole = () => {
  selectRole(create(RoleSchema, {}), "ADD");
};

const selectRole = (role: Role | undefined, mode?: "ADD" | "EDIT") => {
  state.detail.role = role;
  if (mode) {
    state.detail.mode = mode;
  }
};

const prepare = async () => {
  try {
    await roleStore.fetchRoleList();
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
