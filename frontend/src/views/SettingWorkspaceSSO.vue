<template>
  <div class="w-full space-y-4">
    <div class="w-full flex flex-row justify-between items-center">
      <div class="textinfolabel mr-4">
        {{ $t("settings.sso.description") }}
        <a
          href="https://bytebase.com/docs/administration/sso/overview?source=console"
          class="normal-link inline-flex flex-row items-center"
          target="_blank"
        >
          {{ $t("common.learn-more") }}
          <heroicons-outline:external-link class="w-4 h-4" />
        </a>
      </div>
      <div>
        <NButton
          v-if="identityProviderList.length > 0"
          type="primary"
          :disabled="!allowCreateSSO"
          @click="handleCreateSSO"
        >
          {{ $t("common.create") }}
          <FeatureBadge :feature="'bb.feature.sso'" custom-class="ml-2" />
        </NButton>
      </div>
    </div>
    <NoDataPlaceholder v-if="identityProviderList.length === 0">
      <NButton
        type="primary"
        :disabled="!allowCreateSSO"
        @click="handleCreateSSO"
      >
        {{ $t("settings.sso.create") }}
        <FeatureBadge
          :feature="'bb.feature.sso'"
          custom-class="ml-2 !text-white"
        />
      </NButton>
    </NoDataPlaceholder>
    <template v-else>
      <div class="w-full flex flex-col justify-start items-start space-y-4">
        <div
          v-for="identityProvider in identityProviderList"
          :key="identityProvider.name"
          class="w-full flex flex-col justify-start items-start border p-4"
          @click="state.selectedIdentityProviderName = identityProvider.name"
        >
          <div class="w-full flex flex-row justify-between items-center">
            <span class="truncate">{{ identityProvider.title }}</span>
            <NButton
              :disabled="!allowGetSSO"
              @click="handleViewSSO(identityProvider)"
            >
              {{ $t("common.view") }}
            </NButton>
          </div>

          <div
            class="mt-3 pt-3 border-t w-full flex flex-row justify-start items-center"
          >
            <span class="textlabel w-48 opacity-60">{{
              $t("settings.sso.form.type")
            }}</span>
            <span>{{
              identityProviderTypeToString(identityProvider.type)
            }}</span>
          </div>
          <div
            class="mt-3 pt-3 border-t w-full flex flex-row justify-start items-center"
          >
            <span class="textlabel w-48 opacity-60">{{
              $t("settings.sso.form.domain")
            }}</span>
            <span>{{ identityProvider.domain }}</span>
          </div>
        </div>
      </div>
    </template>
  </div>

  <FeatureModal
    feature="bb.feature.sso"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { computed, onMounted, reactive } from "vue";
import { useRouter } from "vue-router";
import {
  SETTING_ROUTE_WORKSPACE_SSO_CREATE,
  SETTING_ROUTE_WORKSPACE_SSO_DETAIL,
} from "@/router/dashboard/workspaceSetting";
import { featureToRef, useCurrentUserV1 } from "@/store";
import { useIdentityProviderStore } from "@/store/modules/idp";
import { IdentityProvider } from "@/types/proto/v1/idp_service";
import {
  hasWorkspacePermissionV2,
  identityProviderTypeToString,
} from "@/utils";

interface LocalState {
  showFeatureModal: boolean;
  showCreatingSSOModal: boolean;
  selectedIdentityProviderName: string;
}

const router = useRouter();
const currentUser = useCurrentUserV1();
const state = reactive<LocalState>({
  showFeatureModal: false,
  showCreatingSSOModal: false,
  selectedIdentityProviderName: "",
});
const identityProviderStore = useIdentityProviderStore();
const hasSSOFeature = featureToRef("bb.feature.sso");

const identityProviderList = computed(() => {
  return identityProviderStore.identityProviderList;
});

const allowCreateSSO = computed(() => {
  return hasWorkspacePermissionV2(
    currentUser.value,
    "bb.identityProviders.create"
  );
});

const allowGetSSO = computed(() => {
  return hasWorkspacePermissionV2(
    currentUser.value,
    "bb.identityProviders.get"
  );
});

onMounted(() => {
  identityProviderStore.fetchIdentityProviderList();
});

const handleCreateSSO = () => {
  if (!hasSSOFeature.value) {
    state.showFeatureModal = true;
    return;
  }

  router.push({
    name: SETTING_ROUTE_WORKSPACE_SSO_CREATE,
  });
};

const handleViewSSO = (identityProvider: IdentityProvider) => {
  if (!hasSSOFeature.value) {
    state.showFeatureModal = true;
    return;
  }

  router.push({
    name: SETTING_ROUTE_WORKSPACE_SSO_DETAIL,
    params: {
      ssoName: identityProvider.name,
    },
  });
};
</script>
