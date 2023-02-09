<template>
  <div
    class="w-full flex flex-col justify-start items-start overflow-x-hidden px-1"
    :class="[isCreating && '!w-128']"
  >
    <div
      v-if="isCreating"
      class="w-full flex flex-col justify-start items-start"
    >
      <div class="w-full flex flex-row justify-between items-center">
        <p class="textlabel my-2">{{ $t("settings.sso.form.type") }}</p>
        <a
          v-if="userDocLink"
          :href="userDocLink"
          class="normal-link text-sm inline-flex flex-row items-center"
          target="_blank"
        >
          {{ $t("settings.sso.form.learn-more-with-user-doc") }}
          <heroicons-outline:external-link class="w-4 h-4" />
        </a>
      </div>
      <div class="w-full flex flex-row justify-start items-start space-x-2">
        <label
          v-for="item in identityProviderTypeList"
          :key="item"
          class="flex flex-row justify-center items-center cursor-pointer mr-2"
          :for="`radio-${item}`"
        >
          <input
            :id="`radio-${item}`"
            v-model="state.type"
            type="radio"
            class="btn mr-2"
            :value="item"
            :checked="state.type === item"
          />
          <span>{{ identityProviderTypeToString(item) }}</span>
        </label>
      </div>
    </div>

    <!-- OAuth2 form group -->
    <div
      v-if="state.type === IdentityProviderType.OAUTH2"
      class="w-full flex flex-col justify-start items-start space-y-3"
    >
      <template v-if="isCreating">
        <p class="textlabel mt-4">
          {{ $t("settings.sso.form.use-template") }}
        </p>
        <div class="w-full flex flex-row justify-start items-start space-x-2">
          <label
            v-for="template in templateList"
            :key="template.title"
            class="w-24 h-24 border rounded-md flex flex-col justify-center items-center cursor-pointer hover:bg-gray-100"
            :for="`radio-${template.title}`"
            @click="handleTemplateSelect(template)"
          >
            <span>{{ template.title }}</span>
            <input
              :id="`radio-${template.title}`"
              type="radio"
              class="btn mt-4"
              :checked="selectedTemplate?.title === template.title"
            />
          </label>
        </div>
      </template>
      <hr class="w-full bg-gray-50" />
      <p class="text-lg font-medium !mt-4">
        {{ $t("settings.sso.form.basic-information") }}
      </p>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          {{ $t("settings.sso.form.name") }}
          <span class="text-red-600">*</span>
        </p>
        <input
          v-model="identityProvider.title"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>
      <div
        v-if="isCreating"
        class="w-full flex flex-col justify-start items-start"
      >
        <p class="textlabel">
          {{ $t("settings.sso.form.resource-id") }}
          <span class="text-red-600">*</span>
        </p>
        <input
          v-model="identityProvider.name"
          type="text"
          class="textfield mt-1 w-full"
          :placeholder="$t('settings.sso.form.resource-id-description')"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          {{ $t("settings.sso.form.domain") }}
          <span class="text-red-600">*</span>
        </p>
        <input
          v-model="identityProvider.domain"
          type="text"
          class="textfield mt-1 w-full"
          :placeholder="$t('settings.sso.form.domain-description')"
        />
      </div>

      <div class="w-full flex flex-col justify-start items-start">
        <p class="text-lg font-medium mt-2">
          {{ $t("settings.sso.form.identity-provider-information") }}
        </p>
        <p class="textinfolabel">
          {{
            $t("settings.sso.form.identity-provider-information-description")
          }}
        </p>
      </div>
      <div
        v-if="isCreating"
        class="w-full flex flex-row justify-start items-center"
      >
        <p class="textlabel">
          {{ $t("settings.sso.form.redirect-url") }}
        </p>
        <ShowMoreIcon
          class="ml-1 mr-2"
          :content="$t('settings.sso.form.redirect-url-description')"
        />
        <div class="relative grow">
          <input
            type="text"
            class="textfield w-full pr-10"
            readonly
            disabled
            :value="redirectUrl"
          />
          <button
            tabindex="-1"
            class="absolute right-0 top-1/2 -translate-y-1/2 mr-2 p-1 text-control-light rounded hover:bg-gray-100"
            @click.prevent="copyRedirectUrl"
          >
            <heroicons-outline:clipboard class="w-5 h-5" />
          </button>
        </div>
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          Client ID
          <span class="text-red-600">*</span>
        </p>
        <input
          v-model="configForOAuth2.clientId"
          type="text"
          class="textfield mt-1 w-full"
          placeholder="ex. 6655asd77895265aa110ac0d3"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          Client secret
          <span class="text-red-600">*</span>
        </p>
        <input
          v-model="configForOAuth2.clientSecret"
          type="text"
          class="textfield mt-1 w-full"
          :placeholder="
            isCreating
              ? 'ex. 5bbezxc3972ca304de70c5d70a6aa932asd8'
              : $t('common.sensitive-placeholder')
          "
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          Auth URL
          <span class="text-red-600">*</span>
          <span class="textinfolabel">
            ({{ $t("settings.sso.form.auth-url-description") }})
          </span>
        </p>
        <input
          v-model="configForOAuth2.authUrl"
          type="text"
          class="textfield mt-1 w-full"
          placeholder="ex. https://github.com/login/oauth/authorize"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          Scopes
          <span class="text-red-600">*</span>
          <span class="textinfolabel">
            ({{ $t("settings.sso.form.scopes-description") }})
          </span>
        </p>
        <input
          v-model="scopesStringOfConfig"
          type="text"
          class="textfield mt-1 w-full"
          placeholder="ex. user"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          Token URL
          <span class="text-red-600">*</span>
          <span class="textinfolabel">
            ({{ $t("settings.sso.form.token-url-description") }})
          </span>
        </p>
        <input
          v-model="configForOAuth2.tokenUrl"
          type="text"
          class="textfield mt-1 w-full"
          placeholder="ex. https://github.com/login/oauth/access_token"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          User information URL
          <span class="text-red-600">*</span>
          <span class="textinfolabel">
            ({{ $t("settings.sso.form.user-info-url-description") }})
          </span>
        </p>
        <input
          v-model="configForOAuth2.userInfoUrl"
          type="text"
          class="textfield mt-1 w-full"
          placeholder="ex. https://api.github.com/user"
        />
      </div>

      <div class="w-full flex flex-col justify-start items-start">
        <p class="text-lg font-medium mt-2">
          {{ $t("settings.sso.form.user-information-mapping") }}
        </p>
        <p class="textinfolabel">
          {{ $t("settings.sso.form.user-information-mapping-description") }}
          <a
            href="https://www.bytebase.com/docs/administration/sso/oauth2#user-information-field-mapping?source=console"
            class="normal-link text-sm inline-flex flex-row items-center"
            target="_blank"
          >
            {{ $t("common.learn-more") }}
            <heroicons-outline:external-link class="w-4 h-4" />
          </a>
        </p>
      </div>
      <div class="w-full grid grid-cols-[256px_1fr]">
        <input
          v-model="configForOAuth2.fieldMapping!.identifier"
          type="text"
          class="textfield mt-1 w-full"
          placeholder="ex. login"
        />
        <div class="w-full flex flex-row justify-start items-center text-sm">
          <heroicons-outline:arrow-right
            class="mx-1 h-auto w-4 text-gray-300"
          />
          <p>
            {{ $t("settings.sso.form.identifier") }}
            <span class="text-red-600">*</span>
          </p>
        </div>
      </div>
      <div class="w-full grid grid-cols-[256px_1fr]">
        <input
          v-model="configForOAuth2.fieldMapping!.displayName"
          type="text"
          class="textfield mt-1 w-full"
          placeholder="ex. name"
        />
        <div class="w-full flex flex-row justify-start items-center text-sm">
          <heroicons-outline:arrow-right
            class="mx-1 h-auto w-4 text-gray-300"
          />
          <p>
            {{ $t("settings.sso.form.display-name") }}
          </p>
        </div>
      </div>
      <div class="w-full grid grid-cols-[256px_1fr]">
        <input
          v-model="configForOAuth2.fieldMapping!.email"
          type="text"
          class="textfield mt-1 w-full"
          placeholder="ex. email"
        />
        <div class="w-full flex flex-row justify-start items-center text-sm">
          <heroicons-outline:arrow-right
            class="mx-1 h-auto w-4 text-gray-300"
          />
          <p>
            {{ $t("settings.sso.form.email") }}
          </p>
        </div>
      </div>
    </div>

    <!-- OIDC form group -->
    <div
      v-else-if="state.type === IdentityProviderType.OIDC"
      class="w-full flex flex-col justify-start items-start space-y-3"
    >
      <p class="textinfolabel !mt-4">
        {{ $t("settings.sso.form.basic-information") }}
      </p>
      <div class="w-full flex flex-col justify-start items-start">
        <label>{{ $t("settings.sso.form.name") }}</label>
        <input
          v-model="identityProvider.title"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>
      <div
        v-if="isCreating"
        class="w-full flex flex-col justify-start items-start"
      >
        <label>{{ $t("settings.sso.form.resource-id") }}</label>
        <input
          v-model="identityProvider.name"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <label>{{ $t("settings.sso.form.domain") }}</label>
        <input
          v-model="identityProvider.domain"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <label>Issuer</label>
        <input
          v-model="configForOIDC.issuer"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <label>Client ID</label>
        <input
          v-model="configForOIDC.clientId"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <label>Client secret</label>
        <input
          v-model="configForOIDC.clientSecret"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>

      <p class="textinfolabel !mt-4">
        {{ $t("settings.sso.form.user-information-mapping") }}
      </p>
      <div class="w-full flex flex-col justify-start items-start">
        <label>{{ $t("settings.sso.form.identifier") }}</label>
        <input
          v-model="configForOIDC.fieldMapping!.identifier"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <label>{{ $t("settings.sso.form.display-name") }}</label>
        <input
          v-model="configForOIDC.fieldMapping!.displayName"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <label>{{ $t("common.email") }}</label>
        <input
          v-model="configForOIDC.fieldMapping!.email"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>
    </div>

    <!-- Button group -->
    <div
      class="mt-4 space-x-4 w-full flex flex-row justify-between items-center"
    >
      <div class="space-x-4 flex flex-row justify-start items-center">
        <button
          :disabled="!allowTestConnection"
          class="btn-normal"
          @click="testConnection"
        >
          {{ $t("identity-provider.test-connection") }}
        </button>
        <BBButtonConfirm
          v-if="!isCreating"
          :style="'DELETE'"
          :button-text="$t('settings.sso.deletion')"
          :ok-text="'Delete'"
          :confirm-title="`Delete SSO '${identityProvider.name}'?`"
          :require-confirm="true"
          @confirm="handleDeleteButtonClick"
        />
      </div>
      <div class="space-x-4 flex flex-row justify-end items-center">
        <template v-if="isCreating">
          <button class="btn-normal" @click="handleCancelButtonClick">
            {{ $t("common.cancel") }}
          </button>
          <button
            class="btn-primary"
            :disabled="!allowCreate"
            @click="handleCreateButtonClick"
          >
            {{ $t("common.create") }}
          </button>
        </template>
        <template v-else>
          <button
            class="btn-normal"
            :disabled="!allowUpdate"
            @click="handleDiscardChangesButtonClick"
          >
            {{ $t("common.discard-changes") }}
          </button>
          <button
            class="btn-primary"
            :disabled="!allowUpdate"
            @click="handleUpdateButtonClick"
          >
            {{ $t("common.update") }}
          </button>
        </template>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { cloneDeep, head, isEqual } from "lodash-es";
