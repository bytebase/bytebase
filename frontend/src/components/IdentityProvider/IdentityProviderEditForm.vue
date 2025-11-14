<template>
  <div class="w-full flex flex-col gap-y-0 pt-4">
    <div class="divide-y divide-block-border">
      <!-- Basic Information Section -->
      <div class="pb-6 lg:flex">
        <div class="text-left lg:w-1/4">
          <h1 class="text-3xl font-bold text-gray-900">
            {{ $t("common.general") }}
          </h1>
        </div>
        <div class="flex-1 mt-4 lg:px-4 lg:mt-0 flex flex-col gap-y-6">
          <div>
            <p class="text-base font-semibold text-gray-800 mb-2">
              {{ $t("common.type") }}
            </p>
            <div class="flex items-center gap-x-3 p-3 bg-gray-50 rounded-md">
              <component
                :is="getProviderIcon(localIdentityProvider.type)"
                class="w-5 h-5 text-gray-600"
              />
              <span class="text-base font-medium text-gray-800">
                {{ identityProviderTypeToString(localIdentityProvider.type) }}
              </span>
            </div>
            <p class="text-sm text-gray-600 mt-1">
              {{ $t("settings.sso.form.provider-type-readonly-hint") }}
            </p>
          </div>

          <div>
            <p class="text-base font-semibold text-gray-800 mb-2">
              {{ $t("settings.sso.form.name") }}
              <RequiredStar />
            </p>
            <BBTextField
              v-model:value="localIdentityProvider.title"
              required
              :disabled="!allowEdit"
              size="large"
              class="w-full text-base"
              :placeholder="$t('settings.sso.form.name-description')"
              :maxlength="200"
            />
            <ResourceIdField
              resource-type="idp"
              :readonly="true"
              :value="resourceId"
              class="mt-1"
            />
          </div>

          <div>
            <p class="text-base font-semibold text-gray-800 mb-2">
              {{ $t("settings.sso.form.domain") }}
            </p>
            <BBTextField
              v-model:value="localIdentityProvider.domain"
              :disabled="!allowEdit"
              size="large"
              class="w-full text-base"
              :placeholder="$t('settings.sso.form.domain-description')"
            />
            <p class="text-sm text-gray-600 mt-1">
              {{ $t("settings.sso.form.domain-optional-hint") }}
            </p>
          </div>
        </div>
      </div>

      <!-- Configuration Section -->
      <div class="py-6 lg:flex">
        <div class="text-left lg:w-1/4">
          <h1 class="text-3xl font-bold text-gray-900">
            {{ $t("settings.sso.form.configuration") }}
          </h1>
          <p class="text-base text-gray-600 mt-3">
            {{ $t("settings.sso.form.configuration-description") }}
          </p>
        </div>
        <div class="flex-1 mt-4 lg:px-4 lg:mt-0">
          <IdentityProviderForm
            :identity-provider="localIdentityProvider"
            :provider-type="localIdentityProvider.type"
            :config-for-oauth2="configForOAuth2"
            :config-for-oidc="configForOIDC"
            :config-for-ldap="configForLDAP"
            :scopes-string="scopesStringOfConfig"
            :is-edit-mode="true"
            @update:config-for-oauth2="configForOAuth2 = $event"
            @update:config-for-oidc="configForOIDC = $event"
            @update:config-for-ldap="configForLDAP = $event"
            @update:scopes-string="scopesStringOfConfig = $event"
          />
        </div>
      </div>

      <!-- User Information Mapping Section -->
      <div class="py-6 lg:flex">
        <div class="text-left lg:w-1/4">
          <h1 class="text-3xl font-bold text-gray-900">
            {{ $t("settings.sso.form.user-information-mapping") }}
          </h1>
          <p class="text-base text-gray-600 mt-3">
            {{ $t("settings.sso.form.user-information-mapping-description") }}
            <LearnMoreLink
              class="ml-1"
              url="https://docs.bytebase.com/administration/sso/oauth2#user-information-field-mapping?source=console"
            />
          </p>
        </div>
        <div class="flex-1 mt-4 lg:px-4 lg:mt-0 flex flex-col gap-y-6">
          <div class="grid grid-cols-[256px_1fr] gap-4 items-center">
            <BBTextField
              v-model:value="fieldMapping.identifier"
              :disabled="!allowEdit"
              size="large"
              class="w-full text-base"
              :placeholder="$t('settings.sso.form.identifier-placeholder')"
            />
            <div class="flex flex-row justify-start items-center text-base">
              <ArrowRightIcon class="mx-2 h-auto w-5 text-gray-400" />
              <p
                class="flex flex-row justify-start items-center font-semibold text-gray-800"
              >
                {{ $t("settings.sso.form.identifier") }}
                <RequiredStar />
                <NTooltip>
                  <template #trigger>
                    <InfoIcon class="ml-1 w-4 h-auto text-blue-500" />
                  </template>
                  {{ $t("settings.sso.form.identifier-tips") }}
                </NTooltip>
              </p>
            </div>
          </div>

          <div class="grid grid-cols-[256px_1fr] gap-4 items-center">
            <BBTextField
              v-model:value="fieldMapping.displayName"
              :disabled="!allowEdit"
              size="large"
              class="w-full text-base"
              :placeholder="$t('settings.sso.form.display-name-placeholder')"
            />
            <div class="flex flex-row justify-start items-center text-base">
              <ArrowRightIcon class="mx-2 h-auto w-5 text-gray-400" />
              <p class="font-semibold text-gray-800">
                {{ $t("settings.sso.form.display-name") }}
              </p>
            </div>
          </div>

          <div class="grid grid-cols-[256px_1fr] gap-4 items-center">
            <BBTextField
              v-model:value="fieldMapping.phone"
              :disabled="!allowEdit"
              size="large"
              class="w-full text-base"
              :placeholder="$t('settings.sso.form.phone-placeholder')"
            />
            <div class="flex flex-row justify-start items-center text-base">
              <ArrowRightIcon class="mx-2 h-auto w-5 text-gray-400" />
              <p class="font-semibold text-gray-800">
                {{ $t("settings.sso.form.phone") }}
              </p>
            </div>
          </div>

          <div
            v-if="localIdentityProvider.type === IdentityProviderType.OIDC"
            class="grid grid-cols-[256px_1fr] gap-4 items-center"
          >
            <BBTextField
              v-model:value="fieldMapping.groups"
              :disabled="!allowEdit"
              size="large"
              class="w-full text-base"
              :placeholder="$t('settings.sso.form.groups-placeholder')"
            />
            <div class="flex flex-row justify-start items-center text-base">
              <ArrowRightIcon class="mx-2 h-auto w-5 text-gray-400" />
              <p class="font-semibold text-gray-800">
                {{ $t("settings.sso.form.groups") }}
              </p>
            </div>
          </div>

          <div>
            <TestConnection
              :disabled="!allowTestConnection"
              :size="'large'"
              :is-creating="false"
              :idp="buildUpdatedIdentityProvider()"
            />
          </div>
        </div>
      </div>

      <!-- Actions Section -->
      <div class="py-6">
        <div class="flex flex-row justify-between items-center">
          <BBButtonConfirm
            :type="'DELETE'"
            :button-text="$t('settings.sso.delete')"
            :ok-text="$t('common.delete')"
            :confirm-title="$t('settings.sso.delete')"
            :confirm-description="t('identity-provider.delete-warning')"
            :require-confirm="true"
            @confirm="handleDelete"
          />
          <div class="gap-x-3 flex flex-row justify-end items-center">
            <NButton v-if="hasChanges" @click="handleDiscard" class="text-base">
              {{ $t("common.discard-changes") }}
            </NButton>
            <NButton
              type="primary"
              :disabled="!canUpdate"
              :loading="isUpdating"
              @click="handleUpdate"
              class="text-base"
            >
              {{ $t("common.update") }}
            </NButton>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { create as createProto } from "@bufbuild/protobuf";
