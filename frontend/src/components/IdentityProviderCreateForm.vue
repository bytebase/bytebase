<template>
  <div
    class="w-full flex flex-col justify-start items-start"
    :class="[isCreating && '!w-128']"
  >
    <div
      v-if="isCreating"
      class="w-full flex flex-col justify-start items-start"
    >
      <p class="textinfolabel my-2">{{ $t("settings.sso.form.type") }}</p>
      <div class="w-full flex flex-row justify-start items-start space-x-2">
        <label
          v-for="item in identityProviderTypeList"
          :key="item"
          class="w-24 h-24 border rounded flex flex-col justify-center items-center cursor-pointer"
          :for="`radio-${item}`"
        >
          <span>{{ identityProviderTypeToString(item) }}</span>
          <input
            :id="`radio-${item}`"
            v-model="state.type"
            type="radio"
            class="btn mt-4"
            :value="item"
            :checked="state.type === item"
          />
        </label>
      </div>

      <p class="textinfolabel !mt-4">Authorization callback URL</p>
      <input
        type="text"
        class="textfield mt-1 w-full"
        readonly
        :value="callbackUrl"
      />
    </div>

    <!-- OAuth2 form group -->
    <div
      v-if="state.type === IdentityProviderType.OAUTH2"
      class="w-full flex flex-col justify-start items-start space-y-3"
    >
      <p class="textinfolabel !mt-4">
        {{ $t("settings.sso.form.basic-information") }}
      </p>
      <div class="w-full flex flex-col justify-start items-start">
        <label for="">{{ $t("settings.sso.form.name") }}</label>
        <input
          v-model="identityProvider.title"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <label for="">{{ $t("settings.sso.form.domain") }}</label>
        <input
          v-model="identityProvider.domain"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <label for="">{{ $t("settings.sso.form.resource-id") }}</label>
        <input
          v-model="identityProvider.name"
          type="text"
          class="textfield mt-1 w-full"
          :disabled="!isCreating"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <label for="">Client ID</label>
        <input
          v-model="configForOAuth2.clientId"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <label for="">Client secret</label>
        <input
          v-model="configForOAuth2.clientSecret"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>

      <p class="textinfolabel !mt-4">Endpoints</p>
      <div class="w-full flex flex-col justify-start items-start">
        <label for="">Auth URL</label>
        <input
          v-model="configForOAuth2.authUrl"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <label for="">Token URL</label>
        <input
          v-model="configForOAuth2.tokenUrl"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <label for="">User information URL</label>
        <input
          v-model="configForOAuth2.userInfoUrl"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <label for="">Scopes</label>
        <input
          v-model="scopesStringOfConfig"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>

      <p class="textinfolabel !mt-4">
        {{ $t("settings.sso.form.user-information-mapping") }}
      </p>
      <div class="w-full flex flex-col justify-start items-start">
        <label for="">{{ $t("settings.sso.form.identifier") }}</label>
        <input
          v-model="configForOAuth2.fieldMapping!.identifier"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <label for="">{{ $t("settings.sso.form.display-name") }}</label>
        <input
          v-model="configForOAuth2.fieldMapping!.displayName"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <label for="">Email</label>
        <input
          v-model="configForOAuth2.fieldMapping!.email"
          type="text"
          class="textfield mt-1 w-full"
        />
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
        <label for="">{{ $t("settings.sso.form.name") }}</label>
        <input
          v-model="identityProvider.title"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <label for="">{{ $t("settings.sso.form.domain-name") }}</label>
        <input
          v-model="identityProvider.domain"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <label for="">{{ $t("settings.sso.form.resource-id") }}</label>
        <input
          v-model="identityProvider.name"
          type="text"
          class="textfield mt-1 w-full"
          :disabled="!isCreating"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <label for="">Issuer</label>
        <input
          v-model="configForOIDC.issuer"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <label for="">Client ID</label>
        <input
          v-model="configForOIDC.clientId"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <label for="">Client secret</label>
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
        <label for="">{{ $t("settings.sso.form.identifier") }}</label>
        <input
          v-model="configForOIDC.fieldMapping!.identifier"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <label for="">{{ $t("settings.sso.form.display-name") }}</label>
        <input
          v-model="configForOIDC.fieldMapping!.displayName"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <label for="">Email</label>
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
import { cloneDeep, isEqual } from "lodash-es";
import {
  computed,
  reactive,
  defineEmits,
  defineProps,
  ref,
  onMounted,
} from "vue";
import {
  FieldMapping,
  IdentityProvider,
  IdentityProviderConfig,
  IdentityProviderType,
  OAuth2IdentityProviderConfig,
  OIDCIdentityProviderConfig,
} from "@/types/proto/v1/idp_service";
import {
  identityProviderTypeToString,
  useIdentityProviderStore,
} from "@/store/modules/idp";
import { useActuatorStore } from "@/store";

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
const identityProviderTypeList = [
  IdentityProviderType.OAUTH2,
  IdentityProviderType.OIDC,
];

const callbackUrl = computed(() => {
  return `${
    useActuatorStore().serverInfo?.externalUrl || window.origin
  }/oauth/callback`;
});

const isCreating = computed(() => {
  return !props.identityProviderName || props.identityProviderName === "";
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
      !configForOAuth2.value.clientSecret ||
      !configForOAuth2.value.scopes ||
      !configForOAuth2.value.authUrl ||
      !configForOAuth2.value.tokenUrl ||
      !configForOAuth2.value.userInfoUrl ||
      !configForOAuth2.value.fieldMapping?.identifier ||
      !configForOAuth2.value.fieldMapping?.displayName ||
      !configForOAuth2.value.fieldMapping?.email
    ) {
      return false;
    }
  } else if (state.type === IdentityProviderType.OIDC) {
    if (
      !configForOIDC.value.clientId ||
      !configForOIDC.value.clientSecret ||
      !configForOIDC.value.issuer ||
      !configForOIDC.value.fieldMapping?.identifier ||
      !configForOIDC.value.fieldMapping?.displayName ||
      !configForOIDC.value.fieldMapping?.email
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

const updatedIdentityProvider = computed(() => {
  const tempIdentityProvider: IdentityProvider = {
    ...identityProvider.value,
    config: IdentityProviderConfig.fromPartial({}),
  };
  if (state.type === IdentityProviderType.OAUTH2) {
    tempIdentityProvider.config!.oauth2Config = {
      ...configForOAuth2.value,
      scopes: scopesStringOfConfig.value.split(" "),
    };
  } else if (state.type === IdentityProviderType.OIDC) {
    tempIdentityProvider.config!.oidcConfig = configForOIDC.value;
  } else {
    // should not reach here.
  }
  return tempIdentityProvider;
});

const allowUpdate = computed(() => {
  if (!isFormCompleted.value) {
    return false;
  }
  if (isEqual(updatedIdentityProvider.value, originIdentityProvider.value)) {
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

const handleDeleteButtonClick = async () => {
  emit("delete", identityProvider.value);
};

const handleCancelButtonClick = () => {
  emit("cancel");
};

const handleDiscardChangesButtonClick = async () => {
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
    // should not reach here.
  }

  const createdIdentityProvider =
    await identityProviderStore.createIdentityProvider(identityProviderCreate);
  emit("confirm", createdIdentityProvider);
};

const handleUpdateButtonClick = async () => {
  await identityProviderStore.patchIdentityProvider(
    updatedIdentityProvider.value
  );
};
</script>