import { ClientError } from "nice-grpc-common";
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import { useI18n } from "vue-i18n";
import {
  computed,
  reactive,
  defineEmits,
  defineProps,
  ref,
  onMounted,
  onUnmounted,
  watch,
} from "vue";
import {
  FieldMapping,
  IdentityProvider,
  IdentityProviderConfig,
  IdentityProviderType,
  OAuth2IdentityProviderConfig,
  OIDCIdentityProviderConfig,
} from "@/types/proto/v1/idp_service";
import { useIdentityProviderStore } from "@/store/modules/idp";
import { pushNotification, useActuatorStore } from "@/store";
import {
  IdentityProviderTemplate,
  identityProviderTemplateList,
  identityProviderTypeToString,
  isDev,
  openWindowForSSO,
} from "@/utils";
import { OAuthWindowEventPayload } from "@/types";
import { identityProviderClient } from "@/grpcweb";

interface LocalState {
  type: IdentityProviderType;
}

const props = defineProps<{
  identityProviderName?: string;
}>();

const emit = defineEmits<{
  (e: "delete", identityProvider: IdentityProvider): void;
  (e: "cancel"): void;
  (e: "confirm", identityProvider: IdentityProvider): void;
}>();

const { t } = useI18n();
const identityProviderStore = useIdentityProviderStore();
const state = reactive<LocalState>({
  type: IdentityProviderType.OAUTH2,
});
const identityProvider = ref<IdentityProvider>(
  IdentityProvider.fromPartial({})
);
const configForOAuth2 = ref<OAuth2IdentityProviderConfig>(
  OAuth2IdentityProviderConfig.fromPartial({
    fieldMapping: FieldMapping.fromPartial({}),
  })
);
const scopesStringOfConfig = ref<string>("");
const configForOIDC = ref<OIDCIdentityProviderConfig>(
  OIDCIdentityProviderConfig.fromPartial({
    fieldMapping: FieldMapping.fromPartial({}),
  })
);
const selectedTemplate = ref<IdentityProviderTemplate>();

