<template>
  <Drawer :show="props.show" @update:show="$emit('update:show', $event)">
    <DrawerContent
      :title="$t('settings.sso.create')"
      class="w-5xl max-w-[100vw]"
      :closable="true"
    >
      <div class="w-full mx-auto flex flex-col gap-y-6">
        <!-- Steps Navigation -->
        <NSteps :current="currentStep" size="medium">
          <NStep :title="$t('settings.sso.form.type')" />
          <NStep
            v-if="selectedType === IdentityProviderType.OAUTH2"
            :title="$t('settings.sso.form.use-template')"
          />
          <NStep :title="$t('common.general')" />
          <NStep :title="$t('settings.sso.form.configuration')" />
          <NStep :title="$t('settings.sso.form.user-information-mapping')" />
        </NSteps>
        <!-- Step Content -->
        <div
          class="bg-white rounded-lg border border-gray-200 px-2 sm:px-6 pt-6 pb-10"
        >
          <!-- Step 1: Select Provider Type -->
          <div v-if="currentStep === 1" class="flex flex-col gap-y-6">
            <div class="text-center flex flex-col gap-y-2">
              <h2 class="text-2xl font-bold text-gray-900">
                {{ $t("settings.sso.form.type") }}
              </h2>
              <p class="text-gray-600">
                {{ $t("settings.sso.form.type-description") }}
              </p>
            </div>

            <div class="max-w-2xl mx-auto">
              <NRadioGroup
                v-model:value="selectedType"
                size="large"
                class="flex flex-col w-full"
              >
                <div
                  v-for="item in identityProviderTypeList"
                  :key="item.type"
                  class="border border-gray-200 rounded-lg mb-4 p-4 hover:border-gray-300 transition-colors"
                  :class="{
                    'border-blue-500 bg-blue-50': selectedType === item.type,
                  }"
                >
                  <NRadio
                    :value="item.type"
                    :disabled="!subscriptionStore.hasFeature(item.feature)"
                    class="w-full"
                  >
                    <div class="flex items-start gap-x-3 w-full">
                      <component
                        :is="getProviderIcon(item.type)"
                        class="w-6 h-6 mt-1 shrink-0"
                        :stroke-width="1.5"
                      />
                      <div class="flex-1">
                        <div class="flex items-center gap-x-2">
                          <span class="text-lg font-medium text-gray-900">
                            {{ identityProviderTypeToString(item.type) }}
                          </span>
                          <FeatureBadge :feature="item.feature" />
                        </div>
                        <p class="text-sm text-gray-600 mt-1">
                          {{ getProviderDescription(item.type) }}
                        </p>
                      </div>
                    </div>
                  </NRadio>
                </div>
              </NRadioGroup>

              <!-- External URL Warning -->
              <MissingExternalURLAttention
                v-if="
                  selectedType === IdentityProviderType.OAUTH2 ||
                    selectedType === IdentityProviderType.OIDC
                  "
                class="mt-6"
              />
            </div>
          </div>

          <!-- Step 2: Select OAuth2 Template (only for OAuth2) -->
          <div
            v-else-if="
              currentStep === 2 && selectedType === IdentityProviderType.OAUTH2
            "
            class="flex flex-col gap-y-6"
          >
            <div class="text-center flex flex-col gap-y-2">
              <h2 class="text-2xl font-bold text-gray-900">
                {{ $t("settings.sso.form.use-template") }}
              </h2>
              <p class="text-gray-600">
                {{ $t("settings.sso.form.template-description") }}
              </p>
            </div>

            <div class="max-w-3xl mx-auto">
              <NRadioGroup
                :value="selectedTemplate?.title"
                @update:value="
                  (value) => {
                    const template = templateList.find(
                      (t) => t.title === value
                    );
                    if (template) handleTemplateSelect(template);
                  }
                "
                size="large"
                class="grid! grid-cols-1 sm:grid-cols-2 gap-4"
              >
                <div
                  v-for="template in templateList"
                  :key="template.title"
                  class="border border-gray-200 rounded-lg p-4 hover:border-gray-300 transition-colors"
                  :class="{
                    'border-blue-500 bg-blue-50':
                      selectedTemplate?.title === template.title,
                  }"
                >
                  <NRadio
                    :value="template.title"
                    :disabled="!subscriptionStore.hasFeature(template.feature)"
                    class="w-full"
                  >
                    <div class="flex items-center gap-x-3">
                      <component
                        :is="getTemplateIcon(template.title)"
                        class="w-8 h-8 shrink-0"
                        :stroke-width="1"
                      />
                      <div class="flex-1">
                        <div class="flex items-center gap-x-2">
                          <span class="text-base font-medium text-gray-900">
                            {{ template.title }}
                          </span>
                          <FeatureBadge :feature="template.feature" />
                        </div>
                        <p class="text-sm text-gray-600">
                          {{ getTemplateDescription(template.title) }}
                        </p>
                      </div>
                    </div>
                  </NRadio>
                </div>
              </NRadioGroup>
            </div>
          </div>

          <!-- Step 3: Basic Information -->
          <div
            v-else-if="
              currentStep ===
              (selectedType === IdentityProviderType.OAUTH2 ? 3 : 2)
            "
            class="flex flex-col gap-y-6"
          >
            <div class="text-center flex flex-col gap-y-2">
              <h2 class="text-2xl font-bold text-gray-900">
                {{ $t("common.general") }}
              </h2>
              <p class="text-gray-600">
                {{ $t("settings.sso.form.general-setting-description") }}
              </p>
            </div>

            <div class="max-w-2xl mx-auto flex flex-col gap-y-6">
              <div>
                <label class="block text-base font-semibold text-gray-800 mb-2">
                  {{ $t("settings.sso.form.name") }}
                  <RequiredStar />
                </label>
                <BBTextField
                  v-model:value="identityProvider.title"
                  required
                  size="large"
                  class="w-full text-base mb-2"
                  :placeholder="$t('settings.sso.form.name-description')"
                />
                <ResourceIdField
                  ref="resourceIdField"
                  resource-type="idp"
                  v-model:value="resourceIdValue"
                  :suffix="true"
                  :resource-title="identityProvider.title"
                  :fetch-resource="
                    (id) =>
                      identityProviderStore.getOrFetchIdentityProviderByName(
                        `${idpNamePrefix}${id}`,
                        true /* silent */
                      )
                  "
                />
              </div>

              <div>
                <label class="block text-base font-semibold text-gray-800 mb-2">
                  {{ $t("settings.sso.form.domain") }}
                </label>
                <BBTextField
                  v-model:value="identityProvider.domain"
                  :disabled="selectedTemplate?.domainDisabled"
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

          <!-- Step 4: Provider Configuration -->
          <div
            v-else-if="
              currentStep ===
              (selectedType === IdentityProviderType.OAUTH2 ? 4 : 3)
            "
            class="flex flex-col gap-y-6"
          >
            <div class="text-center flex flex-col gap-y-2">
              <h2 class="text-2xl font-bold text-gray-900">
                {{ $t("settings.sso.form.configuration") }}
              </h2>
              <p class="text-gray-600">
                {{ $t("settings.sso.form.configuration-description") }}
              </p>
            </div>

            <div class="max-w-3xl mx-auto">
              <!-- External URL Info -->
              <IdentityProviderExternalURL
                v-if="
                  selectedType === IdentityProviderType.OAUTH2 ||
                  selectedType === IdentityProviderType.OIDC
                "
                :type="selectedType"
                class="mb-6"
              />

              <!-- Configuration Form based on type -->
              <IdentityProviderForm
                :identity-provider="identityProvider"
                :provider-type="selectedType"
                :config-for-oauth2="configForOAuth2"
                :config-for-oidc="configForOIDC"
                :config-for-ldap="configForLDAP"
                :scopes-string="scopesStringOfConfig"
                @update:config-for-oauth2="configForOAuth2 = $event"
                @update:config-for-oidc="configForOIDC = $event"
                @update:config-for-ldap="configForLDAP = $event"
                @update:scopes-string="scopesStringOfConfig = $event"
              />
            </div>
          </div>

          <!-- Step 5: User Information Mapping -->
          <div
            v-else-if="
              currentStep ===
              (selectedType === IdentityProviderType.OAUTH2 ? 5 : 4)
            "
            class="flex flex-col gap-y-6"
          >
            <div class="text-center flex flex-col gap-y-2">
              <h2 class="text-2xl font-bold text-gray-900">
                {{ $t("settings.sso.form.user-information-mapping") }}
              </h2>
              <p class="text-gray-600">
                {{
                  $t("settings.sso.form.user-information-mapping-description")
                }}
                <LearnMoreLink
                  class="ml-1"
                  url="https://docs.bytebase.com/administration/sso/oauth2#user-information-field-mapping?source=console"
                />
              </p>
            </div>

            <div class="max-w-2xl mx-auto flex flex-col gap-y-6">
              <div class="grid grid-cols-[256px_1fr] gap-4 items-center">
                <BBTextField
                  v-model:value="fieldMapping.identifier"
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
                  size="large"
                  class="w-full text-base"
                  :placeholder="
                    $t('settings.sso.form.display-name-placeholder')
                  "
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
                v-if="selectedType === IdentityProviderType.OIDC"
                class="grid grid-cols-[256px_1fr] gap-4 items-center"
              >
                <BBTextField
                  v-model:value="fieldMapping.groups"
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

              <div v-if="selectedType === IdentityProviderType.OIDC">
                <p class="text-sm text-gray-600">
                  {{ $t("settings.sso.form.groups-description") }}
                </p>
              </div>

              <div>
                <TestConnection
                  :disabled="!allowTestConnection"
                  :size="'large'"
                  :is-creating="true"
                  :idp="idpToCreate"
                />
              </div>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="flex items-center justify-end gap-x-3">
          <NButton
            v-if="currentStep === 1"
            @click="$emit('update:show', false)"
          >
            {{ $t("common.cancel") }}
          </NButton>
          <NButton v-if="currentStep > 1" @click="handlePrevStep">
            {{ $t("common.back") }}
          </NButton>
          <NButton
            v-if="!isLastStep"
            type="primary"
            :disabled="!canProceedToNextStep"
            @click="handleNextStep"
          >
            {{ $t("common.next") }}
          </NButton>
          <NButton
            v-else
            type="primary"
            :disabled="!canCreate"
            :loading="isCreating"
            @click="handleCreate"
          >
            {{ $t("common.create") }}
          </NButton>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { create as createProto } from "@bufbuild/protobuf";
