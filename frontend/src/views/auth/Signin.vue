<template>
  <template v-if="initialized">
    <div class="mx-auto w-full max-w-sm">
      <BytebaseLogo class="mx-auto" />

      <div v-if="showSignInForm" class="mt-8">
        <NCard>
          <NTabs
            class="card-tabs"
            :default-value="defaultTabValue"
            size="small"
            animated
            pane-style="padding: 12px 0 0 0"
          >
            <NTabPane
              v-if="!disallowPasswordSignin"
              name="standard"
              tab="Standard"
            >
              <PasswordSigninForm
                :loading="state.isLoading"
                @signin="trySignin"
              />

              <div class="mt-3">
                <div
                  class="flex justify-center items-center text-sm text-control"
                >
                  <template v-if="isDemo">
                    <span class="text-accent">
                      {{
                        $t("auth.sign-in.demo-note", {
                          username: "demo@example.com",
                          password: "12345678",
                        })
                      }}
                    </span>
                  </template>
                  <template v-else-if="!disallowSignup">
                    <span>
                      {{ $t("auth.sign-in.new-user") }}
                    </span>
                    <router-link
                      :to="{ name: AUTH_SIGNUP_MODULE, query: route.query }"
                      class="accent-link px-2"
                    >
                      {{ $t("common.sign-up") }}
                    </router-link>
                  </template>
                </div>
              </div>
            </NTabPane>

            <template
              v-for="identityProvider in groupedIdentityProviderList"
              :key="identityProvider.name"
            >
              <NTabPane
                v-if="identityProvider.type === IdentityProviderType.LDAP"
                :name="identityProvider.name"
                :tab="identityProvider.title"
              >
                <form
                  class="flex flex-col gap-y-6"
                  @submit.prevent="trySigninWithIdp(identityProvider.name)"
                >
                  <div>
                    <label
                      for="email"
                      class="block text-sm font-medium leading-5 text-control"
                    >
                      {{ $t("common.username") }}
                      <RequiredStar />
                    </label>
                    <div class="mt-1 rounded-md shadow-xs">
                      <BBTextField
                        v-model:value="state.email"
                        required
                        placeholder="jim"
                        :input-props="{ id: 'username', autocomplete: 'on' }"
                      />
                    </div>
                  </div>

                  <div>
                    <label
                      for="password"
                      class="flex justify-between text-sm font-medium leading-5 text-control"
                    >
                      <div>
                        {{ $t("common.password") }}
                        <RequiredStar />
                      </div>
                    </label>
                    <div
                      class="relative flex flex-row items-center mt-1 rounded-md shadow-xs"
                    >
                      <BBTextField
                        v-model:value="state.password"
                        :type="state.showPassword ? 'text' : 'password'"
                        :input-props="{ id: 'password', autocomplete: 'on' }"
                        required
                      />
                      <div
                        class="hover:cursor-pointer absolute right-3"
                        @click="
                          () => {
                            state.showPassword = !state.showPassword;
                          }
                        "
                      >
                        <EyeIcon v-if="state.showPassword" class="w-4 h-4" />
                        <EyeOffIcon v-else class="w-4 h-4" />
                      </div>
                    </div>
                  </div>

                  <div class="w-full">
                    <NButton
                      attr-type="submit"
                      type="primary"
                      size="large"
                      :disabled="!allowIdPSignin"
                      :loading="state.isLoading"
                      style="width: 100%"
                    >
                      {{ $t("common.sign-in") }}
                    </NButton>
                  </div>
                </form>
              </NTabPane>
            </template>
          </NTabs>
        </NCard>
      </div>

      <div v-if="separatedIdentityProviderList.length > 0" class="mb-3 px-1">
        <div v-if="showSignInForm" class="relative my-4">
          <div class="absolute inset-0 flex items-center" aria-hidden="true">
            <div class="w-full border-t border-control-border"></div>
          </div>
          <div class="relative flex justify-center text-sm">
            <span class="px-2 bg-white text-control">{{
              $t("common.or")
            }}</span>
          </div>
        </div>
        <template
          v-for="identityProvider in separatedIdentityProviderList"
          :key="identityProvider.name"
        >
          <div class="w-full mb-2">
            <NButton
              style="width: 100%"
              size="large"
              @click.prevent="trySigninWithIdentityProvider(identityProvider)"
            >
              {{
                $t("auth.sign-in.sign-in-with-idp", {
                  idp: identityProvider.title,
                })
              }}
            </NButton>
          </div>
        </template>
      </div>
    </div>
    <slot name="footer">
      <AuthFooter />
    </slot>
  </template>
  <template v-else>
    <div class="inset-0 absolute flex flex-row justify-center items-center">
      <BBSpin />
    </div>
  </template>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import type { ConnectError } from "@connectrpc/connect";
