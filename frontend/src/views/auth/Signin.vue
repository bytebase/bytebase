<template>
  <div class="mx-auto w-full max-w-sm">
    <div>
      <img
        class="h-12 w-auto"
        src="../../assets/logo-full.svg"
        alt="Bytebase"
      />
    </div>

    <div class="mt-8 mb-3">
      <template
        v-for="authProvider in authProviderList"
        :key="authProvider.type"
      >
        <button
          type="button"
          class="btn-normal flex justify-center w-full h-10 mb-2 tooltip-wrapper"
          :disabled="!has3rdPartyLoginFeature"
          @click.prevent="
            () => {
              state.activeAuthProvider = authProvider;
              trySigninWithOAuth();
            }
          "
        >
          <img
            class="w-5 mr-1"
            :src="AuthProviderConfig[authProvider.type].iconPath"
          />
          <span class="text-center font-semibold align-middle">
            {{
              authProviderList.length == 1
                ? $t("auth.sign-in.gitlab")
                : authProvider.name
            }}
          </span>
          <span v-if="isDemo" class="tooltip">{{
            $t("auth.sign-in.gitlab-demo")
          }}</span>
          <span v-else-if="!has3rdPartyLoginFeature" class="tooltip">{{
            $t("subscription.features.bb-feature-3rd-party-auth.login")
          }}</span>
        </button>
      </template>

      <template v-if="authProviderList.length == 0">
        <button
          disabled
          type="button"
          class="btn-normal flex justify-center w-full h-10 mb-2"
        >
          <img
            class="w-5 mr-1"
            :src="AuthProviderConfig['GITLAB_SELF_HOST'].iconPath"
          />
          <span class="text-center font-semibold align-middle">
            {{ $t("auth.sign-in.gitlab-oauth") }}
          </span>
        </button>
      </template>
    </div>

    <div class="relative">
      <div class="absolute inset-0 flex items-center" aria-hidden="true">
        <div class="w-full border-t border-control-border"></div>
      </div>
      <div class="relative flex justify-center text-sm">
        <span class="px-2 bg-white text-control">{{ $t("common.or") }}</span>
      </div>
    </div>

    <div class="mt-2">
      <div class="mt-2">
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
            <div class="mt-1 rounded-md shadow-sm">
              <input
                id="password"
                v-model="state.password"
                type="password"
                autocomplete="on"
                required
                class="appearance-none block w-full px-3 py-2 border border-control-border rounded-md placeholder-control-placeholder focus:outline-none focus:shadow-outline-blue focus:border-control-border sm:text-sm sm:leading-5"
              />
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
  </div>

  <AuthFooter />
</template>

<script lang="ts">
import {
  computed,
  defineComponent,
  onMounted,
  onUnmounted,
  reactive,
} from "vue";
import { useRouter } from "vue-router";
import {
  AuthProvider,
  EmptyAuthProvider,
  VCSLoginInfo,
  LoginInfo,
  OAuthWindowEventPayload,
  openWindowForOAuth,
} from "../../types";
import { isDev, isValidEmail } from "../../utils";
import AuthFooter from "./AuthFooter.vue";
import { featureToRef, useActuatorStore, useAuthStore } from "@/store";
import { storeToRefs } from "pinia";

interface LocalState {
  email: string;
  password: string;
  activeAuthProvider: AuthProvider;
}

export default defineComponent({
  name: "SigninPage",
  components: { AuthFooter },
  setup() {
    const actuatorStore = useActuatorStore();
    const authStore = useAuthStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      email: "",
      password: "",
      activeAuthProvider: EmptyAuthProvider,
    });
    const { isDemo } = storeToRefs(actuatorStore);

    onMounted(() => {
      state.email = isDev() || isDemo.value ? "demo@example.com" : "";
      state.password = isDev() || isDemo.value ? "1024" : "";
      // Navigate to signup if needs admin setup.
      // Unable to achieve it in router.beforeEach because actuator/info is fetched async and returns
      // after router has already made the decision on first page load.
      if (actuatorStore.needAdminSetup) {
        router.push({ name: "auth.signup", replace: true });
      }

      authStore.fetchProviderList();

      window.addEventListener("bb.oauth.signin", eventListener, false);
    });

    onUnmounted(() => {
      window.removeEventListener("bb.oauth.signin", eventListener);
    });

    const allowSignin = computed(() => {
      return isValidEmail(state.email) && state.password;
    });

    const { authProviderList } = storeToRefs(authStore);

    const eventListener = (event: Event) => {
      const payload = (event as CustomEvent).detail as OAuthWindowEventPayload;
      if (payload.error) {
        return;
      }
      const gitlabLoginInfo: VCSLoginInfo = {
        vcsId: state.activeAuthProvider.id,
        name: state.activeAuthProvider.name,
        code: payload.code,
      };
      authStore
        .login({
          authProvider: "GITLAB_SELF_HOST",
          payload: gitlabLoginInfo,
        })
        .then(() => {
          router.push("/");
        });
    };

    const trySignin = () => {
      const loginInfo: LoginInfo = {
        authProvider: "BYTEBASE",
        payload: {
          email: state.email,
          password: state.password,
        },
      };
      authStore.login(loginInfo).then(() => {
        router.push("/");
      });
    };

    const AuthProviderConfig = {
      GITLAB_SELF_HOST: {
        apiPath: "oauth/authorize",
        // see https://vitejs.cn/guide/assets.html#the-public-directory for static resource import during run time
        iconPath: new URL("../../assets/gitlab-logo.svg", import.meta.url).href,
      },
    };

    const trySigninWithOAuth = () => {
      const authProvider = state.activeAuthProvider;

      // the following 3 lines is for a lint error
      if (authProvider.type == "BYTEBASE") {
        return;
      }

      openWindowForOAuth(
        `${authProvider.instanceUrl}/${
          AuthProviderConfig[authProvider.type].apiPath
        }`,
        authProvider.applicationId,
        "bb.oauth.signin",
        authProvider.type
      );
    };

    const has3rdPartyLoginFeature = featureToRef("bb.feature.3rd-party-auth");

    return {
      state,
      isDemo,
      allowSignin,
      authProviderList,
      AuthProviderConfig,
      trySignin,
      trySigninWithOAuth,
      has3rdPartyLoginFeature,
    };
  },
});
</script>
