<template>
  <div
    class="w-auto max-w-full p-4 rounded flex flex-col justify-start items-start border"
  >
    <p class="textinfolabel flex flex-row justify-start items-center mb-2">
      {{ $t("settings.sso.form.identity-provider-needed-information") }}
      <ShowMoreIcon
        class="ml-1 mr-2"
        :content="$t('settings.sso.form.redirect-url-description')"
      />
    </p>
    <div class="flex flex-row justify-start items-center space-x-4">
      <p class="textlabel my-auto text-right whitespace-nowrap">
        {{ $t("settings.sso.form.redirect-url") }}
      </p>
      <div class="w-full relative break-all pr-8 text-sm">
        <div class="bg-gray-100 p-1 border border-gray-300 rounded">
          {{ redirectUrl }}
        </div>
        <button
          tabindex="-1"
          class="absolute right-0 top-1/2 -translate-y-1/2 p-1 text-control-light rounded hover:bg-gray-100"
          @click.prevent="copyRedirectUrl"
        >
          <heroicons-outline:clipboard class="w-5 h-5" />
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification, useActuatorV1Store } from "@/store";
import { IdentityProviderType } from "@/types/proto/api/v1alpha/idp_service";
import { toClipboard } from "@/utils";
import ShowMoreIcon from "../ShowMoreIcon.vue";

const props = defineProps<{
  type: IdentityProviderType;
}>();

const { t } = useI18n();

const externalUrl = computed(
  () => useActuatorV1Store().serverInfo?.externalUrl ?? ""
);

const redirectUrl = computed(() => {
  const url = externalUrl.value || window.origin;
  switch (props.type) {
    case IdentityProviderType.OAUTH2:
      return `${url}/oauth/callback`;
    case IdentityProviderType.OIDC:
      return `${url}/oidc/callback`;
    default:
      return "";
  }
});

const copyRedirectUrl = () => {
  toClipboard(redirectUrl.value).then(() => {
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: t("settings.sso.copy-redirect-url"),
    });
  });
};
</script>