import { head } from "lodash-es";
import {
  ArrowRightIcon,
  BuildingIcon,
  ChromeIcon,
  DatabaseIcon,
  GithubIcon,
  GitlabIcon,
  InfoIcon,
  KeyIcon,
  ShieldCheckIcon,
} from "lucide-vue-next";
import {
  NButton,
  NRadio,
  NRadioGroup,
  NStep,
  NSteps,
  NTooltip,
} from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBTextField } from "@/bbkit";
import { FeatureBadge } from "@/components/FeatureGuard";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { MissingExternalURLAttention } from "@/components/v2/Form";
import ResourceIdField from "@/components/v2/Form/ResourceIdField.vue";
import { hasFeature, pushNotification, useSubscriptionV1Store } from "@/store";
import { useIdentityProviderStore } from "@/store/modules/idp";
import { idpNamePrefix } from "@/store/modules/v1/common";
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
  OAuth2AuthStyle,
  OAuth2IdentityProviderConfigSchema,
  OIDCIdentityProviderConfigSchema,
} from "@/types/proto-es/v1/idp_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { OAuth2IdentityProviderTemplate } from "@/utils";
import { identityProviderTypeToString } from "@/utils";
import IdentityProviderExternalURL from "./IdentityProviderExternalURL.vue";
import IdentityProviderForm from "./IdentityProviderForm.vue";
import TestConnection from "./TestConnection.vue";

