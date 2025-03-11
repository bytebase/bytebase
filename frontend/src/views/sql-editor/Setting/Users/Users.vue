<template>
  <div
    class="w-full flex flex-col py-4 px-4 gap-4 divide-block-border"
    v-bind="$attrs"
  >
    <SettingWorkspaceUsers :on-click-user="handleClickUser" />
  </div>
  <Drawer v-model:show="state.detail.show">
    <DrawerContent
      :title="detailTitle"
      style="width: 800px; max-width: calc(100vw - 4rem)"
    >
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
import { Drawer, DrawerContent } from "@/components/v2";
import { useUserStore } from "@/store";
import type { User } from "@/types/proto/v1/user_service";
import ProfileDashboard from "@/views/ProfileDashboard.vue";
import SettingWorkspaceUsers from "@/views/SettingWorkspaceUsers.vue";

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

// TODO(ed): not use the drawer.
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
