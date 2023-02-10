<template>
  <div class="w-full mt-4 space-y-4">
    <div class="w-full flex flex-row justify-between items-center">
      <div class="textinfolabel mr-4">
        {{ $t("settings.sso.description") }}
        <a
          href="https://bytebase.com/docs/administration/sso?source=console"
          class="normal-link inline-flex flex-row items-center"
          target="_blank"
        >
          {{ $t("common.learn-more") }}
          <heroicons-outline:external-link class="w-4 h-4" />
        </a>
      </div>
      <div>
        <button class="btn-primary" @click="handleCreateSSO">
          {{ $t("common.create") }}
          <FeatureBadge :feature="'bb.feature.sso'" class="ml-2" />
        </button>
      </div>
    </div>
    <hr />
    <div
      v-if="identityProviderList.length === 0"
      class="w-full flex flex-col justify-center items-center"
    >
      <img src="../assets/illustration/no-data.webp" class="mt-12 w-96" />
    </div>
    <template v-else>
      <div class="w-full flex flex-col justify-start items-start space-y-4">
        <div
          v-for="identityProvider in identityProviderList"
          :key="identityProvider.name"
          class="w-full flex flex-row justify-between items-center cursor-pointer border p-4 space-x-4"
          @click="state.selectedIdentityProviderName = identityProvider.name"
        >
          <div class="flex flex-col justify-start items-start">
            <p>
              <span class="max-w-full truncate">{{
                identityProvider.title
              }}</span>
              <span class="text-sm text-gray-400 ml-1"
                >({{
                  identityProviderTypeToString(identityProvider.type)
                }})</span
              >
            </p>
            <p class="textinfolabel text-xs">
              {{ $t("common.resource-id") }}: {{ identityProvider.name }}
            </p>
          </div>
          <button class="btn-normal" @click="handleViewSSO(identityProvider)">
            {{ $t("common.view") }}
          </button>
        </div>
      </div>
    </template>
  </div>

  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.sso"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { head } from "lodash-es";
import { computed, reactive, watch, watchEffect } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useIdentityProviderStore } from "@/store/modules/idp";
import { IdentityProvider } from "@/types/proto/v1/idp_service";
import { identityProviderTypeToString } from "@/utils";
import { featureToRef } from "@/store";

interface LocalState {
  showFeatureModal: boolean;
  showCreatingSSOModal: boolean;
  selectedIdentityProviderName: string;
}

const route = useRoute();
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

watchEffect(() => {
  const hashValue = route.hash.slice(1);
  const identityProviderNameList = identityProviderList.value.map(
    (idp) => idp.name
  );
  if (identityProviderNameList.includes(hashValue)) {
    state.selectedIdentityProviderName = hashValue;
  } else {
    state.selectedIdentityProviderName = head(identityProviderNameList) || "";
  }
});

watch(
  () => state.selectedIdentityProviderName,
  () => {
    const hashValue = `#${state.selectedIdentityProviderName}`;
    if (route.hash !== hashValue) {
      router.push({
        hash: hashValue,
      });
    }
  }
);

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
      name: identityProvider.name,
    },
  });
};
</script>
