<template>
  <div class="space-y-4">
    <div class="space-y-2">
      <div
        v-for="(invite, index) in state.inviteList"
        :key="index"
        class="flex justify-between py-0.5 select-none"
      >
        <p id="invite_members_helper" class="sr-only">
          Invite by email address
        </p>
        <div class="flex flex-row space-x-4">
          <div class="flex-grow">
            <input
              type="email"
              name="invite_members"
              autocomplete="off"
              class="w-36 sm:w-64 textfield lowercase"
              placeholder="foo@example.com"
              v-model="invite.email"
              @blur="validateInvite(invite, index)"
              @input="clearValidationError(index)"
              aria-describedby="invite_members_helper"
            />
            <p
              v-if="state.errorList[index]"
              class="mt-2 text-sm text-error"
              id="email-error"
            >
              {{ state.errorList[index] }}
            </p>
          </div>
          <div class="sm:hidden w-36">
            <RoleSelect
              :selectedRole="invite.role"
              @change-role="
                (role) => {
                  invite.role = role;
                }
              "
            />
          </div>
          <div class="hidden sm:flex sm:flex-row radio-set">
            <div class="radio">
              <input
                :name="`invite_role${index}`"
                tabindex="-1"
                type="radio"
                class="btn"
                value="OWNER"
                v-model="invite.role"
              />
              <label class="label"> Owner </label>
            </div>
            <div class="radio">
              <input
                :name="`invite_role${index}`"
                tabindex="-1"
                type="radio"
                class="btn"
                value="DBA"
                v-model="invite.role"
              />
              <label class="label"> DBA </label>
            </div>
            <div class="radio">
              <input
                :name="`invite_role${index}`"
                tabindex="-1"
                type="radio"
                class="btn"
                value="DEVELOPER"
                v-model="invite.role"
              />
              <label class="label"> Developer </label>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="flex justify-between">
      <span class="flex items-center">
        <button type="button" class="btn-secondary" @click.prevent="addInvite">
          + Add More
        </button>
      </span>

      <button
        type="button"
        class="btn-primary"
        :disabled="!hasValidInviteOnly()"
        @click.prevent="sendInvite"
      >
        <svg
          class="mr-2 h-5 w-5"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
          ></path>
        </svg>
        Send Invites
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, reactive } from "vue";
import { useStore } from "vuex";
import RoleSelect from "../components/RoleSelect.vue";
import { UNKNOWN_ID, Principal, RoleMappingNew } from "../types";
import { isValidEmail } from "../utils";

interface LocalState {
  inviteList: RoleMappingNew[];
  errorList: string[];
}

export default {
  name: "MemberInvite",
  components: { RoleSelect },
  props: {},
  setup(props, ctx) {
    const store = useStore();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const state = reactive<LocalState>({
      inviteList: [],
      errorList: [],
    });

    for (let i = 0; i < 3; i++) {
      state.inviteList.push({
        principalId: UNKNOWN_ID,
        email: "",
        role: "DEVELOPER",
        updaterId: currentUser.value.id,
      });
      state.errorList.push("");
    }

    const validateInvite = (invite: RoleMappingNew, index: number): boolean => {
      state.errorList[index] = "";
      if (invite.email) {
        if (!isValidEmail(invite.email)) {
          state.errorList[index] = "Invalid email address";
          return false;
        } else if (
          store.getters["roleMapping/roleMappingByEmail"](invite.email)
        ) {
          state.errorList[index] = "Already a member";
          return false;
        }
      }
      return true;
    };

    const clearValidationError = (index: number) => {
      state.errorList[index] = "";
    };

    const addInvite = () => {
      state.inviteList.push({
        principalId: UNKNOWN_ID,
        email: "",
        role: "DEVELOPER",
        updaterId: currentUser.value.id,
      });
      state.errorList.push("");
    };

    const hasValidInviteOnly = () => {
      let hasEmailInput = false;
      state.inviteList.forEach((invite) => {
        if (invite.email) {
          hasEmailInput = true;
        }
      });

      for (const error of state.errorList) {
        if (error) {
          return false;
        }
      }
      return hasEmailInput;
    };

    const sendInvite = () => {
      for (const invite of state.inviteList) {
        if (isValidEmail(invite.email)) {
          // We created a new principal for that email if not exists.
          // Note "principal/createPrincipal" would return the existing principal.
          // This could happen if another client has just created the principal
          // with this email.
          if (invite.principalId == UNKNOWN_ID) {
            invite.principalId = store.getters["principal/principalByEmail"](
              invite.email
            ).id;
          }
          if (invite.principalId != UNKNOWN_ID) {
            store.dispatch("roleMapping/createdRoleMapping", invite);
          } else {
            store
              .dispatch("principal/createPrincipal", {
                email: invite.email,
              })
              .then((principal: Principal) => {
                invite.principalId = principal.id;
                store.dispatch("roleMapping/createdRoleMapping", invite);
              })
              .catch((error) => {
                console.error(error);
              });
          }

          store.dispatch("uistate/saveIntroStateByKey", {
            key: "member.invite",
            newState: true,
          });
        }
      }
      state.inviteList = [
        {
          principalId: UNKNOWN_ID,
          email: "",
          role: "DEVELOPER",
          updaterId: currentUser.value.id,
        },
      ];
      state.errorList = [""];
    };

    return {
      state,
      validateInvite,
      clearValidationError,
      addInvite,
      hasValidInviteOnly,
      sendInvite,
    };
  },
};
</script>