interface IdentityProviderTemplate extends OAuth2IdentityProviderTemplate {
  feature: PlanFeature;
  domainDisabled?: boolean;
}

const props = defineProps<{
  show: boolean;
}>();

const emit = defineEmits<{
  (e: "created", provider: IdentityProvider): void;
  (e: "update:show", show: boolean): void;
}>();

const { t } = useI18n();
const identityProviderStore = useIdentityProviderStore();
const subscriptionStore = useSubscriptionV1Store();

// State
const currentStep = ref(1);
const selectedType = ref<IdentityProviderType>(IdentityProviderType.OAUTH2);
const selectedTemplate = ref<IdentityProviderTemplate>();
const isCreating = ref(false);

const identityProvider = ref<IdentityProvider>(
  createProto(IdentityProviderSchema, {})
);
const configForOAuth2 = ref<OAuth2IdentityProviderConfig>(
  createProto(OAuth2IdentityProviderConfigSchema, {
    authStyle: OAuth2AuthStyle.IN_PARAMS,
  })
);
const configForOIDC = ref<OIDCIdentityProviderConfig>(
  createProto(OIDCIdentityProviderConfigSchema, {
    authStyle: OAuth2AuthStyle.IN_PARAMS,
  })
);
const configForLDAP = ref<LDAPIdentityProviderConfig>(
  createProto(LDAPIdentityProviderConfigSchema, {
    port: 389,
  })
);
const scopesStringOfConfig = ref<string>("");
const fieldMapping = reactive<FieldMapping>(
  createProto(FieldMappingSchema, {})
);

