<template>
  <div class="w-full flex flex-col gap-y-4">
    <div class="textinfolabel">
      {{ $t("settings.sso.description") }}
      <LearnMoreLink
        url="https://docs.bytebase.com/administration/sso/overview?source=console"
      />
    </div>
    <div class="w-full flex flex-row justify-end items-center">
      <PermissionGuardWrapper
        v-slot="slotProps"
        :permissions="['bb.identityProviders.create']"
      >
        <NButton
          type="primary"
          :disabled="slotProps.disabled"
          @click="handleCreateSSO"
        >
          <template #icon>
            <PlusIcon class="h-4 w-4" />
            <FeatureBadge
              :feature="PlanFeature.FEATURE_GOOGLE_AND_GITHUB_SSO"
              class="text-white"
            />
          </template>
          {{ $t("settings.sso.create") }}
        </NButton>
      </PermissionGuardWrapper>
    </div>
    <IdentityProviderTable
      :identity-provider-list="identityProviderList"
      :bordered="true"
      :loading="state.isLoading"
    />
  </div>

  <FeatureModal
    :feature="PlanFeature.FEATURE_GOOGLE_AND_GITHUB_SSO"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />

  <IdentityProviderCreateWizard
    :show="state.showCreateDrawer"
    @update:show="state.showCreateDrawer = $event"
    @created="handleProviderCreated"
  />
</template>

<script lang="ts" setup>
import { PlusIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, onMounted, reactive } from "vue";
import { useRouter } from "vue-router";
import { FeatureBadge, FeatureModal } from "@/components/FeatureGuard";
import {
  IdentityProviderCreateWizard,
  IdentityProviderTable,
} from "@/components/IdentityProvider";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { WORKSPACE_ROUTE_IDENTITY_PROVIDER_DETAIL } from "@/router/dashboard/workspaceRoutes";
import { featureToRef, getIdentityProviderResourceId } from "@/store";
import { useIdentityProviderStore } from "@/store/modules/idp";
import type { IdentityProvider } from "@/types/proto-es/v1/idp_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

interface LocalState {
  isLoading: boolean;
  showFeatureModal: boolean;
  showCreateDrawer: boolean;
}

const router = useRouter();
const state = reactive<LocalState>({
  isLoading: true,
  showFeatureModal: false,
  showCreateDrawer: false,
});
const identityProviderStore = useIdentityProviderStore();
const hasSSOFeature = featureToRef(PlanFeature.FEATURE_GOOGLE_AND_GITHUB_SSO);

const identityProviderList = computed(() => {
  return identityProviderStore.identityProviderList;
});

onMounted(async () => {
  await identityProviderStore.fetchIdentityProviderList();
  state.isLoading = false;
});

const handleCreateSSO = () => {
  if (!hasSSOFeature.value) {
    state.showFeatureModal = true;
    return;
  }
  state.showCreateDrawer = true;
};

const handleProviderCreated = (provider: IdentityProvider) => {
  state.showCreateDrawer = false;
  router.replace({
    name: WORKSPACE_ROUTE_IDENTITY_PROVIDER_DETAIL,
    params: {
      idpId: getIdentityProviderResourceId(provider.name),
    },
  });
};
</script>