import { EyeIcon, EyeOffIcon } from "lucide-vue-next";
import { NButton, NCard, NTabPane, NTabs } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, onMounted, reactive, ref, watchEffect } from "vue";
import { useRoute, useRouter } from "vue-router";
import { BBSpin, BBTextField } from "@/bbkit";
import BytebaseLogo from "@/components/BytebaseLogo.vue";
import PasswordSigninForm from "@/components/PasswordSigninForm.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import { AUTH_SIGNUP_MODULE } from "@/router/auth";
import {
  pushNotification,
  useActuatorV1Store,
  useAuthStore,
  useIdentityProviderStore,
} from "@/store";
import { idpNamePrefix } from "@/store/modules/v1/common";
import {
  type LoginRequest,
  LoginRequestSchema,
} from "@/types/proto-es/v1/auth_service_pb";
import type { IdentityProvider } from "@/types/proto-es/v1/idp_service_pb";
import { IdentityProviderType } from "@/types/proto-es/v1/idp_service_pb";
import { openWindowForSSO } from "@/utils";
import AuthFooter from "./AuthFooter.vue";

const props = withDefaults(
  defineProps<{
    redirect?: boolean;
    redirectUrl?: string;
    allowSignup?: boolean;
  }>(),
  {
    redirect: true,
    allowSignup: true,
  }
);

interface LocalState {
  email: string;
  password: string;
  showPassword: boolean;
  isLoading: boolean;
}

const router = useRouter();
const route = useRoute();
const actuatorStore = useActuatorV1Store();
const authStore = useAuthStore();
const identityProviderStore = useIdentityProviderStore();

const state = reactive<LocalState>({
  email: "",
  password: "",
  showPassword: false,
  isLoading: false,
});
const initialized = ref(false);
const { isDemo, disallowPasswordSignin } = storeToRefs(actuatorStore);

const disallowSignup = computed(
  () => !props.allowSignup || actuatorStore.disallowSignup
);

const separatedIdentityProviderList = computed(() =>
  identityProviderStore.identityProviderList.filter(
    (idp) => idp.type !== IdentityProviderType.LDAP
  )
);
const groupedIdentityProviderList = computed(() =>
  identityProviderStore.identityProviderList.filter(
    (idp) => idp.type === IdentityProviderType.LDAP
  )
);

const showSignInForm = computed(() => {
  return (
    !disallowPasswordSignin.value ||
    groupedIdentityProviderList.value.length > 0
  );
});

const defaultTabValue = computed(() => {
  if (!disallowPasswordSignin.value) {
    return "standard";
  }
  if (groupedIdentityProviderList.value.length > 0) {
    return groupedIdentityProviderList.value[0].name;
  }
  return "standard";
});

watchEffect(() => {
  // Navigate to signup if needs admin setup.
  // Unable to achieve it in router.beforeEach because actuator/info is fetched async and returns
  // after router has already made the decision on first page load.
  if (actuatorStore.needAdminSetup && !disallowSignup.value) {
    router.push({ name: AUTH_SIGNUP_MODULE, replace: true });
  }
});

onMounted(async () => {
  try {
    // Prepare all identity providers.
    await identityProviderStore.fetchIdentityProviderList();
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Request error occurred`,
      description: (error as ConnectError).message,
    });
  }
  // Check if there is an identity provider in the query string and try to sign in with it.
  if (route.query["idp"]) {
    const idpName = `${idpNamePrefix}${route.query["idp"] as string}`;
    const identityProvider = identityProviderStore.identityProviderList.find(
      (idp) => idp.name === idpName
    );
    if (identityProvider) {
      await trySigninWithIdentityProvider(identityProvider);
      // If we successfully signed in with the identity provider, return early.
      return;
    }
  }
  initialized.value = true;
});

const allowIdPSignin = computed(() => {
  return state.email && state.password;
});

// Mainly for LDAP signin.
const trySigninWithIdp = (idpName: string) => {
  trySignin(
    create(LoginRequestSchema, {
      email: state.email,
      password: state.password,
      idpName: idpName,
    })
  );
};

const trySignin = async (request: LoginRequest) => {
  if (state.isLoading) return;
  state.isLoading = true;
  try {
    await authStore.login({
      request,
      redirect: props.redirect,
      redirectUrl: props.redirectUrl,
    });
  } finally {
    state.isLoading = false;
  }
};

const trySigninWithIdentityProvider = async (
  identityProvider: IdentityProvider
) => {
  try {
    await openWindowForSSO(
      identityProvider,
      false /* !popup */,
      route.query.redirect as string
    );
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Request error occurred`,
      description: (error as ConnectError).message,
    });
  }
};
</script>
