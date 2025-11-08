<template>
  <div class="p-4 rounded-lg border border-gray-200 bg-gray-50">
    <div class="flex items-start gap-x-3">
      <div class="shrink-0">
        <InfoIcon class="w-5 h-5 text-blue-500 mt-0.5" />
      </div>
      <div class="flex-1 min-w-0">
        <p class="text-sm font-medium text-gray-900 mb-2">
          {{ $t("settings.sso.form.identity-provider-needed-information") }}
        </p>
        <p class="text-sm text-gray-600 mb-3">
          {{ $t("settings.sso.form.redirect-url-description") }}
        </p>

        <div class="flex items-center gap-x-3">
          <div class="flex-1">
            <label class="block text-sm font-medium text-gray-700 mb-1">
              {{ $t("settings.sso.form.redirect-url") }}
            </label>
            <div class="relative">
              <input
                :value="redirectUrl"
                readonly
                class="block w-full px-3 py-2 border border-gray-300 rounded-md bg-white text-sm font-mono text-gray-700 pr-10"
              />
              <div class="absolute inset-y-0 right-2 top-2">
                <CopyButton :content="redirectUrl" />
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { InfoIcon } from "lucide-vue-next";
import { computed } from "vue";
import { CopyButton } from "@/components/v2";
import { useActuatorV1Store } from "@/store";
import { IdentityProviderType } from "@/types/proto-es/v1/idp_service_pb";

const props = defineProps<{
  type: IdentityProviderType;
}>();

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
</script>