import { cloneDeep, isEqual } from "lodash-es";
import {
  ArrowRightIcon,
  DatabaseIcon,
  InfoIcon,
  KeyIcon,
  ShieldCheckIcon,
} from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBButtonConfirm, BBTextField } from "@/bbkit";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import { ResourceIdField } from "@/components/v2";
import { WORKSPACE_ROUTE_IDENTITY_PROVIDERS } from "@/router/dashboard/workspaceRoutes";
import { pushNotification } from "@/store";
import { useIdentityProviderStore } from "@/store/modules/idp";
import { getIdentityProviderResourceId } from "@/store/modules/v1/common";
import type {
  FieldMapping,
  IdentityProvider,
  LDAPIdentityProviderConfig,
  OAuth2IdentityProviderConfig,
  OIDCIdentityProviderConfig,
} from "@/types/proto-es/v1/idp_service_pb";
import {
  FieldMappingSchema,
  IdentityProviderConfigSchema,
  IdentityProviderSchema,
  IdentityProviderType,
  LDAPIdentityProviderConfigSchema,
  OAuth2IdentityProviderConfigSchema,
  OIDCIdentityProviderConfigSchema,
} from "@/types/proto-es/v1/idp_service_pb";
import {
  hasWorkspacePermissionV2,
  identityProviderTypeToString,
} from "@/utils";
import IdentityProviderForm from "./IdentityProviderForm.vue";
import TestConnection from "./TestConnection.vue";

