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
      <div class="w-full flex flex-row justify-start items-start space-x-4">
        <div
          v-for="identityProvider in identityProviderList"
          :key="identityProvider.name"
          class="w-28 h-28 px-2 border rounded-md flex flex-col justify-center items-center cursor-pointer"
          @click="state.selectedIdentityProviderName = identityProvider.name"
        >
          <span class="max-w-full truncate">{{ identityProvider.title }}</span>
          <span class="text-sm text-gray-400"
            >({{ identityProviderTypeToString(identityProvider.type) }})</span
          >
          <input
            type="radio"
            class="btn mt-2"
            :checked="
              state.selectedIdentityProviderName === identityProvider.name
            "
          />
        </div>
      </div>
    </template>

    <!-- Edit form -->
    <div v-if="selectedIdentityProvider" class="pb-6">
      <IdentityProviderCreateForm
        :key="selectedIdentityProvider.name"
        :identity-provider-name="selectedIdentityProvider.name"
        @delete="handleDeleteIdentityProvider"
      />
    </div>
  </div>

  <BBModal
    v-if="state.showCreatingSSOModal"
    :title="$t('settings.sso.create')"
    @close="hideCreateSSOModal"
  >
    <IdentityProviderCreateForm
      @cancel="hideCreateSSOModal"
      @confirm="handleCreateIdentityProvider"
    />
  </BBModal>

  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.sso"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { head } from "lodash-es";
import { computed, onMounted, reactive, watch, watchEffect } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useIdentityProviderStore } from "@/store/modules/idp";
import IdentityProviderCreateForm from "@/components/IdentityProviderCreateForm.vue";
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

const selectedIdentityProvider = computed(() => {
  return identityProviderList.value.find(
    (idp) => idp.name === state.selectedIdentityProviderName
  );
});

onMounted(async () => {
  await identityProviderStore.fetchIdentityProviderList();
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
  state.showCreatingSSOModal = true;
};

const hideCreateSSOModal = () => {
  state.showCreatingSSOModal = false;
};

const handleDeleteIdentityProvider = async (
  identityProvider: IdentityProvider
) => {
  await identityProviderStore.deleteIdentityProvider(identityProvider);
  if (state.selectedIdentityProviderName === identityProvider.name) {
    state.selectedIdentityProviderName =
      head(identityProviderList.value)?.name || "";
  }
};

const handleCreateIdentityProvider = async (
  identityProvider: IdentityProvider
) => {
  state.selectedIdentityProviderName = identityProvider.name;
  hideCreateSSOModal();
};
</script>
