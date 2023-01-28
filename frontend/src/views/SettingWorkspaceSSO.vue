<template>
  <div class="w-full mt-4 space-y-4">
    <div class="w-full flex flex-row justify-between items-center">
      <div class="textinfolabel"></div>
      <div>
        <button class="btn-primary" @click="state.showCreatingSSOModal = true">
          {{ $t("common.create") }}
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
          class="w-28 h-28 px-2 border rounded flex flex-col justify-center items-center cursor-pointer"
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
    title="Create SSO"
    @close="hideCreateSSOModal"
  >
    <IdentityProviderCreateForm
      @cancel="hideCreateSSOModal"
      @confirm="handleCreateIdentityProvider"
    />
  </BBModal>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive } from "vue";
import {
  identityProviderTypeToString,
  useIdentityProviderStore,
} from "@/store/modules/idp";
import IdentityProviderCreateForm from "@/components/IdentityProviderCreateForm.vue";
import { head } from "lodash-es";
import { IdentityProvider } from "@/types/proto/v1/idp_service";

interface LocalState {
  showCreatingSSOModal: boolean;
  selectedIdentityProviderName: string;
}

const state = reactive<LocalState>({
  showCreatingSSOModal: false,
  selectedIdentityProviderName: "",
});
const identityProviderStore = useIdentityProviderStore();

const identityProviderList = computed(() => {
  return identityProviderStore.identityProviderList;
});

const selectedIdentityProvider = computed(() => {
  return identityProviderList.value.find(
    (idp) => idp.name === state.selectedIdentityProviderName
  );
});

onMounted(async () => {
  const identityProviderList =
    await identityProviderStore.fetchIdentityProviderList();
  state.selectedIdentityProviderName = head(identityProviderList)?.name || "";
});

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