const identityProviderTypeList = computed(() => {
  const list = [IdentityProviderType.OAUTH2];
  if (isDev()) {
    list.push(IdentityProviderType.OIDC);
  }
  return list;
});

const redirectUrl = computed(() => {
  return `${
    useActuatorStore().serverInfo?.externalUrl || window.origin
  }/oauth/callback`;
});

const isCreating = computed(() => {
  return !props.identityProviderName || props.identityProviderName === "";
});

const userDocLink = computed(() => {
  if (state.type === IdentityProviderType.OAUTH2) {
    return "https://www.bytebase.com/docs/administration/sso/oauth2?source=console";
  }
  return "";
});

const templateList = computed(() => {
  if (state.type === IdentityProviderType.OAUTH2) {
    return identityProviderTemplateList.filter(
      (template) => template.type === IdentityProviderType.OAUTH2
    );
  }
  return [];
});

const originIdentityProvider = computed(() => {
  return identityProviderStore.getIdentityProviderByName(
    props.identityProviderName || ""
  );
});

const isFormCompleted = computed(() => {
  if (
    !identityProvider.value.name ||
    !identityProvider.value.title ||
    !identityProvider.value.domain
  ) {
    return false;
  }

  if (state.type === IdentityProviderType.OAUTH2) {
    if (
      !configForOAuth2.value.clientId ||
      !configForOAuth2.value.authUrl ||
      !configForOAuth2.value.tokenUrl ||
      !configForOAuth2.value.userInfoUrl ||
      !configForOAuth2.value.fieldMapping?.identifier ||
      // Only request client secret when creating.
      (isCreating.value && !configForOAuth2.value.clientSecret)
    ) {
      return false;
    }
  } else if (state.type === IdentityProviderType.OIDC) {
    if (
      !configForOIDC.value.clientId ||
      !configForOIDC.value.issuer ||
      !configForOIDC.value.fieldMapping?.identifier ||
      (isCreating.value && !configForOIDC.value.clientSecret)
    ) {
      return false;
    }
  } else {
    return false;
  }

  return true;
});

