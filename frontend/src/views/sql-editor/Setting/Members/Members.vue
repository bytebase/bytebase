<template>
  <div
    class="w-full flex flex-col py-4 px-4 gap-4 divide-block-border overflow-y-auto"
    v-bind="$attrs"
  >
    <SettingWorkspaceMembers :on-click-user="handleClickUser" />
  </div>
  <Drawer v-model:show="state.detail.show">
    <DrawerContent :title="detailTitle">
      <ProfileDashboard
        v-if="state.detail.user"
        :principal-email="state.detail.user.email"
      />
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import Drawer from "@/components/v2/Container/Drawer.vue";
import DrawerContent from "@/components/v2/Container/DrawerContent.vue";
import type { User } from "@/types/proto/v1/auth_service";
import ProfileDashboard from "@/views/ProfileDashboard.vue";
import SettingWorkspaceMembers from "@/views/SettingWorkspaceMembers.vue";

type LocalState = {
  detail: { show: boolean; user?: User };
};
const state = reactive<LocalState>({
  detail: { show: false },
});

const detailTitle = computed(() => {
  const { user } = state.detail;
  if (!user) return "";
  return `${user.title} (${user.email})`;
});

const handleClickUser = (user: User) => {
  state.detail.show = true;
  state.detail.user = user;
};
</script>