const resourceIdField = ref<InstanceType<typeof ResourceIdField>>();
const resourceIdValue = ref<string>("");

// Computed
const idpToCreate = computed((): IdentityProvider => {
  const base = createProto(IdentityProviderSchema, {
    name: resourceIdField.value?.resourceId || resourceIdValue.value,
    title: identityProvider.value.title,
    domain: identityProvider.value.domain,
    type: selectedType.value,
  });

  if (selectedType.value === IdentityProviderType.OAUTH2) {
    base.config = createProto(IdentityProviderConfigSchema, {
      config: {
        case: "oauth2Config",
        value: createProto(OAuth2IdentityProviderConfigSchema, {
          ...configForOAuth2.value,
          scopes: scopesStringOfConfig.value.split(" ").filter(Boolean),
          fieldMapping: createProto(FieldMappingSchema, fieldMapping),
        }),
      },
    });
  } else if (selectedType.value === IdentityProviderType.OIDC) {
    base.config = createProto(IdentityProviderConfigSchema, {
      config: {
        case: "oidcConfig",
        value: createProto(OIDCIdentityProviderConfigSchema, {
          ...configForOIDC.value,
          scopes: scopesStringOfConfig.value.split(" ").filter(Boolean),
          fieldMapping: createProto(FieldMappingSchema, fieldMapping),
        }),
      },
    });
  } else if (selectedType.value === IdentityProviderType.LDAP) {
    base.config = createProto(IdentityProviderConfigSchema, {
      config: {
        case: "ldapConfig",
        value: createProto(LDAPIdentityProviderConfigSchema, {
          ...configForLDAP.value,
          fieldMapping: createProto(FieldMappingSchema, fieldMapping),
        }),
      },
    });
  }

  return base;
});

const resourceId = computed(() => {
  return resourceIdField.value?.resourceId || resourceIdValue.value || "";
});

const identityProviderTypeList = computed(() => {
  return [
    {
      type: IdentityProviderType.OAUTH2,
      feature: PlanFeature.FEATURE_GOOGLE_AND_GITHUB_SSO,
    },
    {
      type: IdentityProviderType.OIDC,
      feature: PlanFeature.FEATURE_ENTERPRISE_SSO,
    },
    {
      type: IdentityProviderType.LDAP,
      feature: PlanFeature.FEATURE_ENTERPRISE_SSO,
    },
  ];
});

