<template>
  <div v-if="isDeleted" class="w-full sticky top-0 mb-4">
    <ArchiveBanner />
  </div>
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
    </div>
  </div>

  <IdentityProviderCreateForm
    v-if="!isLoading"
    class="py-4"
    :identity-provider-name="currentIdentityProvider?.name"
  />
  <BBSpin v-else class="w-full h-64" />

  <FeatureModal
    feature="bb.feature.sso"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { computed, reactive, ref, watchEffect } from "vue";
import IdentityProviderCreateForm from "@/components/IdentityProviderCreateForm.vue";
import { useIdentityProviderStore } from "@/store/modules/idp";
import { ssoNamePrefix } from "@/store/modules/v1/common";
import { State } from "@/types/proto/v1/common";

const props = defineProps<{
  ssoId?: string;
}>();

interface LocalState {
  showFeatureModal: boolean;
}

const state = reactive<LocalState>({
  showFeatureModal: false,
});
const identityProviderStore = useIdentityProviderStore();
const isLoading = ref<boolean>(true);

const ssoName = computed(() => {
  if (!props.ssoId) {
    return "";
  }
  return `${ssoNamePrefix}${props.ssoId}`;
});

watchEffect(async () => {
  if (ssoName.value) {
    isLoading.value = true;
    await identityProviderStore.getOrFetchIdentityProviderByName(ssoName.value);
  }
  isLoading.value = false;
});

const currentIdentityProvider = computed(() => {
  return identityProviderStore.getIdentityProviderByName(ssoName.value);
});

const isDeleted = computed(() => {
  return currentIdentityProvider.value?.state == State.DELETED;
});
</script>
