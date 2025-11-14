<template>
  <div class="flex flex-col gap-y-6">
    <!-- OAuth2 Configuration -->
    <div
      v-if="providerType === IdentityProviderType.OAUTH2"
      class="flex flex-col gap-y-6"
    >
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <label class="block text-base font-semibold text-gray-800 mb-2">
            Client ID
            <RequiredStar />
          </label>
          <BBTextField
            :value="oauth2Config.clientId || ''"
            @update:value="updateOAuth2Config('clientId', $event)"
            required
            size="large"
            class="w-full text-base"
            placeholder="e.g. 6655asd77895265aa110ac0d3"
          />
        </div>

        <div>
          <label class="block text-base font-semibold text-gray-800 mb-2">
            Client Secret
            <RequiredStar v-if="!isEditMode" />
          </label>
          <BBTextField
            :value="oauth2Config.clientSecret || ''"
            @update:value="updateOAuth2Config('clientSecret', $event)"
            :required="!isEditMode"
            size="large"
            type="password"
            class="w-full text-base"
            :placeholder="
              isEditMode && !oauth2Config.clientSecret
                ? 'Leave empty to keep existing secret'
                : 'e.g. 5bbezxc3972ca304de70c5d70a6aa932asd8'
            "
          />
        </div>
      </div>

      <div>
        <label class="block text-base font-semibold text-gray-800 mb-2">
          {{ $t("settings.sso.form.auth-url") }}
          <RequiredStar />
        </label>
        <BBTextField
          :value="oauth2Config.authUrl || ''"
          @update:value="updateOAuth2Config('authUrl', $event)"
          required
          size="large"
          class="w-full text-base"
          :placeholder="$t('settings.sso.form.auth-url-placeholder')"
        />
        <p class="text-sm text-gray-600 mt-1">
          {{ $t("settings.sso.form.auth-url-description") }}
        </p>
      </div>

      <div>
        <label class="block text-base font-semibold text-gray-800 mb-2">
          {{ $t("settings.sso.form.token-url") }}
          <RequiredStar />
        </label>
        <BBTextField
          :value="oauth2Config.tokenUrl || ''"
          @update:value="updateOAuth2Config('tokenUrl', $event)"
          required
          size="large"
          class="w-full text-base"
          :placeholder="$t('settings.sso.form.token-url-placeholder')"
        />
        <p class="text-sm text-gray-600 mt-1">
          {{ $t("settings.sso.form.token-url-description") }}
        </p>
      </div>

      <div>
        <label class="block text-base font-semibold text-gray-800 mb-2">
          {{ $t("settings.sso.form.user-info-url") }}
          <RequiredStar />
        </label>
        <BBTextField
          :value="oauth2Config.userInfoUrl || ''"
          @update:value="updateOAuth2Config('userInfoUrl', $event)"
          required
          size="large"
          class="w-full text-base"
          :placeholder="$t('settings.sso.form.user-info-url-placeholder')"
        />
        <p class="text-sm text-gray-600 mt-1">
          {{ $t("settings.sso.form.user-info-url-description") }}
        </p>
      </div>

      <div>
        <label class="block text-base font-semibold text-gray-800 mb-2">
          {{ $t("settings.sso.form.scopes") }}
          <RequiredStar />
        </label>
        <BBTextField
          :value="scopesString || ''"
          @update:value="$emit('update:scopes-string', $event)"
          required
          size="large"
          class="w-full text-base"
          :placeholder="$t('settings.sso.form.scopes-placeholder')"
        />
        <p class="text-sm text-gray-600 mt-1">
          {{ $t("settings.sso.form.scopes-description") }}
        </p>
      </div>

      <div>
        <label class="block text-base font-semibold text-gray-800 mb-3">
          {{ $t("settings.sso.form.authentication-style") }}
          <RequiredStar />
        </label>
        <NRadioGroup
          :value="oauth2Config.authStyle"
          @update:value="updateOAuth2Config('authStyle', $event)"
          size="large"
          class="flex flex-col gap-y-3"
        >
          <NRadio
            :value="OAuth2AuthStyle.IN_PARAMS"
            class="text-base font-medium text-gray-800"
          >
            <div class="flex flex-col gap-y-1">
              <div class="font-medium">
                {{ $t("settings.sso.form.in-parameters") }}
              </div>
              <div class="text-sm text-gray-600">
                {{ $t("settings.sso.form.in-parameters-description") }}
              </div>
            </div>
          </NRadio>
          <NRadio
            :value="OAuth2AuthStyle.IN_HEADER"
            class="text-base font-medium text-gray-800"
          >
            <div class="flex flex-col gap-y-1">
              <div class="font-medium">
                {{ $t("settings.sso.form.in-header") }}
              </div>
              <div class="text-sm text-gray-600">
                {{ $t("settings.sso.form.in-header-description") }}
              </div>
            </div>
          </NRadio>
        </NRadioGroup>
      </div>

      <div>
        <label class="block text-base font-semibold text-gray-800 mb-3">
          {{ $t("settings.sso.form.security-options") }}
        </label>
        <NCheckbox
          :checked="oauth2Config.skipTlsVerify"
          @update:checked="updateOAuth2Config('skipTlsVerify', $event)"
          size="large"
          class="text-base font-medium text-gray-800"
        >
          {{ $t("settings.sso.form.skip-tls-verification") }}
        </NCheckbox>
        <p class="text-sm text-gray-600 mt-1 ml-6">
          {{ $t("settings.sso.form.skip-tls-warning") }}
        </p>
      </div>
    </div>

    <!-- OIDC Configuration -->
    <div
      v-else-if="providerType === IdentityProviderType.OIDC"
      class="flex flex-col gap-y-6"
    >
      <div>
        <label class="block text-base font-semibold text-gray-800 mb-2">
          Issuer URL
          <RequiredStar />
        </label>
        <BBTextField
          :value="oidcConfig.issuer || ''"
          @update:value="updateOIDCConfig('issuer', $event)"
          required
          size="large"
          class="w-full text-base"
          :placeholder="$t('settings.sso.form.issuer-url-placeholder')"
        />
        <p class="text-sm text-gray-600 mt-1">
          {{ $t("settings.sso.form.issuer-url-description") }}
        </p>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <label class="block text-base font-semibold text-gray-800 mb-2">
            Client ID
            <RequiredStar />
          </label>
          <BBTextField
            :value="oidcConfig.clientId || ''"
            @update:value="updateOIDCConfig('clientId', $event)"
            required
            size="large"
            class="w-full text-base"
            placeholder="e.g. 6655asd77895265aa110ac0d3"
          />
        </div>

        <div>
          <label class="block text-base font-semibold text-gray-800 mb-2">
            Client Secret
            <RequiredStar v-if="!isEditMode" />
          </label>
          <BBTextField
            :value="oidcConfig.clientSecret || ''"
            @update:value="updateOIDCConfig('clientSecret', $event)"
            :required="!isEditMode"
            size="large"
            type="password"
            class="w-full text-base"
            :placeholder="
              isEditMode && !oidcConfig.clientSecret
                ? 'Leave empty to keep existing secret'
                : 'e.g. 5bbezxc3972ca304de70c5d70a6aa932asd8'
            "
          />
        </div>
      </div>

      <div>
        <label class="block text-base font-semibold text-gray-800 mb-2">
          {{ $t("settings.sso.form.scopes") }}
          <RequiredStar />
        </label>
        <BBTextField
          :value="scopesString"
          @update:value="$emit('update:scopes-string', $event)"
          required
          size="large"
          class="w-full text-base"
          :placeholder="$t('settings.sso.form.scopes-placeholder-oidc')"
        />
        <p class="text-sm text-gray-600 mt-1">
          {{ $t("settings.sso.form.openid-scopes-description") }}
        </p>
      </div>

      <div>
        <label class="block text-base font-semibold text-gray-800 mb-3">
          {{ $t("settings.sso.form.authentication-style") }}
          <RequiredStar />
        </label>
        <NRadioGroup
          :value="oidcConfig.authStyle"
          @update:value="updateOIDCConfig('authStyle', $event)"
          size="large"
          class="flex flex-col gap-y-3"
        >
          <NRadio
            :value="OAuth2AuthStyle.IN_PARAMS"
            class="text-base font-medium text-gray-800"
          >
            <div class="flex flex-col gap-y-1">
              <div class="font-medium">
                {{ $t("settings.sso.form.in-parameters") }}
              </div>
              <div class="text-sm text-gray-600">
                {{ $t("settings.sso.form.in-parameters-description") }}
              </div>
            </div>
          </NRadio>
          <NRadio
            :value="OAuth2AuthStyle.IN_HEADER"
            class="text-base font-medium text-gray-800"
          >
            <div class="flex flex-col gap-y-1">
              <div class="font-medium">
                {{ $t("settings.sso.form.in-header") }}
              </div>
              <div class="text-sm text-gray-600">
                {{ $t("settings.sso.form.in-header-description") }}
              </div>
            </div>
          </NRadio>
        </NRadioGroup>
      </div>

      <div>
        <label class="block text-base font-semibold text-gray-800 mb-3">
          {{ $t("settings.sso.form.security-options") }}
        </label>
        <NCheckbox
          :checked="oidcConfig.skipTlsVerify"
          @update:checked="updateOIDCConfig('skipTlsVerify', $event)"
          size="large"
          class="text-base font-medium text-gray-800"
        >
          {{ $t("settings.sso.form.skip-tls-verification") }}
        </NCheckbox>
        <p class="text-sm text-gray-600 mt-1 ml-6">
          {{ $t("settings.sso.form.skip-tls-warning") }}
        </p>
      </div>
    </div>

    <!-- LDAP Configuration -->
    <div
      v-else-if="providerType === IdentityProviderType.LDAP"
      class="flex flex-col gap-y-6"
    >
      <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div class="md:col-span-2">
          <label class="block text-base font-semibold text-gray-800 mb-2">
            Host
            <RequiredStar />
          </label>
          <BBTextField
            :value="ldapConfig.host || ''"
            @update:value="updateLDAPConfig('host', $event)"
            required
            size="large"
            class="w-full text-base"
            :placeholder="$t('settings.sso.form.host-placeholder')"
          />
        </div>

        <div>
          <label class="block text-base font-semibold text-gray-800 mb-2">
            Port
            <RequiredStar />
          </label>
          <NInputNumber
            :value="ldapConfig.port"
            @update:value="updateLDAPConfig('port', $event)"
            size="large"
            class="w-full text-base"
            placeholder="389"
            :min="1"
            :max="65535"
          />
        </div>
      </div>

      <div>
        <label class="block text-base font-semibold text-gray-800 mb-2">
          {{ $t("settings.sso.form.bind-dn") }}
          <RequiredStar />
        </label>
        <BBTextField
          :value="ldapConfig.bindDn || ''"
          @update:value="updateLDAPConfig('bindDn', $event)"
          required
          size="large"
          class="w-full text-base"
          :placeholder="$t('settings.sso.form.bind-dn-placeholder')"
        />
        <p class="text-sm text-gray-600 mt-1">
          {{ $t("settings.sso.form.bind-dn-description") }}
        </p>
      </div>

      <div>
        <label class="block text-base font-semibold text-gray-800 mb-2">
          {{ $t("settings.sso.form.bind-password") }}
          <RequiredStar />
        </label>
        <BBTextField
          :value="ldapConfig.bindPassword || ''"
          @update:value="updateLDAPConfig('bindPassword', $event)"
          required
          size="large"
          type="password"
          class="w-full text-base"
          placeholder="••••••••"
        />
      </div>

      <div>
        <label class="block text-base font-semibold text-gray-800 mb-2">
          {{ $t("settings.sso.form.base-dn") }}
          <RequiredStar />
        </label>
        <BBTextField
          :value="ldapConfig.baseDn || ''"
          @update:value="updateLDAPConfig('baseDn', $event)"
          required
          size="large"
          class="w-full text-base"
          :placeholder="$t('settings.sso.form.base-dn-placeholder')"
        />
        <p class="text-sm text-gray-600 mt-1">
          {{ $t("settings.sso.form.base-dn-description") }}
        </p>
      </div>

      <div>
        <label class="block text-base font-semibold text-gray-800 mb-2">
          {{ $t("settings.sso.form.user-filter") }}
          <RequiredStar />
        </label>
        <BBTextField
          :value="ldapConfig.userFilter || ''"
          @update:value="updateLDAPConfig('userFilter', $event)"
          required
          size="large"
          class="w-full text-base"
          :placeholder="$t('settings.sso.form.user-filter-placeholder')"
        />
        <p class="text-sm text-gray-600 mt-1">
          {{ $t("settings.sso.form.user-filter-description") }}
        </p>
      </div>

      <div>
        <label class="block text-base font-semibold text-gray-800 mb-3">
          {{ $t("settings.sso.form.security-protocol") }}
          <RequiredStar />
        </label>
        <NRadioGroup
          :value="ldapConfig.securityProtocol"
          @update:value="updateLDAPConfig('securityProtocol', $event)"
          size="large"
          class="flex flex-col gap-y-3"
        >
          <NRadio
            :value="LDAPIdentityProviderConfig_SecurityProtocol.START_TLS"
            class="text-base font-medium text-gray-800"
          >
            <div class="flex flex-col gap-y-1">
              <div class="font-medium">
                {{ $t("settings.sso.form.starttls") }}
              </div>
              <div class="text-sm text-gray-600">
                {{ $t("settings.sso.form.starttls-description") }}
              </div>
            </div>
          </NRadio>
          <NRadio
            :value="LDAPIdentityProviderConfig_SecurityProtocol.LDAPS"
            class="text-base font-medium text-gray-800"
          >
            <div class="flex flex-col gap-y-1">
              <div class="font-medium">{{ $t("settings.sso.form.ldaps") }}</div>
              <div class="text-sm text-gray-600">
                {{ $t("settings.sso.form.ldaps-description") }}
              </div>
            </div>
          </NRadio>
          <NRadio
            :value="
              LDAPIdentityProviderConfig_SecurityProtocol.SECURITY_PROTOCOL_UNSPECIFIED
            "
            class="text-base font-medium text-gray-800"
          >
            <div class="flex flex-col gap-y-1">
              <div class="font-medium">{{ $t("settings.sso.form.none") }}</div>
              <div class="text-sm text-gray-600">
                {{ $t("settings.sso.form.none-description") }}
              </div>
            </div>
          </NRadio>
        </NRadioGroup>
      </div>

      <div>
        <label class="block text-base font-semibold text-gray-800 mb-3">
          {{ $t("settings.sso.form.security-options") }}
        </label>
        <NCheckbox
          :checked="ldapConfig.skipTlsVerify"
          @update:checked="updateLDAPConfig('skipTlsVerify', $event)"
          size="large"
          class="text-base font-medium text-gray-800"
        >
          {{ $t("settings.sso.form.skip-tls-verification") }}
        </NCheckbox>
        <p class="text-sm text-gray-600 mt-1 ml-6">
          {{ $t("settings.sso.form.skip-tls-warning") }}
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { NCheckbox, NInputNumber, NRadio, NRadioGroup } from "naive-ui";
import { computed, watch } from "vue";
import { BBTextField } from "@/bbkit";
import RequiredStar from "@/components/RequiredStar.vue";
import {
  type IdentityProvider,
  IdentityProviderType,
  type LDAPIdentityProviderConfig,
  LDAPIdentityProviderConfig_SecurityProtocol,
  OAuth2AuthStyle,
  type OAuth2IdentityProviderConfig,
  type OIDCIdentityProviderConfig,
} from "@/types/proto-es/v1/idp_service_pb";

