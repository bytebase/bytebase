<template>
  <div class="flex flex-col gap-4 py-4 px-4">
    <SettingWorkspaceSSO
      :on-click-create="handleClickCreate"
      :on-click-view="handleClickView"
    />
  </div>

  <Drawer v-model:show="state.detail.show">
    <DrawerContent
      :title="drawerTitle"
      body-style="width: 60vw; min-width: 440px; max-width: calc(100vw - 4rem)"
    >
      <SettingWorkspaceSSODetail
        :sso-id="state.detail.sso ? getSSOId(state.detail.sso.name) : undefined"
        :on-created="(sso) => (state.detail.sso = sso)"
        :on-updated="() => (state.detail.show = false)"
        :on-deleted="() => (state.detail.show = false)"
      />
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { Drawer, DrawerContent } from "@/components/v2";
import { getSSOId } from "@/store/modules/v1/common";
import type { IdentityProvider } from "@/types/proto/v1/idp_service";
import SettingWorkspaceSSO from "@/views/SettingWorkspaceSSO.vue";
import SettingWorkspaceSSODetail from "@/views/SettingWorkspaceSSODetail.vue";

type LocalState = {
  detail: {
    show: boolean;
    sso: IdentityProvider | undefined;
  };
};
const { t } = useI18n();

const state = reactive<LocalState>({
  detail: {
    show: false,
    sso: undefined,
  },
});

const drawerTitle = computed(() => {
  const { sso } = state.detail;
  if (!sso) return t("settings.sso.create");
  return sso.title;
});

const handleClickCreate = () => {
  state.detail = {
    show: true,
    sso: undefined,
  };
};
const handleClickView = (sso: IdentityProvider) => {
  state.detail = {
    show: true,
    sso,
  };
};
</script>
