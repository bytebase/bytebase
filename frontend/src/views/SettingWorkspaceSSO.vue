<template>
  <div class="w-full space-y-4">
    <div class="textinfolabel">
      {{ $t("settings.sso.description") }}
      <a
        href="https://docs.bytebase.com/administration/sso/overview?source=console"
        class="normal-link inline-flex flex-row items-center"
        target="_blank"
      >
        {{ $t("common.learn-more") }}
        <heroicons-outline:external-link class="w-4 h-4" />
      </a>
    </div>
    <div class="w-full flex flex-row justify-end items-center">
      <NButton
        type="primary"
        :disabled="!allowCreateSSO"
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
    </div>
    <IdentityProviderTable
      :identity-provider-list="identityProviderList"
      :bordered="true"
    />
  </div>

  <FeatureModal
    :feature="PlanFeature.FEATURE_GOOGLE_AND_GITHUB_SSO"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { PlusIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, onMounted, reactive } from "vue";
import { useRouter } from "vue-router";
import { FeatureBadge, FeatureModal } from "@/components/FeatureGuard";
import { IdentityProviderTable } from "@/components/IdentityProvider";
import { WORKSPACE_ROUTE_SSO_CREATE } from "@/router/dashboard/workspaceRoutes";
import { featureToRef } from "@/store";
import { useIdentityProviderStore } from "@/store/modules/idp";
import type { IdentityProvider } from "@/types/proto/v1/idp_service";
import { PlanFeature } from "@/types/proto/v1/subscription_service";
import { hasWorkspacePermissionV2 } from "@/utils";

interface LocalState {
  isLoading: boolean;
  showFeatureModal: boolean;
}

const props = defineProps<{
  onClickCreate?: () => void;
  onClickView?: (sso: IdentityProvider) => void;
}>();

const router = useRouter();
const state = reactive<LocalState>({
  isLoading: true,
  showFeatureModal: false,
});
const identityProviderStore = useIdentityProviderStore();
const hasSSOFeature = featureToRef(PlanFeature.FEATURE_GOOGLE_AND_GITHUB_SSO);

const identityProviderList = computed(() => {
  return identityProviderStore.identityProviderList;
});

const allowCreateSSO = computed(() => {
  return hasWorkspacePermissionV2("bb.identityProviders.create");
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

  if (props.onClickCreate) {
    props.onClickCreate();
    return;
  }

  router.push({
    name: WORKSPACE_ROUTE_SSO_CREATE,
  });
};
</script>
