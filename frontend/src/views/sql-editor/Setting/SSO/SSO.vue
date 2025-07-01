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
        :idp-id="
          state.detail.identityProvider
            ? getIdentityProviderResourceId(state.detail.identityProvider.name)
            : undefined
        "
        :on-created="
          (identityProvider) =>
            (state.detail.identityProvider = identityProvider)
        "
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
import { getIdentityProviderResourceId } from "@/store/modules/v1/common";
import type { IdentityProvider } from "@/types/proto-es/v1/idp_service_pb";
import SettingWorkspaceSSO from "@/views/SettingWorkspaceSSO.vue";
import SettingWorkspaceSSODetail from "@/views/SettingWorkspaceSSODetail.vue";

type LocalState = {
  detail: {
    show: boolean;
    identityProvider: IdentityProvider | undefined;
  };
};
const { t } = useI18n();

const state = reactive<LocalState>({
  detail: {
    show: false,
    identityProvider: undefined,
  },
});

const drawerTitle = computed(() => {
  const { identityProvider } = state.detail;
  if (!identityProvider) return t("settings.sso.create");
  return identityProvider.title;
});

const handleClickCreate = () => {
  state.detail = {
    show: true,
    identityProvider: undefined,
  };
};
const handleClickView = (identityProvider: IdentityProvider) => {
  state.detail = {
    show: true,
    identityProvider,
  };
};
</script>
