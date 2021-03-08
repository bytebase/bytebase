<template>
  <div class="w-full space-y-4">
    <div class="space-y-2">
      <div
        v-for="(invite, index) in state.inviteList"
        :key="index"
        class="py-0.5 select-none"
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
              class="textfield w-full lowercase"
              placeholder="foo@example.com"
              v-model="invite.email"
              aria-describedby="invite_members_helper"
            />
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
        :disabled="!hasValidInvite()"
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
import { RoleMappingNew } from "../types";
import { isValidEmail } from "../utils";

interface LocalState {
  inviteList: RoleMappingNew[];
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
    });

    for (let i = 0; i < 3; i++) {
      state.inviteList.push({
        email: "",
        role: "DEVELOPER",
        updaterId: currentUser.value.id,
      });
    }

    const addInvite = () => {
      state.inviteList.push({
        email: "",
        role: "DEVELOPER",
        updaterId: currentUser.value.id,
      });
    };

    const hasValidInvite = () => {
      let hasValidEmail = false;
      for (const invite of state.inviteList) {
        if (invite.email) {
          if (isValidEmail(invite.email)) {
            hasValidEmail = true;
          } else {
            return false;
          }
        }
      }
      return hasValidEmail;
    };

    const sendInvite = () => {
      for (const invite of state.inviteList) {
        if (isValidEmail(invite.email)) {
          store.dispatch("roleMapping/createdRoleMapping", invite);
        }
      }
      state.inviteList = [];
      state.inviteList.push({
        email: "",
        role: "DEVELOPER",
        updaterId: currentUser.value.id,
      });
    };

    return {
      state,
      addInvite,
      hasValidInvite,
      sendInvite,
    };
  },
};
</script>