interface Props {
  identityProvider: IdentityProvider;
  providerType: IdentityProviderType;
  configForOauth2: OAuth2IdentityProviderConfig;
  configForOidc: OIDCIdentityProviderConfig;
  configForLdap: LDAPIdentityProviderConfig;
  scopesString: string;
  isEditMode?: boolean;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  "update:config-for-oauth2": [config: OAuth2IdentityProviderConfig];
  "update:config-for-oidc": [config: OIDCIdentityProviderConfig];
  "update:config-for-ldap": [config: LDAPIdentityProviderConfig];
  "update:scopes-string": [scopes: string];
}>();

// Create computed properties to ensure reactivity
const oauth2Config = computed(() => props.configForOauth2);
const oidcConfig = computed(() => props.configForOidc);
const ldapConfig = computed(() => props.configForLdap);

// Watch for OAuth2 config changes to ensure reactivity
watch(
  () => props.configForOauth2,
  () => {
    // Trigger reactivity when props change
  },
  { deep: true }
);

// Helper methods to update configurations
const updateOAuth2Config = <T extends keyof OAuth2IdentityProviderConfig>(
  key: T,
  value: OAuth2IdentityProviderConfig[T]
) => {
  emit("update:config-for-oauth2", {
    ...props.configForOauth2,
    [key]: value,
  });
};

const updateOIDCConfig = <T extends keyof OIDCIdentityProviderConfig>(
  key: T,
  value: OIDCIdentityProviderConfig[T]
) => {
  emit("update:config-for-oidc", {
    ...props.configForOidc,
    [key]: value,
  });
};

const updateLDAPConfig = <T extends keyof LDAPIdentityProviderConfig>(
  key: T,
  value: LDAPIdentityProviderConfig[T] | null
) => {
  emit("update:config-for-ldap", {
    ...props.configForLdap,
    [key]: value,
  });
};
</script>