const allowCreate = computed(() => {
  if (!isFormCompleted.value) {
    return false;
  }
  return true;
});

const allowTestConnection = computed(() => {
  if (state.type === IdentityProviderType.OAUTH2) {
    if (isFormCompleted.value && configForOAuth2.value.clientSecret) {
      return true;
    }
  }
  return false;
});

const editedIdentityProvider = computed(() => {
  const tempIdentityProvider: IdentityProvider = {
    ...identityProvider.value,
    type: state.type,
    config: IdentityProviderConfig.fromPartial({}),
  };
  if (tempIdentityProvider.type === IdentityProviderType.OAUTH2) {
    tempIdentityProvider.config!.oauth2Config = {
      ...configForOAuth2.value,
      scopes: scopesStringOfConfig.value.split(" "),
    };
  } else if (tempIdentityProvider.type === IdentityProviderType.OIDC) {
    tempIdentityProvider.config!.oidcConfig = configForOIDC.value;
  } else {
    throw new Error(`identity provider type ${state.type} is invalid`);
  }
  return tempIdentityProvider;
});

const allowUpdate = computed(() => {
  if (!isFormCompleted.value) {
    return false;
  }
  if (isEqual(editedIdentityProvider.value, originIdentityProvider.value)) {
    return false;
  }
  return true;
});

onMounted(async () => {
  if (originIdentityProvider.value) {
    const tempIdentityProvider = cloneDeep(originIdentityProvider.value);
    identityProvider.value = tempIdentityProvider;
    state.type = tempIdentityProvider.type;
    if (tempIdentityProvider.type === IdentityProviderType.OAUTH2) {
      configForOAuth2.value =
        tempIdentityProvider.config?.oauth2Config ||
        OAuth2IdentityProviderConfig.fromPartial({
          fieldMapping: FieldMapping.fromPartial({}),
        });
      scopesStringOfConfig.value = configForOAuth2.value.scopes.join(" ");
    } else if (tempIdentityProvider.type === IdentityProviderType.OIDC) {
      configForOIDC.value =
        tempIdentityProvider.config?.oidcConfig ||
        OIDCIdentityProviderConfig.fromPartial({
          fieldMapping: FieldMapping.fromPartial({}),
        });
    }
  }
});

