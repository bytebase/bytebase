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
          <div v-if="hasAdminFeature" class="sm:hidden w-36">
            <RoleSelect
              :selectedRole="invite.role"
              @change-role="
                (role) => {
                  invite.role = role;
                }
              "
            />
          </div>
          <div
            v-if="hasAdminFeature"
            class="hidden sm:flex sm:flex-row space-x-4"
            :class="state.errorList[index] ? '-mt-7' : ''"
          >
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
import { computed, ComputedRef, reactive } from "vue";
import { useStore } from "vuex";
import RoleSelect from "../components/RoleSelect.vue";
import { Principal, PrincipalCreate, MemberCreate, RoleType } from "../types";
import { isValidEmail } from "../utils";

type Invite = {
  email: string;
  role: RoleType;
};

interface LocalState {
  inviteList: Invite[];
  errorList: string[];
}

export default {
  name: "MemberInvite",
  components: { RoleSelect },
  props: {},
  setup(props, ctx) {
    const store = useStore();

    const currentUser: ComputedRef<Principal> = computed(() =>
      store.getters["auth/currentUser"]()
    );

    const hasAdminFeature = computed(() =>
      store.getters["plan/feature"]("bb.admin")
    );

    const state = reactive<LocalState>({
      inviteList: [],
      errorList: [],
    });

    for (let i = 0; i < 3; i++) {
      state.inviteList.push({
        email: "",
        role: "DEVELOPER",
      });
      state.errorList.push("");
    }

    const validateInviteInternal = (invite: Invite): string => {
      if (invite.email) {
        if (!isValidEmail(invite.email)) {
          return "Invalid email address";
        } else {
          const principal = store.getters["principal/principalByEmail"](
            invite.email
          );
          if (principal) {
            return "Already a member";
          }
        }
      }
      return "";
    };

    const validateInvite = (invite: Invite, index: number): boolean => {
      state.errorList[index] = validateInviteInternal(invite);
      return state.errorList[index].length == 0;
    };

    const clearValidationError = (index: number) => {
      state.errorList[index] = "";
    };

    const addInvite = () => {
      state.inviteList.push({
        email: "",
        role: "DEVELOPER",
      });
      state.errorList.push("");
    };

    const hasValidInviteOnly = () => {
      let hasEmailInput = false;
      let hasError = false;
      state.inviteList.forEach((invite) => {
        if (invite.email) {
          hasEmailInput = true;
          if (validateInviteInternal(invite).length > 0) {
            hasError = true;
            return;
          }
        }
      });
      return !hasError && hasEmailInput;
    };

    const sendInvite = () => {
      for (const invite of state.inviteList) {
        if (isValidEmail(invite.email)) {
          // If admin feature is NOT enabled, then we set every intite to OWNER role.
          if (!hasAdminFeature.value) {
            invite.role = "OWNER";
          }
          // Note "principal/createPrincipal" would return the existing principal.
          // This could happen if another client has just created the principal
          // with this email.
          const newPrincipal: PrincipalCreate = {
            creatorId: currentUser.value.id,
            name: invite.email.split("@")[0],
            email: invite.email,
          };
          store
            .dispatch("principal/createPrincipal", newPrincipal)
            .then((principal: Principal) => {
              const newMember: MemberCreate = {
                creatorId: currentUser.value.id,
                principalId: principal.id,
                role: invite.role,
              };
              // Note "principal/createdMember" would return the existing role mapping.
              // This could happen if another client has just created the role mapping with
              // this principal.
              store.dispatch("member/createdMember", newMember);
            })
            .catch((error) => {
              console.error(error);
            });

          store.dispatch("uistate/saveIntroStateByKey", {
            key: "member.invite",
            newState: true,
          });
        }
      }
      state.inviteList.forEach((item) => {
        item.email = "";
        item.role = "DEVELOPER";
      });
      state.errorList = [""];
    };

    return {
      state,
      hasAdminFeature,
      validateInvite,
      clearValidationError,
      addInvite,
      hasValidInviteOnly,
      sendInvite,
    };
  },
};
</script>
