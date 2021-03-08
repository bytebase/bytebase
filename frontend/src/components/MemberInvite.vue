<template>
  <div class="space-y-6">
    <div class="space-y-2">
      <div class="space-y-1">
        <p id="invite_members_helper" class="sr-only">Add by email address</p>
        <div
          class="flex flex-col space-y-2 sm:flex-row sm:space-x-4 sm:space-y-0"
        >
          <div class="flex-grow">
            <input
              type="email"
              name="invite_members"
              id="invite_members"
              class="textfield w-full lowercase"
              placeholder="foo@example.com"
              v-model="state.inviteEmail"
              ref="inviteEmailTextField"
              aria-describedby="invite_members_helper"
            />
          </div>
          <fieldset>
            <div class="radio-set">
              <div class="radio">
                <input
                  name="invite_role"
                  type="radio"
                  class="btn"
                  value="OWNER"
                  v-model="state.inviteRole"
                />
                <label class="label"> Owner </label>
              </div>
              <div class="radio">
                <input
                  name="invite_role"
                  type="radio"
                  class="btn"
                  value="DBA"
                  v-model="state.inviteRole"
                />
                <label class="label"> DBA </label>
              </div>
              <div class="radio">
                <input
                  name="invite_role"
                  type="radio"
                  class="btn"
                  value="DEVELOPER"
                  v-model="state.inviteRole"
                />
                <label class="label"> Developer </label>
              </div>
            </div>
          </fieldset>

          <span class="flex justify-end">
            <button
              type="button"
              class="btn-normal"
              :disabled="!isValidEmail(state.inviteEmail)"
              @click.prevent="addInvite"
            >
              <!-- Heroicon name: solid/plus -->
              <svg
                class="-ml-2 mr-1 h-5 w-5 text-gray-400"
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 20 20"
                fill="currentColor"
                aria-hidden="true"
              >
                <path
                  fill-rule="evenodd"
                  d="M10 5a1 1 0 011 1v3h3a1 1 0 110 2h-3v3a1 1 0 11-2 0v-3H6a1 1 0 110-2h3V6a1 1 0 011-1z"
                  clip-rule="evenodd"
                />
              </svg>
              <span>Add</span>
            </button>
          </span>
        </div>
      </div>

      <div class="border-b border-gray-200 py-1 select-none">
        <ul class="divide-y divide-gray-200">
          <li
            v-for="(invite, index) in state.inviteList"
            :key="index"
            class="py-4 flex"
          >
            <div
              class="w-full flex flex-col space-y-2 sm:flex-row sm:justify-between sm:space-y-0"
            >
              <div class="flex items-center justify-between">
                <div class="flex items-center space-x-2">
                  <BBAvatar :username="invite.email" />
                  <span class="text-sm text-gray-500">{{ invite.email }}</span>
                </div>

                <button
                  class="btn-icon"
                  type="button"
                  @click.prevent="deleteInvite(invite)"
                >
                  <svg
                    class="btn-icon sm:hidden w-4 h-4"
                    fill="currentColor"
                    viewBox="0 0 20 20"
                    xmlns="http://www.w3.org/2000/svg"
                  >
                    <path
                      fill-rule="evenodd"
                      d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                      clip-rule="evenodd"
                    ></path>
                  </svg>
                </button>
              </div>

              <div class="flex flex-row items-center space-x-2">
                <fieldset>
                  <div class="radio-set">
                    <div class="radio">
                      <input
                        type="radio"
                        class="btn"
                        value="OWNER"
                        v-model="invite.role"
                      />
                      <label class="label"> Owner </label>
                    </div>
                    <div class="radio">
                      <input
                        type="radio"
                        class="btn"
                        value="DBA"
                        v-model="invite.role"
                      />
                      <label class="label"> DBA </label>
                    </div>
                    <div class="radio">
                      <input
                        type="radio"
                        class="btn"
                        value="DEVELOPER"
                        v-model="invite.role"
                      />
                      <label class="label"> Developer </label>
                    </div>
                  </div>
                </fieldset>

                <button
                  class="btn-icon"
                  type="button"
                  @click.prevent="deleteInvite(invite)"
                >
                  <svg
                    class="btn-icon hidden sm:block w-4 h-4"
                    fill="currentColor"
                    viewBox="0 0 20 20"
                    xmlns="http://www.w3.org/2000/svg"
                  >
                    <path
                      fill-rule="evenodd"
                      d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                      clip-rule="evenodd"
                    ></path>
                  </svg>
                </button>
              </div>
            </div>
          </li>
        </ul>
      </div>
    </div>

    <div class="flex justify-end">
      <button
        type="button"
        class="btn-primary"
        :disabled="state.inviteList.length == 0"
        @click.prevent="sendInvite"
      >
        Invite
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, Ref } from "vue";
import { useStore } from "vuex";
import cloneDeep from "lodash-es/cloneDeep";
import { RoleType, RoleMappingNew } from "../types";
import { isValidEmail } from "../utils";

interface LocalState {
  inviteList: RoleMappingNew[];
  inviteEmail: Ref<string>;
  inviteRole: Ref<RoleType>;
}

export default {
  name: "MemberInvite",
  props: {},
  setup(props, ctx) {
    const store = useStore();
    const state = reactive<LocalState>({
      inviteList: [],
      inviteEmail: ref(""),
      inviteRole: ref("OWNER"),
    });
    const inviteEmailTextField = ref();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const keyboardHandler = (e: KeyboardEvent) => {
      if (e.code == "Enter") {
        if (inviteEmailTextField.value === document.activeElement) {
          addInvite();
        }
      }
    };

    onMounted(() => {
      document.addEventListener("keydown", keyboardHandler);
    });

    onUnmounted(() => {
      document.removeEventListener("keydown", keyboardHandler);
    });

    const addInvite = () => {
      if (isValidEmail(state.inviteEmail)) {
        state.inviteList.push(
          cloneDeep({
            email: state.inviteEmail,
            role: state.inviteRole,
            updaterId: currentUser.value.id,
          })
        );
        state.inviteEmail = "";
      }
    };

    const deleteInvite = (invite: RoleMappingNew) => {
      const i = state.inviteList.indexOf(invite);
      if (i > -1) {
        state.inviteList.splice(i, 1);
      }
    };

    const sendInvite = () => {
      console.log(state.inviteList);
      for (const invite of state.inviteList) {
        store.dispatch("roleMapping/createdRoleMapping", invite);
      }
      state.inviteList = [];
    };

    return {
      state,
      inviteEmailTextField,
      addInvite,
      deleteInvite,
      sendInvite,
      isValidEmail,
    };
  },
};
</script>
