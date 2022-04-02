<template>
  <div class="space-y-4">
    <div class="space-y-2">
      <div
        v-for="(user, index) in state.userList"
        :key="index"
        class="flex justify-between py-0.5 select-none"
      >
        <p id="add_or_invite_members_helper" class="sr-only">
          {{ $t("settings.members.helper") }}
        </p>
        <div class="flex flex-row space-x-4">
          <div class="flex-grow">
            <input
              v-model="user.email"
              type="email"
              name="add_or_invite_members"
              autocomplete="off"
              class="w-36 sm:w-64 textfield lowercase"
              placeholder="foo@example.com"
              aria-describedby="add_or_invite_members_helper"
              @blur="validateUser(user, index)"
              @input="clearValidationError(index)"
            />
            <p
              v-if="state.errorList[index]"
              id="email-error"
              class="mt-2 text-sm text-error"
            >
              {{ state.errorList[index] }}
            </p>
          </div>
          <div v-if="hasRBACFeature" class="sm:hidden w-36">
            <RoleSelect
              :selected-role="user.role"
              @change-role="
                (role) => {
                  user.role = role;
                }
              "
            />
          </div>
          <div
            v-if="hasRBACFeature"
            class="hidden sm:flex sm:flex-row space-x-4"
            :class="state.errorList[index] ? '-mt-7' : ''"
          >
            <div class="radio">
              <input
                v-model="user.role"
                :name="`add_or_invite_role${index}`"
                tabindex="-1"
                type="radio"
                class="btn"
                value="OWNER"
              />
              <label class="label">{{ $t("common.role.owner") }}</label>
            </div>
            <div class="radio">
              <input
                v-model="user.role"
                :name="`add_or_invite_role${index}`"
                tabindex="-1"
                type="radio"
                class="btn"
                value="DBA"
              />
              <label class="label">{{ $t("common.role.dba") }}</label>
            </div>
            <div class="radio">
              <input
                v-model="user.role"
                :name="`add_or_invite_role${index}`"
                tabindex="-1"
                type="radio"
                class="btn"
                value="DEVELOPER"
              />
              <label class="label">{{ $t("common.role.developer") }}</label>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="flex justify-between">
      <span class="flex items-center">
        <button type="button" class="btn-secondary" @click.prevent="addUser">
          {{ $t("settings.members.add-more") }}
        </button>
      </span>

      <button
        type="button"
        class="btn-primary"
        :disabled="!hasValidUserOnly()"
        @click.prevent="addOrInvite"
      >
        <heroicons-solid:plus v-if="isAdd" class="h-5 w-5" />
        <heroicons-outline:mail v-else class="mr-2 h-5 w-5" />
        {{ $t(`settings.members.${isAdd ? "add" : "invites"}`) }}
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, reactive } from "vue";
import { useStore } from "vuex";
import RoleSelect from "./RoleSelect.vue";
import {
  Principal,
  PrincipalCreate,
  MemberCreate,
  RoleType,
  UNKNOWN_ID,
} from "../types";
import { isOwner, isValidEmail } from "../utils";
import { useUIStateStore } from "@/store";

type User = {
  email: string;
  role: RoleType;
};

interface LocalState {
  userList: User[];
  errorList: string[];
}

export default defineComponent({
  name: "MemberAddOrInvite",
  components: { RoleSelect },
  props: {},
  setup() {
    const store = useStore();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const isAdd = computed(() => {
      return isOwner(currentUser.value.role);
    });

    const hasRBACFeature = computed(() =>
      store.getters["subscription/feature"]("bb.feature.rbac")
    );

    const state = reactive<LocalState>({
      userList: [],
      errorList: [],
    });

    for (let i = 0; i < 3; i++) {
      state.userList.push({
        email: "",
        role: "DEVELOPER",
      });
      state.errorList.push("");
    }

    const validateUserInternal = (user: User): string => {
      if (user.email) {
        if (!isValidEmail(user.email)) {
          return "Invalid email address";
        } else {
          const member = store.getters["member/memberByEmail"](user.email);
          if (member.id != UNKNOWN_ID) {
            return "Already a member";
          }
        }
      }
      return "";
    };

    const validateUser = (user: User, index: number): boolean => {
      state.errorList[index] = validateUserInternal(user);
      return state.errorList[index].length == 0;
    };

    const clearValidationError = (index: number) => {
      state.errorList[index] = "";
    };

    const addUser = () => {
      state.userList.push({
        email: "",
        role: "DEVELOPER",
      });
      state.errorList.push("");
    };

    const hasValidUserOnly = () => {
      let hasEmailInput = false;
      let hasError = false;
      state.userList.forEach((user) => {
        if (user.email) {
          hasEmailInput = true;
          if (validateUserInternal(user).length > 0) {
            hasError = true;
            return;
          }
        }
      });
      return !hasError && hasEmailInput;
    };

    const addOrInvite = () => {
      for (const user of state.userList) {
        if (isValidEmail(user.email)) {
          // Needs to assign to a local variable as user will change after createPrincipal but before createdMember
          let role = user.role;
          // If admin feature is NOT enabled, then we set every user to OWNER role.
          if (!hasRBACFeature.value) {
            role = "OWNER";
          }
          // Note "principal/createPrincipal" would return the existing principal.
          // This could happen if another client has just created the principal
          // with this email.
          const newPrincipal: PrincipalCreate = {
            name: user.email.split("@")[0],
            email: user.email,
          };
          store
            .dispatch("principal/createPrincipal", newPrincipal)
            .then((principal: Principal) => {
              const newMember: MemberCreate = {
                principalId: principal.id,
                status: isAdd.value ? "ACTIVE" : "INVITED",
                role,
              };
              // Note "principal/createdMember" would return the existing role mapping.
              // This could happen if another client has just created the role mapping with
              // this principal.
              store.dispatch("member/createdMember", newMember);
            });

          useUIStateStore().saveIntroStateByKey({
            key: "member.addOrInvite",
            newState: true,
          });
        }
      }
      state.userList.forEach((item) => {
        item.email = "";
        item.role = "DEVELOPER";
      });
      state.errorList = [""];
    };

    return {
      state,
      isAdd,
      hasRBACFeature,
      validateUser,
      clearValidationError,
      addUser,
      hasValidUserOnly,
      addOrInvite,
    };
  },
});
</script>
