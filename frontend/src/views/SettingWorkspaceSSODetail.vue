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
    </div>
    <hr />
  </div>

  <IdentityProviderCreateForm
    class="py-4"
    :identity-provider-name="selectedIdentityProvider?.name"
    @confirm="handleCreateIdentityProvider"
  />

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

interface LocalState {
  showFeatureModal: boolean;
  selectedIdentityProviderName: string;
}

const route = useRoute();
const router = useRouter();
const state = reactive<LocalState>({
  showFeatureModal: false,
  selectedIdentityProviderName: "",
});
const identityProviderStore = useIdentityProviderStore();

const identityProviderList = computed(() => {
  return identityProviderStore.identityProviderList;
});

const selectedIdentityProvider = computed(() => {
  return identityProviderList.value.find(
    (idp) => idp.name === route.params.name
  );
});

onMounted(() => {
  console.log("here");
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
};
</script>
