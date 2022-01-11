<template>
  <div class="mx-auto w-full max-w-sm">
    <div>
      <img
        class="h-12 w-auto"
        src="../../assets/logo-full.svg"
        alt="Bytebase"
      />
      <h2 class="mt-6 text-3xl leading-9 font-extrabold text-main">
        {{ $t("auth.sign-in.title") }}
      </h2>
      <h2
        v-if="authProviderList.length == 0"
        class="text-gray-500 leading-4 tracking-wide font-light text-sm"
      >
        {{ $t("auth.sign-in.third-party") }}
      </h2>
    </div>

    <div class="mt-4">
      <div class="mt-4">
        <form class="space-y-6" @submit.prevent="trySignin">
          <div>
            <label
              for="email"
              class="block text-sm font-medium leading-5 text-control"
            >
              {{ $t("common.email") }}<span class="text-red-600">*</span>
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
                {{ $t("common.password") }}<span class="text-red-600">*</span>
              </div>
              <router-link
                to="/auth/password-forgot"
                class="text-sm font-normal text-control-light hover:underline focus:outline-none"
              >
                {{ $t("auth.sign-in.forget-password") }}
              </router-link>
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

              <template
                v-for="authProvider in authProviderList"
                :key="authProvider.type"
              >
                <!-- GitLab auth provider -->
                <n-button
                  circle
                  quaternary
                  :bordered="false"
                  @click.prevent="
                    () => {
                      state.activeAuthProvider = authProvider;
                      const window = openWindowForOAuth(
                        `${authProvider.instanceUrl}/${
                          AuthProviderConfig[authProvider.type].apiPath
                        }`,
                        authProvider.applicationId,
                        'login'
                      );
                    }
                  "
                >
                  <img
                    class="w-4"
                    :src="AuthProviderConfig[authProvider.type].iconPath"
                  />
                </n-button>
              </template>
            </span>
          </div>
        </form>
      </div>
    </div>

    <div class="mt-6 relative">
      <div class="absolute inset-0 flex items-center" aria-hidden="true">
        <div class="w-full border-t border-control-border"></div>
      </div>
      <div class="relative flex justify-center text-sm">
        <span class="pl-2 bg-white text-control">
          {{ $t("auth.sign-in.new-user") }}
        </span>
        <router-link to="/auth/signup" class="accent-link bg-white px-2">
          {{ $t("common.sign-up") }}
        </router-link>
      </div>
    </div>
  </div>

  <AuthFooter />
</template>

<script lang="ts">
import { computed, onMounted, reactive } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import {
  AuthProvider,
  EmptyAuthProvider,
  VCSLoginInfo,
  LoginInfo,
  OAuthConfig,
  OAuthToken,
  OAuthWindowEvent,
  OAuthWindowEventPayload,
  openWindowForOAuth,
  redirectUrl,
} from "../../types";
import { isDev, isValidEmail } from "../../utils";
import AuthFooter from "./AuthFooter.vue";

interface LocalState {
  email: string;
  password: string;
  activeAuthProvider: AuthProvider;
}

export default {
  name: "SigninPage",
  components: { AuthFooter },
  setup() {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      email: "",
      password: "",
      activeAuthProvider: EmptyAuthProvider,
    });

    onMounted(() => {
      const demo = store.getters["actuator/isDemo"]();
      state.email = isDev() || demo ? "demo@example.com" : "";
      state.password = isDev() || demo ? "1024" : "";
      // Navigate to signup if needs admin setup.
      // Unable to achieve it in router.beforeEach because actuator/info is fetched async and returns
      // after router has already made the decision on first page load.
      if (store.getters["actuator/needAdminSetup"]()) {
        router.push({ name: "auth.signup", replace: true });
      }

      store.dispatch("auth/fetchProviderList");
      window.addEventListener(OAuthWindowEvent, eventListener, false);
    });

    const AuthProviderConfig = {
      GITLAB_SELF_HOST: {
        apiPath: "oauth/authorize",
        // see https://vitejs.cn/guide/assets.html#the-public-directory for static resource import during run time
        iconPath: new URL("../../assets/gitlab-logo.svg", import.meta.url).href,
      },
    };

    const allowSignin = computed(() => {
      return isValidEmail(state.email) && state.password;
    });

    const authProviderList = computed(() => {
      return store.getters["auth/authProviderList"]();
    });

    const eventListener = (event: Event) => {
      const payload = (event as CustomEvent).detail as OAuthWindowEventPayload;
      if (payload.error) {
        return;
      }
      const oAuthConfig: OAuthConfig = {
        endpoint: `${state.activeAuthProvider.instanceUrl}/oauth/token`,
        applicationId: state.activeAuthProvider.applicationId,
        secret: state.activeAuthProvider.secret,
        redirectUrl: redirectUrl(),
      };
      store
        .dispatch("gitlab/exchangeToken", {
          oAuthConfig,
          code: payload.code,
        })
        .then((token: OAuthToken) => {
          const gitlabLoginInfo: VCSLoginInfo = {
            applicationId: state.activeAuthProvider!.applicationId,
            secret: state.activeAuthProvider.secret,
            instanceUrl: state.activeAuthProvider.instanceUrl,
            accessToken: token.accessToken,
          };

          store
            .dispatch("auth/login", {
              authProvider: "GITLAB_SELF_HOST",
              payload: gitlabLoginInfo,
            })
            .then(() => {
              router.push("/");
            });
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
      store.dispatch("auth/login", loginInfo).then(() => {
        router.push("/");
      });
    };

    return {
      state,
      allowSignin,
      authProviderList,
      AuthProviderConfig,
      trySignin,
      openWindowForOAuth,
    };
  },
};
</script>