const templateList = computed((): IdentityProviderTemplate[] => {
  return [
    {
      title: "Google",
      name: "",
      domain: "google.com",
      domainDisabled: !hasFeature(PlanFeature.FEATURE_ENTERPRISE_SSO),
      type: IdentityProviderType.OAUTH2,
      feature: PlanFeature.FEATURE_GOOGLE_AND_GITHUB_SSO,
      config: createProto(OAuth2IdentityProviderConfigSchema, {
        clientId: "",
        clientSecret: "",
        authUrl: "https://accounts.google.com/o/oauth2/v2/auth",
        tokenUrl: "https://oauth2.googleapis.com/token",
        userInfoUrl: "https://www.googleapis.com/oauth2/v2/userinfo",
        scopes: [
          "https://www.googleapis.com/auth/userinfo.email",
          "https://www.googleapis.com/auth/userinfo.profile",
        ],
        skipTlsVerify: false,
        authStyle: OAuth2AuthStyle.IN_PARAMS,
        fieldMapping: createProto(FieldMappingSchema, {
          identifier: "email",
          displayName: "name",
          phone: "",
          groups: "",
        }),
      }),
    },
    {
      title: "GitHub",
      name: "",
      domain: "github.com",
      domainDisabled: !hasFeature(PlanFeature.FEATURE_ENTERPRISE_SSO),
      type: IdentityProviderType.OAUTH2,
      feature: PlanFeature.FEATURE_GOOGLE_AND_GITHUB_SSO,
      config: createProto(OAuth2IdentityProviderConfigSchema, {
        clientId: "",
        clientSecret: "",
        authUrl: "https://github.com/login/oauth/authorize",
        tokenUrl: "https://github.com/login/oauth/access_token",
        userInfoUrl: "https://api.github.com/user",
        scopes: ["user"],
        skipTlsVerify: false,
        authStyle: OAuth2AuthStyle.IN_PARAMS,
        fieldMapping: createProto(FieldMappingSchema, {
          identifier: "email",
          displayName: "name",
          phone: "",
          groups: "",
        }),
      }),
    },
    {
      title: "GitLab",
      name: "",
      domain: "gitlab.com",
      type: IdentityProviderType.OAUTH2,
      feature: PlanFeature.FEATURE_ENTERPRISE_SSO,
      config: createProto(OAuth2IdentityProviderConfigSchema, {
        clientId: "",
        clientSecret: "",
        authUrl: "https://gitlab.com/oauth/authorize",
        tokenUrl: "https://gitlab.com/oauth/token",
        userInfoUrl: "https://gitlab.com/api/v4/user",
        scopes: ["read_user"],
        skipTlsVerify: false,
        authStyle: OAuth2AuthStyle.IN_PARAMS,
        fieldMapping: createProto(FieldMappingSchema, {
          identifier: "email",
          displayName: "name",
          phone: "",
          groups: "",
        }),
      }),
    },
    {
      title: "Microsoft Entra",
      name: "",
      domain: "",
      type: IdentityProviderType.OAUTH2,
      feature: PlanFeature.FEATURE_ENTERPRISE_SSO,
      config: createProto(OAuth2IdentityProviderConfigSchema, {
        clientId: "",
        clientSecret: "",
        authUrl:
          "https://login.microsoftonline.com/{uuid}/oauth2/v2.0/authorize",
        tokenUrl: "https://login.microsoftonline.com/{uuid}/oauth2/v2.0/token",
        userInfoUrl: "https://graph.microsoft.com/v1.0/me",
        scopes: ["user.read"],
        skipTlsVerify: false,
        authStyle: OAuth2AuthStyle.IN_PARAMS,
        fieldMapping: createProto(FieldMappingSchema, {
          identifier: "userPrincipalName",
          displayName: "displayName",
          phone: "",
          groups: "",
        }),
      }),
    },
    {
      title: "Custom",
      name: "",
      domain: "",
      type: IdentityProviderType.OAUTH2,
      feature: PlanFeature.FEATURE_ENTERPRISE_SSO,
      config: createProto(OAuth2IdentityProviderConfigSchema, {
        clientId: "",
        clientSecret: "",
        authUrl: "",
        tokenUrl: "",
        userInfoUrl: "",
        scopes: [],
        skipTlsVerify: false,
        authStyle: OAuth2AuthStyle.IN_PARAMS,
        fieldMapping: createProto(FieldMappingSchema, {
          identifier: "",
          displayName: "",
          phone: "",
          groups: "",
        }),
      }),
    },
  ];
});