interface Props {
  identityProvider: IdentityProvider;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  (event: "updated", identityProvider: IdentityProvider): void;
  (event: "deleted"): void;
}>();

const { t } = useI18n();
const router = useRouter();
const identityProviderStore = useIdentityProviderStore();

// State
const isUpdating = ref(false);
const localIdentityProvider = ref<IdentityProvider>(
  cloneDeep(props.identityProvider)
);
const configForOAuth2 = ref<OAuth2IdentityProviderConfig>(
  createProto(
    OAuth2IdentityProviderConfigSchema,
    props.identityProvider.config?.config?.case === "oauth2Config"
      ? props.identityProvider.config.config.value
      : {}
  )
);
const configForOIDC = ref<OIDCIdentityProviderConfig>(
  createProto(
    OIDCIdentityProviderConfigSchema,
    props.identityProvider.config?.config?.case === "oidcConfig"
      ? props.identityProvider.config.config.value
      : {}
  )
);
const configForLDAP = ref<LDAPIdentityProviderConfig>(
  createProto(
    LDAPIdentityProviderConfigSchema,
    props.identityProvider.config?.config?.case === "ldapConfig"
      ? props.identityProvider.config.config.value
      : {}
  )
);
const scopesStringOfConfig = ref<string>("");
const fieldMapping = reactive<FieldMapping>(
  createProto(FieldMappingSchema, {})
);

// Computed
const resourceId = computed(() => {
  return getIdentityProviderResourceId(localIdentityProvider.value.name);
});

const allowEdit = computed(() => {
  return hasWorkspacePermissionV2("bb.identityProviders.update");
});

const hasChanges = computed(() => {
  const current = buildUpdatedIdentityProvider();
  return !isEqual(current, props.identityProvider);
});

// Track if secrets have been modified
const isClientSecretModified = ref(false);
const isBindPasswordModified = ref(false);

const isFormValid = computed(() => {
  if (!localIdentityProvider.value.title) return false;
  if (!fieldMapping.identifier) return false;

  if (localIdentityProvider.value.type === IdentityProviderType.OAUTH2) {
    // For edit mode, client secret is optional if not modified
    const isClientSecretValid = isClientSecretModified.value
      ? !!configForOAuth2.value.clientSecret
      : true;

    return !!(
      configForOAuth2.value.clientId &&
      isClientSecretValid &&
      configForOAuth2.value.authUrl &&
      configForOAuth2.value.tokenUrl &&
      configForOAuth2.value.userInfoUrl
    );
  } else if (localIdentityProvider.value.type === IdentityProviderType.OIDC) {
    // For edit mode, client secret is optional if not modified
    const isClientSecretValid = isClientSecretModified.value
      ? !!configForOIDC.value.clientSecret
      : true;

    return !!(
      configForOIDC.value.clientId &&
      isClientSecretValid &&
      configForOIDC.value.issuer
    );
  } else if (localIdentityProvider.value.type === IdentityProviderType.LDAP) {
    // For edit mode, bind password is optional if not modified
    const isBindPasswordValid = isBindPasswordModified.value
      ? !!configForLDAP.value.bindPassword
      : true;

    return !!(
      configForLDAP.value.host &&
      configForLDAP.value.port &&
      configForLDAP.value.bindDn &&
      isBindPasswordValid &&
      configForLDAP.value.baseDn &&
      configForLDAP.value.userFilter
    );
  }

  return false;
});

const canUpdate = computed(() => {
  return allowEdit.value && hasChanges.value && isFormValid.value;
});

const allowTestConnection = computed(() => {
  return isFormValid.value;
});

// Methods
const getProviderIcon = (type: IdentityProviderType) => {
  switch (type) {
    case IdentityProviderType.OAUTH2:
      return KeyIcon;
    case IdentityProviderType.OIDC:
      return ShieldCheckIcon;
    case IdentityProviderType.LDAP:
      return DatabaseIcon;
    default:
      return KeyIcon;
  }
};

