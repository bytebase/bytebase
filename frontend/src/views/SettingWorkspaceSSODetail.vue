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
import { State } from "@/types/proto/v1/common";

const props = defineProps<{
  ssoName?: string;
}>();

interface LocalState {
  showFeatureModal: boolean;
}

const state = reactive<LocalState>({
  showFeatureModal: false,
});
const identityProviderStore = useIdentityProviderStore();
const isLoading = ref<boolean>(true);

watchEffect(async () => {
  if (props.ssoName) {
    isLoading.value = true;
    await identityProviderStore.getOrFetchIdentityProviderByName(
      unescape(props.ssoName)
    );
  }
  isLoading.value = false;
});

const currentIdentityProvider = computed(() => {
  return identityProviderStore.getIdentityProviderByName(props.ssoName ?? "");
});

const isDeleted = computed(() => {
  return currentIdentityProvider.value?.state == State.DELETED;
});
</script>