const maxSteps = computed(() => {
  return selectedType.value === IdentityProviderType.OAUTH2 ? 5 : 4;
});

const isLastStep = computed(() => {
  return currentStep.value === maxSteps.value;
});

const canProceedToNextStep = computed(() => {
  switch (currentStep.value) {
    case 1:
      return !!selectedType.value;
    case 2:
      if (selectedType.value === IdentityProviderType.OAUTH2) {
        return !!selectedTemplate.value;
      }
      // For non-OAuth2, step 2 is basic info
      return !!(identityProvider.value.title && resourceId.value);
    case 3:
      if (selectedType.value === IdentityProviderType.OAUTH2) {
        // Basic info step for OAuth2
        return !!(identityProvider.value.title && resourceId.value);
      }
      // Configuration step for non-OAuth2
      return isConfigurationValid.value;
    case 4:
      if (selectedType.value === IdentityProviderType.OAUTH2) {
        // Configuration step for OAuth2
        return isConfigurationValid.value;
      }
      // Mapping step for non-OAuth2
      return !!fieldMapping.identifier;
    case 5:
      // Mapping step for OAuth2
      return !!fieldMapping.identifier;
    default:
      return false;
  }
});

const isConfigurationValid = computed(() => {
  if (selectedType.value === IdentityProviderType.OAUTH2) {
    return !!(
      configForOAuth2.value.clientId &&
      configForOAuth2.value.clientSecret &&
      configForOAuth2.value.authUrl &&
      configForOAuth2.value.tokenUrl &&
      configForOAuth2.value.userInfoUrl &&
      scopesStringOfConfig.value
    );
  } else if (selectedType.value === IdentityProviderType.OIDC) {
    return !!(
      configForOIDC.value.clientId &&
      configForOIDC.value.clientSecret &&
      configForOIDC.value.issuer &&
      scopesStringOfConfig.value
    );
  } else if (selectedType.value === IdentityProviderType.LDAP) {
    return !!(
      configForLDAP.value.host &&
      configForLDAP.value.port &&
      configForLDAP.value.bindDn &&
      configForLDAP.value.bindPassword &&
      configForLDAP.value.baseDn &&
      configForLDAP.value.userFilter
    );
  }
  return false;
});

const canCreate = computed(() => {
  return !!(
    identityProvider.value.title &&
    resourceId.value &&
    isConfigurationValid.value &&
    fieldMapping.identifier
  );
});

