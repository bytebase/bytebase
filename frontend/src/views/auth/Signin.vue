<template>
  <div class="mx-auto w-full max-w-sm">
    <img
      class="h-12 w-auto mx-auto mb-8"
      src="../../assets/logo-full.svg"
      alt="Bytebase"
    />

    <div class="mt-8">
      <n-card>
        <n-tabs
          class="card-tabs"
          default-value="standard"
          size="small"
          animated
          pane-style="padding: 12px 2px 0 2px"
        >
          <n-tab-pane name="standard" tab="Standard">
            <form class="space-y-6" @submit.prevent="trySignin()">
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
                    tabindex="-1"
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
                    :disabled="!allowSignin()"
                    class="btn-primary justify-center flex-grow py-2 px-4"
                  >
                    {{ $t("common.sign-in") }}
                  </button>
                </span>
              </div>
            </form>

            <div class="mt-6 relative">
              <div class="relative flex justify-center text-sm">
                <template v-if="isDemo">
                  <span class="pl-2 bg-white text-accent">{{
                    $t("auth.sign-in.demo-note", {
                      username: "demo@example.com",
                      password: "1024",
                    })
                  }}</span>
                </template>
                <template v-else-if="!disallowSignup">
                  <span class="pl-2 bg-white text-control">{{
                    $t("auth.sign-in.new-user")
                  }}</span>
                  <router-link
                    to="/auth/signup"
                    class="accent-link bg-white px-2"
                    >{{ $t("common.sign-up") }}</router-link
                  >
                </template>
              </div>
            </div>
          </n-tab-pane>

          <template
            v-for="identityProvider in groupedIdentityProviderList"
            :key="identityProvider.name"
          >
            <n-tab-pane
              v-if="identityProvider.type === IdentityProviderType.LDAP"
              name="ldap"
              tab="LDAP"
            >
              <form
                class="space-y-6"
                @submit.prevent="trySignin(identityProvider.name)"
              >
                <div>
                  <label
                    for="email"
                    class="block text-sm font-medium leading-5 text-control"
                  >
                    {{ $t("common.username") }}
                    <span class="text-red-600">*</span>
                  </label>
                  <div class="mt-1 rounded-md shadow-sm">
                    <input
                      id="username"
                      v-model="state.email"
                      type="text"
                      required
                      placeholder="jim"
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
                      :disabled="!allowSignin(identityProvider.name)"
                      class="btn-primary justify-center flex-grow py-2 px-4"
                    >
                      {{ $t("common.sign-in") }}
                    </button>
                  </span>
                </div>
              </form>
            </n-tab-pane>
          </template>
        </n-tabs>
      </n-card>
    </div>

    <div v-if="separatedIdentityProviderList.length > 0" class="mb-3">
      <div class="relative my-4">
        <div class="absolute inset-0 flex items-center" aria-hidden="true">
          <div class="w-full border-t border-control-border"></div>
        </div>
        <div class="relative flex justify-center text-sm">
          <span class="px-2 bg-white text-control">{{ $t("common.or") }}</span>
        </div>
      </div>
      <template
        v-for="identityProvider in separatedIdentityProviderList"
        :key="identityProvider.name"
      >
        <button
          type="button"
          class="btn-normal flex justify-center w-full h-10 mb-2 tooltip-wrapper"
          @click.prevent="trySigninWithIdentityProvider(identityProvider)"
        >
          <span class="text-center align-middle">
            {{
              $t("auth.sign-in.sign-in-with-idp", {
                idp: identityProvider.title,
              })
            }}
          </span>
        </button>
      </template>
    </div>
  </div>
  <AuthFooter />
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive } from "vue";
import { useRoute, useRouter } from "vue-router";
import { storeToRefs } from "pinia";
import { isValidEmail, openWindowForSSO } from "@/utils";
import {
  useActuatorV1Store,
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
});
const { isDemo, disallowSignup } = storeToRefs(actuatorStore);

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

onMounted(async () => {
  // Navigate to signup if needs admin setup.
  // Unable to achieve it in router.beforeEach because actuator/info is fetched async and returns
  // after router has already made the decision on first page load.
  if (actuatorStore.needAdminSetup && !disallowSignup.value) {
    router.push({ name: "auth.signup", replace: true });
  }

  if (isDemo.value) {
    state.email = "demo@example.com";
    state.password = "1024";
    state.showPassword = true;
  } else {
    const url = new URL(window.location.href);
    const params = new URLSearchParams(url.search);
    state.email = params.get("email") ?? "";
    state.password = params.get("password") ?? "";
    state.showPassword = false;
  }

  await identityProviderStore.fetchIdentityProviderList();
  if (state.email && state.password) {
    await trySignin();
  }
});

const allowSignin = (idpName?: string) => {
  if (!idpName) {
    return isValidEmail(state.email) && state.password;
  }
  return state.email && state.password;
};

const trySignin = async (idpName?: string) => {
  const mfaTempToken = await authStore.login({
    email: state.email,
    password: state.password,
    web: true,
    idpName: idpName,
  });
  if (mfaTempToken) {
    router.push({
      name: "auth.mfa",
      query: {
        mfaTempToken,
        redirect: route.query.redirect as string,
      },
    });
  } else {
    router.push("/");
  }
};

const trySigninWithIdentityProvider = async (
  identityProvider: IdentityProvider
) => {
  await openWindowForSSO(
    identityProvider,
    false /* !popup */,
    route.query.redirect as string
  );
};
</script>