onMounted(() => {
  if (state.type === IdentityProviderType.OAUTH2) {
    window.addEventListener(
      `bb.oauth.signin.${editedIdentityProvider.value.name}`,
      loginWithIdentityProviderEventListener,
      false
    );
  }
});

onUnmounted(() => {
  if (state.type === IdentityProviderType.OAUTH2) {
    window.removeEventListener(
      `bb.oauth.signin.${editedIdentityProvider.value.name}`,
      loginWithIdentityProviderEventListener,
      false
    );
  }
});

const loginWithIdentityProviderEventListener = async (event: Event) => {
  const payload = (event as CustomEvent).detail as OAuthWindowEventPayload;
  if (payload.error) {
    return;
  }

  const code = payload.code;
  try {
    await identityProviderClient().testIdentityProvider({
      identityProvider: editedIdentityProvider.value,
      oauth2Context: {
        code: code,
      },
    });
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Request error occurred`,
      description: (error as ClientError).details,
    });
    return;
  }
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: "Test connection succeed",
  });
};

const copyRedirectUrl = () => {
  toClipboard(redirectUrl.value).then(() => {
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: t("settings.sso.copy-redirect-url"),
    });
  });
};

const testConnection = () => {
  if (state.type === IdentityProviderType.OAUTH2) {
    openWindowForSSO(editedIdentityProvider.value);
  }
};

const handleTemplateSelect = (template: IdentityProviderTemplate) => {
  if (template.type !== state.type) {
    return;
  }

  selectedTemplate.value = template;
  identityProvider.value.title = template.title;
  identityProvider.value.name = template.name;
  identityProvider.value.domain = template.domain;
  if (template.type === IdentityProviderType.OAUTH2) {
    configForOAuth2.value = {
      ...configForOAuth2.value,
      ...template.config,
    };
    scopesStringOfConfig.value = template.config.scopes.join(" ");
  }
};

const handleDeleteButtonClick = async () => {
  emit("delete", identityProvider.value);
};

const handleCancelButtonClick = () => {
  emit("cancel");
};

const updateEditState = (updatedIdentityProvider: IdentityProvider) => {
  const tempIdentityProvider = updatedIdentityProvider;
  identityProvider.value = tempIdentityProvider;
  state.type = tempIdentityProvider.type;
  if (tempIdentityProvider.type === IdentityProviderType.OAUTH2) {
    configForOAuth2.value =
      tempIdentityProvider.config?.oauth2Config ||
      OAuth2IdentityProviderConfig.fromPartial({
        fieldMapping: FieldMapping.fromPartial({}),
      });
    scopesStringOfConfig.value = configForOAuth2.value.scopes.join(" ");
  } else if (tempIdentityProvider.type === IdentityProviderType.OIDC) {
    configForOIDC.value =
      tempIdentityProvider.config?.oidcConfig ||
      OIDCIdentityProviderConfig.fromPartial({
        fieldMapping: FieldMapping.fromPartial({}),
      });
  }
};

const handleDiscardChangesButtonClick = async () => {
  if (originIdentityProvider.value) {
    updateEditState(originIdentityProvider.value);
  }
};

const handleCreateButtonClick = async () => {
  const identityProviderCreate: IdentityProvider = {
    ...identityProvider.value,
    type: state.type,
    config: {},
  };
  if (state.type === IdentityProviderType.OAUTH2) {
    configForOAuth2.value.scopes = scopesStringOfConfig.value.split(" ");
    identityProviderCreate.config!.oauth2Config = configForOAuth2.value;
  } else if (state.type === IdentityProviderType.OIDC) {
    identityProviderCreate.config!.oidcConfig = configForOIDC.value;
  } else {
    throw new Error(`identity provider type ${state.type} is invalid`);
  }

  const createdIdentityProvider =
    await identityProviderStore.createIdentityProvider(identityProviderCreate);
  emit("confirm", createdIdentityProvider);
};

const handleUpdateButtonClick = async () => {
  const updatedIdentityProvider =
    await identityProviderStore.patchIdentityProvider(
      editedIdentityProvider.value
    );
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: "Update SSO succeed",
  });
  updateEditState(updatedIdentityProvider);
};

watch(
  () => state.type,
  () => {
    if (!isCreating.value) {
      return;
    }
    if (state.type === IdentityProviderType.OAUTH2) {
      if (!selectedTemplate.value && head(templateList.value)) {
        handleTemplateSelect(
          head(templateList.value) as IdentityProviderTemplate
        );
      }
    }
  },
  {
    immediate: true,
  }
);
</script>