const allowTestConnection = computed(() => {
  return isConfigurationValid.value && fieldMapping.identifier;
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

const getProviderDescription = (type: IdentityProviderType) => {
  switch (type) {
    case IdentityProviderType.OAUTH2:
      return t("settings.sso.form.oauth2-description");
    case IdentityProviderType.OIDC:
      return t("settings.sso.form.oidc-description");
    case IdentityProviderType.LDAP:
      return t("settings.sso.form.ldap-description");
    default:
      return "";
  }
};

const getTemplateIcon = (title: string) => {
  switch (title.toLowerCase()) {
    case "google":
      return ChromeIcon;
    case "github":
      return GithubIcon;
    case "gitlab":
      return GitlabIcon;
    case "microsoft entra":
      return BuildingIcon;
    default:
      return KeyIcon;
  }
};

const getTemplateDescription = (title: string) => {
  switch (title.toLowerCase()) {
    case "google":
      return t("settings.sso.form.google-template-description");
    case "github":
      return t("settings.sso.form.github-template-description");
    case "gitlab":
      return t("settings.sso.form.gitlab-template-description");
    case "microsoft entra":
      return t("settings.sso.form.microsoft-entra-template-description");
    case "custom":
      return t("settings.sso.form.custom-template-description");
    default:
      return "";
  }
};

const handleTemplateSelect = (template: IdentityProviderTemplate) => {
  selectedTemplate.value = template;
  // Update title and domain from template
  identityProvider.value.title = template.title;
  identityProvider.value.domain = template.domain;

  if (template.type === IdentityProviderType.OAUTH2) {
    // Create a completely new object to ensure reactivity
    const newConfig = createProto(OAuth2IdentityProviderConfigSchema, {
      clientId: template.config.clientId || "",
      clientSecret: template.config.clientSecret || "",
      authUrl: template.config.authUrl || "",
      tokenUrl: template.config.tokenUrl || "",
      userInfoUrl: template.config.userInfoUrl || "",
      scopes: template.config.scopes || [],
      skipTlsVerify: template.config.skipTlsVerify || false,
      authStyle: template.config.authStyle || OAuth2AuthStyle.IN_PARAMS,
      fieldMapping:
        template.config.fieldMapping || createProto(FieldMappingSchema, {}),
    });
    configForOAuth2.value = newConfig;

    // Clear existing field mapping and assign new values
    Object.keys(fieldMapping).forEach((key) => {
      delete fieldMapping[key as keyof FieldMapping];
    });
    Object.assign(fieldMapping, template.config.fieldMapping || {});

    scopesStringOfConfig.value = template.config.scopes.join(" ");
  }
};

const handlePrevStep = () => {
  if (currentStep.value > 1) {
    currentStep.value--;
  }
};

const handleNextStep = () => {
  if (currentStep.value < maxSteps.value && canProceedToNextStep.value) {
    // Ensure resource ID is captured before moving to next step
    if (resourceIdField.value?.resourceId) {
      resourceIdValue.value = resourceIdField.value.resourceId;
    }
    currentStep.value++;
  }
};

const handleCreate = async () => {
  if (!canCreate.value) return;

  isCreating.value = true;
  try {
    const finalResourceId =
      resourceIdField.value?.resourceId || resourceIdValue.value;
    if (!finalResourceId) {
      throw new Error("Resource ID is required");
    }

    const createdProvider = await identityProviderStore.createIdentityProvider(
      idpToCreate.value
    );

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("identity-provider.identity-provider-created"),
    });

    emit("created", createdProvider);
  } catch (error) {
    console.error("Failed to create identity provider:", error);
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("identity-provider.identity-provider-create-failed"),
    });
  } finally {
    isCreating.value = false;
  }
};

// Watchers
watch(
  selectedType,
  (newType, oldType) => {
    // Only process if type actually changed, not on initial mount
    if (oldType === undefined) {
      // Initial mount - set up OAuth2 defaults if needed
      if (newType === IdentityProviderType.OAUTH2) {
        if (!selectedTemplate.value) {
          selectedTemplate.value = head(templateList.value);
        }
        if (selectedTemplate.value) {
          handleTemplateSelect(selectedTemplate.value);
        }
      }
      return;
    }

    if (newType !== oldType) {
      // Type changed - reset form but preserve resource ID
      if (newType === IdentityProviderType.OAUTH2) {
        // Switching to OAuth2
        if (!selectedTemplate.value) {
          selectedTemplate.value = head(templateList.value);
        }
        if (selectedTemplate.value) {
          handleTemplateSelect(selectedTemplate.value);
        }
      } else {
        // Switching to OIDC or LDAP
        identityProvider.value.title = "";
        identityProvider.value.domain = "";

        // Clear field mapping
        Object.keys(fieldMapping).forEach((key) => {
          delete fieldMapping[key as keyof FieldMapping];
        });
        scopesStringOfConfig.value = "";
      }
    }
  },
  {
    immediate: true,
  }
);

// Watch for template changes to ensure form is updated
watch(
  selectedTemplate,
  (newTemplate) => {
    if (newTemplate && selectedType.value === IdentityProviderType.OAUTH2) {
      handleTemplateSelect(newTemplate);
    }
  },
  { deep: true }
);
</script>
