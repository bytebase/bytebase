<template>
  <div
    class="flex flex-col py-4 px-4 gap-4 divide-block-border"
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
import { computed, onMounted, reactive, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import Drawer from "@/components/v2/Container/Drawer.vue";
import DrawerContent from "@/components/v2/Container/DrawerContent.vue";
import { useUserStore } from "@/store";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import ProfileDashboard from "@/views/ProfileDashboard.vue";
import SettingWorkspaceMembers from "@/views/SettingWorkspaceMembers.vue";

type LocalState = {
  detail: { show: boolean; user?: User };
};

const route = useRoute();
const router = useRouter();
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

onMounted(async () => {
  const maybeEmail = route.hash.replace(/^#*/g, "");
  if (maybeEmail) {
    const user = await useUserStore().getOrFetchUserByIdentifier(maybeEmail);
    if (user) {
      state.detail.show = true;
      state.detail.user = user;
    }
  }

  watch(
    [() => state.detail.show, () => state.detail.user?.email],
    ([show, email]) => {
      if (show && email) {
        router.replace({ hash: `#${email}` });
      } else {
        router.replace({ hash: "" });
      }
    }
  );
});
</script>
