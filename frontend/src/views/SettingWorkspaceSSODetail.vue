<template>
  <div class="w-full space-y-6">
    <div class="w-full flex flex-row justify-between items-center">
      <div class="textinfolabel mr-4">
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
    </div>

    <!-- Edit existing identity provider -->
    <IdentityProviderEditForm
      v-if="!isLoading && currentIdentityProvider"
      :identity-provider="currentIdentityProvider"
      @updated="props.onUpdated"
      @deleted="props.onDeleted"
    />

    <BBSpin v-else class="w-full h-64" />
  </div>

  <FeatureModal
    :feature="PlanFeature.FEATURE_ENTERPRISE_SSO"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { computed, reactive, ref, watchEffect } from "vue";
import { BBSpin } from "@/bbkit";
import { FeatureModal } from "@/components/FeatureGuard";
import { IdentityProviderEditForm } from "@/components/IdentityProvider";
import { useIdentityProviderStore } from "@/store/modules/idp";
import { idpNamePrefix } from "@/store/modules/v1/common";
import type { IdentityProvider } from "@/types/proto/v1/idp_service";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

const props = defineProps<{
  idpId?: string;
  onCreated?: (identityProvider: IdentityProvider) => void;
  onUpdated?: (identityProvider: IdentityProvider) => void;
  onDeleted?: () => void;
  onCanceled?: () => void;
}>();

interface LocalState {
  showFeatureModal: boolean;
}

const state = reactive<LocalState>({
  showFeatureModal: false,
});
const identityProviderStore = useIdentityProviderStore();
const isLoading = ref<boolean>(true);

const idpName = computed(() => {
  if (!props.idpId) {
    return "";
  }
  return `${idpNamePrefix}${props.idpId}`;
});

watchEffect(async () => {
  if (idpName.value) {
    isLoading.value = true;
    await identityProviderStore.getOrFetchIdentityProviderByName(idpName.value);
  }
  isLoading.value = false;
});

const currentIdentityProvider = computed(() => {
  return identityProviderStore.getIdentityProviderByName(idpName.value);
});
</script>
