<template>
  <div class="mx-auto w-full max-w-sm">
    <img
      class="h-12 w-auto mx-auto mb-8"
      src="../../assets/logo-full.svg"
      alt="Bytebase"
    />

    <div class="mt-4">
      <form class="space-y-6" @submit.prevent="trySignin">
        <div>
          <label
            for="email"
            class="block text-sm font-medium leading-5 text-control"
          >
            {{ $t("common.email") }}
            <span class="text-red-600">*</span>
          </label>
          <div class="mt-1 rounded-md shadow-sm">
            <input
              id="email"
              v-model="state.email"
              type="email"
              required
              placeholder="jim@example.com"
              class="appearance-none block w-full px-3 py-2 border border-control-border rounded-md placeholder-control-placeholder focus:outline-none focus:shadow-outline-blue focus:border-control-border sm:text-sm sm:leading-5"
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
              <span class="text-red-600">*</span>
            </div>
            <router-link
              to="/auth/password-forgot"
              class="text-sm font-normal text-control-light hover:underline focus:outline-none"
              >{{ $t("auth.sign-in.forget-password") }}</router-link
            >
          </label>
          <div
            class="relative flex flex-row items-center mt-1 rounded-md shadow-sm"
          >
            <input
              id="password"
              v-model="state.password"
              :type="state.showPassword ? 'text' : 'password'"
              autocomplete="on"
              required
              class="appearance-none block w-full px-3 py-2 border border-control-border rounded-md placeholder-control-placeholder focus:outline-none focus:shadow-outline-blue focus:border-control-border sm:text-sm sm:leading-5"
            />
            <div
              class="hover:cursor-pointer absolute right-3"
              @click="
                () => {
                  state.showPassword = !state.showPassword;
                }
              "
            >
              <heroicons-outline:eye
                v-if="state.showPassword"
                class="w-4 h-4"
              />
              <heroicons-outline:eye-slash v-else class="w-4 h-4" />
            </div>
          </div>
        </div>

        <div>
          <span class="flex w-full rounded-md items-center">
            <button
              type="submit"
              :disabled="!allowSignin"
              class="btn-primary justify-center flex-grow py-2 px-4"
            >
              {{ $t("common.sign-in") }}
            </button>
          </span>
        </div>
      </form>
    </div>

    <div class="mt-6 relative">
      <div class="relative flex justify-center text-sm">
        <template v-if="isDemo">
          <span class="pl-2 bg-white text-accent">{{
            $t("auth.sign-in.demo-note")
          }}</span>
        </template>
        <template v-else>
          <span class="pl-2 bg-white text-control">{{
            $t("auth.sign-in.new-user")
          }}</span>
          <router-link to="/auth/signup" class="accent-link bg-white px-2">{{
            $t("common.sign-up")
          }}</router-link>
        </template>
      </div>
    </div>

    <div v-if="identityProviderList.length > 0" class="mb-3">
      <div class="relative my-4">
        <div class="absolute inset-0 flex items-center" aria-hidden="true">
          <div class="w-full border-t border-control-border"></div>
        </div>
        <div class="relative flex justify-center text-sm">
          <span class="px-2 bg-white text-control">{{ $t("common.or") }}</span>
        </div>
      </div>
      <template
        v-for="identityProvider in identityProviderList"
        :key="identityProvider.name"
      >
        <button
          type="button"
          class="btn-normal flex justify-center w-full h-10 mb-2 tooltip-wrapper"
          :disabled="!has3rdPartyLoginFeature"
          @click.prevent="
            () => {
              trySigninWithIdentityProvider(identityProvider);
            }
          "
        >
          <span class="text-center align-middle">
            {{
              $t("auth.sign-in.sign-in-with-idp", {
                idp: identityProvider.title,
              })
            }}
          </span>
          <span v-if="isDemo" class="tooltip">{{
            $t("auth.sign-in.3rd-party-auth-demo")
          }}</span>
          <span v-else-if="!has3rdPartyLoginFeature" class="tooltip">{{
            $t("subscription.features.bb-feature-3rd-party-auth.login")
          }}</span>
        </button>
      </template>
    </div>
  </div>

  <AuthFooter />
</template>

<script lang="ts" setup>
import { computed, onMounted, onUnmounted, reactive } from "vue";
import { useRouter } from "vue-router";
import { storeToRefs } from "pinia";
import { OAuthWindowEventPayload } from "@/types";
import { isValidEmail, openWindowForSSO } from "@/utils";
import {
  featureToRef,
  useActuatorStore,
  useAuthStore,
  useIdentityProviderStore,
} from "@/store";
import {
  IdentityProvider,
  IdentityProviderType,
} from "@/types/proto/v1/idp_service";
import AuthFooter from "./AuthFooter.vue";

interface LocalState {
  email: string;
  password: string;
  showPassword: boolean;
  activeIdentityProvider?: IdentityProvider;
}

const actuatorStore = useActuatorStore();
const authStore = useAuthStore();
const identityProviderStore = useIdentityProviderStore();
const router = useRouter();

const state = reactive<LocalState>({
  email: "",
  password: "",
  showPassword: false,
});
const { isDemo } = storeToRefs(actuatorStore);

const identityProviderList = computed(
  () => identityProviderStore.identityProviderList
);

const loginWithIdentityProviderEventListener = async (event: Event) => {
  if (!state.activeIdentityProvider) {
    return;
  }

  if (state.activeIdentityProvider.type === IdentityProviderType.OAUTH2) {
    const payload = (event as CustomEvent).detail as OAuthWindowEventPayload;
    if (payload.error) {
      return;
    }
    const code = payload.code;
    await authStore.loginWithIdentityProvider({
      idpName: state.activeIdentityProvider.name,
      context: {
        oauth2Context: {
          code,
        },
      },
    });
    router.push("/");
  } else if (state.activeIdentityProvider.type === IdentityProviderType.OIDC) {
    // TODO
  }
};

onMounted(async () => {
  // Navigate to signup if needs admin setup.
  // Unable to achieve it in router.beforeEach because actuator/info is fetched async and returns
  // after router has already made the decision on first page load.
  if (actuatorStore.needAdminSetup) {
    router.push({ name: "auth.signup", replace: true });
  }

  if (isDemo.value) {
    state.email = "demo@example.com";
    state.password = "1024";
    state.showPassword = true;
  }

  await identityProviderStore.fetchIdentityProviderList();

  window.addEventListener(
    "bb.oauth.signin",
    loginWithIdentityProviderEventListener,
    false
  );
});

onUnmounted(() => {
  window.removeEventListener(
    "bb.oauth.signin",
    loginWithIdentityProviderEventListener
  );
});

const allowSignin = computed(() => {
  return isValidEmail(state.email) && state.password;
});

const trySignin = () => {
  authStore
    .login({
      email: state.email,
      password: state.password,
    })
    .then(() => {
      router.push("/");
    });
};

const trySigninWithIdentityProvider = (identityProvider: IdentityProvider) => {
  state.activeIdentityProvider = identityProvider;
  openWindowForSSO(identityProvider);
};

const has3rdPartyLoginFeature = featureToRef("bb.feature.3rd-party-auth");
</script>
