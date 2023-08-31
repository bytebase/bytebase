<template>
  <div class="w-full mt-4 space-y-4">
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
        <button
          v-if="identityProviderList.length > 0"
          class="btn-primary"
          @click="handleCreateSSO"
        >
          {{ $t("common.create") }}
          <FeatureBadge :feature="'bb.feature.sso'" custom-class="ml-2" />
        </button>
      </div>
    </div>
    <div
      v-if="identityProviderList.length === 0"
      class="w-full flex flex-col justify-center items-center"
    >
      <div
        class="w-full border-4 border-dashed border-gray-200 rounded-lg h-96 flex justify-center items-center"
      >
        <div class="text-center flex flex-col justify-center items-center">
          <img src="@/assets/illustration/no-data.webp" class="w-52" />
          <div class="mt-6">
            <button
              type="button"
              class="inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
              @click="handleCreateSSO"
            >
              {{ $t("settings.sso.create") }}
              <FeatureBadge :feature="'bb.feature.sso'" custom-class="ml-2" />
            </button>
          </div>
        </div>
      </div>
    </div>
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
            <button class="btn-normal" @click="handleViewSSO(identityProvider)">
              {{ $t("common.view") }}
            </button>
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
import { computed, onMounted, reactive } from "vue";
import { useRouter } from "vue-router";
import { featureToRef } from "@/store";
import { useIdentityProviderStore } from "@/store/modules/idp";
import { IdentityProvider } from "@/types/proto/v1/idp_service";
import { identityProviderTypeToString } from "@/utils";

interface LocalState {
  showFeatureModal: boolean;
  showCreatingSSOModal: boolean;
  selectedIdentityProviderName: string;
}

const router = useRouter();
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

onMounted(() => {
  identityProviderStore.fetchIdentityProviderList();
});

const handleCreateSSO = () => {
  if (!hasSSOFeature.value) {
    state.showFeatureModal = true;
    return;
  }
  router.push({
    name: "setting.workspace.sso.create",
  });
};

const handleViewSSO = (identityProvider: IdentityProvider) => {
  if (!hasSSOFeature.value) {
    state.showFeatureModal = true;
    return;
  }
  router.push({
    name: "setting.workspace.sso.detail",
    params: {
      ssoName: identityProvider.name,
    },
  });
};
</script>
