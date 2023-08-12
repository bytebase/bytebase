<template>
  <div class="w-full sticky top-0">
    <ArchiveBanner v-if="isDeleted" />
  </div>
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
    </div>
  </div>

  <IdentityProviderCreateForm
    class="py-4"
    :identity-provider-name="currentIdentityProvider?.name"
  />

  <FeatureModal
    feature="bb.feature.sso"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import { useRoute } from "vue-router";
import IdentityProviderCreateForm from "@/components/IdentityProviderCreateForm.vue";
import { useIdentityProviderStore } from "@/store/modules/idp";
import { State } from "@/types/proto/v1/common";

interface LocalState {
  showFeatureModal: boolean;
}

const route = useRoute();
const state = reactive<LocalState>({
  showFeatureModal: false,
});
const identityProviderStore = useIdentityProviderStore();

const currentIdentityProvider = computed(() => {
  return identityProviderStore.getIdentityProviderByName(
    (route.params.ssoName as string) || ""
  );
});

const isDeleted = computed(() => {
  return currentIdentityProvider.value?.state == State.DELETED;
});
</script>