const buildUpdatedIdentityProvider = (): IdentityProvider => {
  const result = createProto(IdentityProviderSchema, {
    ...localIdentityProvider.value,
    config: createProto(IdentityProviderConfigSchema, {}),
  });

  if (localIdentityProvider.value.type === IdentityProviderType.OAUTH2) {
    const oauth2Config = {
      ...configForOAuth2.value,
      scopes: scopesStringOfConfig.value.split(" ").filter(Boolean),
      fieldMapping: createProto(FieldMappingSchema, fieldMapping),
    };

    result.config = createProto(IdentityProviderConfigSchema, {
      config: {
        case: "oauth2Config",
        value: oauth2Config,
      },
    });
  } else if (localIdentityProvider.value.type === IdentityProviderType.OIDC) {
    const oidcConfig = {
      ...configForOIDC.value,
      scopes: scopesStringOfConfig.value.split(" ").filter(Boolean),
      fieldMapping: createProto(FieldMappingSchema, fieldMapping),
    };

    result.config = createProto(IdentityProviderConfigSchema, {
      config: {
        case: "oidcConfig",
        value: oidcConfig,
      },
    });
  } else if (localIdentityProvider.value.type === IdentityProviderType.LDAP) {
    const ldapConfig = {
      ...configForLDAP.value,
      fieldMapping: createProto(FieldMappingSchema, fieldMapping),
    };

    result.config = createProto(IdentityProviderConfigSchema, {
      config: {
        case: "ldapConfig",
        value: ldapConfig,
      },
    });
  }

  return result;
};

const initializeFromProps = () => {
  localIdentityProvider.value = cloneDeep(props.identityProvider);

  // Reset secret modification flags
  isClientSecretModified.value = false;
  isBindPasswordModified.value = false;

  if (props.identityProvider.config?.config?.case === "oauth2Config") {
    const oauth2Config = props.identityProvider.config.config.value;
    configForOAuth2.value = createProto(
      OAuth2IdentityProviderConfigSchema,
      oauth2Config
    );
    Object.assign(fieldMapping, oauth2Config.fieldMapping || {});
    scopesStringOfConfig.value = (oauth2Config.scopes || []).join(" ");
  } else if (props.identityProvider.config?.config?.case === "oidcConfig") {
    const oidcConfig = props.identityProvider.config.config.value;
    configForOIDC.value = createProto(
      OIDCIdentityProviderConfigSchema,
      oidcConfig
    );
    Object.assign(fieldMapping, oidcConfig.fieldMapping || {});
    scopesStringOfConfig.value = (oidcConfig.scopes || []).join(" ");
  } else if (props.identityProvider.config?.config?.case === "ldapConfig") {
    const ldapConfig = props.identityProvider.config.config.value;
    configForLDAP.value = createProto(
      LDAPIdentityProviderConfigSchema,
      ldapConfig
    );
    Object.assign(fieldMapping, ldapConfig.fieldMapping || {});
  }
};

const handleUpdate = async () => {
  if (!canUpdate.value) return;

  isUpdating.value = true;
  try {
    const updatedProvider = await identityProviderStore.patchIdentityProvider(
      buildUpdatedIdentityProvider()
    );

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });

    emit("updated", updatedProvider);
    initializeFromProps();
  } catch {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("identity-provider.identity-provider-update-failed"),
    });
  } finally {
    isUpdating.value = false;
  }
};

const handleDiscard = () => {
  initializeFromProps();
};

const handleDelete = async () => {
  if (!hasWorkspacePermissionV2("bb.identityProviders.delete")) {
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: t("identity-provider.identity-provider-permission-denied"),
    });
    return;
  }

  try {
    await identityProviderStore.deleteIdentityProvider(
      props.identityProvider.name
    );

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("identity-provider.identity-provider-deleted"),
    });

    emit("deleted");
    router.replace({
      name: WORKSPACE_ROUTE_IDENTITY_PROVIDERS,
    });
  } catch {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("identity-provider.identity-provider-delete-failed"),
    });
  }
};

// Initialize from props
watch(() => props.identityProvider, initializeFromProps, { immediate: true });

// Watch for secret field changes
watch(
  () => configForOAuth2.value.clientSecret,
  (newVal, oldVal) => {
    if (oldVal !== undefined && newVal !== oldVal) {
      isClientSecretModified.value = true;
    }
  }
);

watch(
  () => configForOIDC.value.clientSecret,
  (newVal, oldVal) => {
    if (oldVal !== undefined && newVal !== oldVal) {
      isClientSecretModified.value = true;
    }
  }
);

watch(
  () => configForLDAP.value.bindPassword,
  (newVal, oldVal) => {
    if (oldVal !== undefined && newVal !== oldVal) {
      isBindPasswordModified.value = true;
    }
  }
);
</script>
