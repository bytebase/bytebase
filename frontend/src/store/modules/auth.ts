import { defineStore, storeToRefs } from "pinia";
import axios from "axios";
import { isEqual } from "lodash-es";
import { computed, Ref } from "vue";
import {
  Principal,
  AuthState,
  AuthProviderType,
  VCSLoginInfo,
  SignupInfo,
  ActivateInfo,
  ResourceObject,
  unknown,
  PrincipalId,
  AuthProvider,
  BytebaseLoginInfo,
} from "@/types";
import { getIntCookie } from "@/utils";
import { authServiceClient } from "@/grpcweb";
import { usePrincipalStore } from "./principal";

function convert(user: ResourceObject): Principal {
  return usePrincipalStore().principalById(parseInt(user.id, 10));
}

function convertAuthProvider(authProvider: ResourceObject) {
  return { ...authProvider.attributes } as AuthProvider;
}

export const useAuthStore = defineStore("auth", {
  state: (): AuthState => ({
    authProviderList: [],
    currentUser: unknown("PRINCIPAL") as Principal,
  }),
  actions: {
    isLoggedIn: () => {
      return getIntCookie("user") != undefined;
    },
    setAuthProviderList(authProviderList: AuthProvider[]) {
      this.authProviderList = authProviderList;
    },
    setCurrentUser(user: Principal) {
      this.currentUser = user;
    },
    async fetchProviderList() {
      const providerList = (await axios.get("/api/auth/provider")).data.data;
      const convertedProviderList: AuthProvider[] = providerList.map(
        (provider: ResourceObject) => {
          return convertAuthProvider(provider);
        }
      );
      this.setAuthProviderList(convertedProviderList);
      return convertedProviderList;
    },
    async login(loginInfo: BytebaseLoginInfo) {
      await authServiceClient.login({
        email: loginInfo.email,
        password: loginInfo.password,
        web: true,
      });
      const userId = getIntCookie("user");
      if (userId) {
        const loggedInUser = await usePrincipalStore().fetchPrincipalById(
          userId
        );

        this.setCurrentUser(loggedInUser);
        return loggedInUser;
      }
      return unknown("PRINCIPAL") as Principal;
    },
    async vcsLogin(
      authProviderType: AuthProviderType,
      loginInfo: VCSLoginInfo
    ) {
      const loggedInUser = (
        await axios.post(`/api/auth/login/${authProviderType}`, {
          data: { type: "loginInfo", attributes: loginInfo },
        })
      ).data.data;
      // Refresh the corresponding principal
      await usePrincipalStore().fetchPrincipalById(loggedInUser.id);
      // The conversion relies on the above refresh.
      const convertedUser = convert(loggedInUser);
      this.setCurrentUser(convertedUser);
      return convertedUser;
    },
    async logout() {
      const unknownPrincipal = unknown("PRINCIPAL") as Principal;
      try {
        await authServiceClient.logout({});
      } finally {
        this.setCurrentUser(unknownPrincipal);
      }
      return unknownPrincipal;
    },
    async signup(signupInfo: SignupInfo) {
      await authServiceClient.createUser({
        user: {
          email: signupInfo.email,
          title: signupInfo.name,
          password: signupInfo.password,
        },
      });
      await authServiceClient.login({
        email: signupInfo.email,
        password: signupInfo.password,
        web: true,
      });
      const userId = getIntCookie("user");
      if (userId) {
        const loggedInUser = await usePrincipalStore().fetchPrincipalById(
          userId
        );

        this.setCurrentUser(loggedInUser);
        return loggedInUser;
      }
      return unknown("PRINCIPAL") as Principal;
    },
    async activate(activateInfo: ActivateInfo) {
      const activatedUser = (
        await axios.post("/api/auth/activate", {
          data: { type: "activateInfo", attributes: activateInfo },
        })
      ).data.data;

      // Refresh the corresponding principal
      await usePrincipalStore().fetchPrincipalById(activatedUser.id);

      // The conversion relies on the above task to get the latest data
      const convertedUser = convert(activatedUser);
      this.setCurrentUser(convertedUser);
      return convertedUser;
    },
    async restoreUser() {
      const userId = getIntCookie("user");
      if (userId) {
        const loggedInUser = await usePrincipalStore().fetchPrincipalById(
          userId
        );

        this.setCurrentUser(loggedInUser);
        return loggedInUser;
      }
      return unknown("PRINCIPAL") as Principal;
    },
    async refreshUserIfNeeded(principalId: PrincipalId) {
      if (principalId == this.currentUser.id) {
        const refreshedUser = usePrincipalStore().principalById(
          this.currentUser.id
        );
        if (!isEqual(refreshedUser, this.currentUser)) {
          this.setCurrentUser(refreshedUser);
        }
      }
    },
  },
});

export const useCurrentUser = (): Ref<Principal> => {
  return storeToRefs(useAuthStore()).currentUser;
};

export const useIsLoggedIn = (): Ref<boolean> => {
  const store = useAuthStore();
  return computed(() => store.isLoggedIn());
};
