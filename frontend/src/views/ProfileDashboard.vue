<template>
  <main
    class="flex-1 relative z-0 overflow-auto focus:outline-none xl:order-last"
    tabindex="0"
  >
    <article>
      <!-- Profile header -->
      <div>
        <div class="h-32 w-full bg-accent lg:h-48"></div>
        <div class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8">
          <div class="-mt-20 sm:flex sm:items-end sm:space-x-5">
            <PrincipalAvatar :principal="principal" :size="'HUGE'" />
            <div
              class="
                mt-6
                sm:flex-1
                sm:min-w-0
                sm:flex
                sm:items-center
                sm:justify-end
                sm:space-x-6
                sm:pb-1
              "
            >
              <div class="mt-6 flex flex-row justify-stretch space-x-4">
                <button v-if="false" type="button" class="btn-normal">
                  <!-- Heroicon name: solid/mail -->
                  <svg
                    class="-ml-1 mr-2 h-5 w-5 text-control-light"
                    xmlns="http://www.w3.org/2000/svg"
                    viewBox="0 0 20 20"
                    fill="currentColor"
                    aria-hidden="true"
                  >
                    <path
                      d="M2.003 5.884L10 9.882l7.997-3.998A2 2 0 0016 4H4a2 2 0 00-1.997 1.884z"
                    />
                    <path
                      d="M18 8.118l-8 4-8-4V14a2 2 0 002 2h12a2 2 0 002-2V8.118z"
                    />
                  </svg>
                  <span>Message</span>
                </button>
                <template v-if="allowEdit">
                  <template v-if="state.editing">
                    <button
                      type="button"
                      class="btn-normal"
                      @click.prevent="cancelEdit"
                    >
                      Cancel
                    </button>
                    <button
                      type="button"
                      class="btn-normal"
                      :disabled="!allowSaveEdit"
                      @click.prevent="saveEdit"
                    >
                      <!-- Heroicon name: solid/save -->
                      <svg
                        class="-ml-1 mr-2 h-5 w-5 text-control-light"
                        fill="currentColor"
                        viewBox="0 0 20 20"
                        xmlns="http://www.w3.org/2000/svg"
                      >
                        <path
                          d="M7.707 10.293a1 1 0 10-1.414 1.414l3 3a1 1 0 001.414 0l3-3a1 1 0 00-1.414-1.414L11 11.586V6h5a2 2 0 012 2v7a2 2 0 01-2 2H4a2 2 0 01-2-2V8a2 2 0 012-2h5v5.586l-1.293-1.293zM9 4a1 1 0 012 0v2H9V4z"
                        ></path>
                      </svg>
                      <span>Save</span>
                    </button>
                  </template>
                  <button
                    v-else
                    type="button"
                    class="btn-normal"
                    @click.prevent="editUser"
                  >
                    <!-- Heroicon name: solid/pencil -->
                    <svg
                      class="-ml-1 mr-2 h-5 w-5 text-control-light"
                      fill="currentColor"
                      viewBox="0 0 20 20"
                      xmlns="http://www.w3.org/2000/svg"
                    >
                      <path
                        d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z"
                      ></path>
                    </svg>
                    <span>Edit</span>
                  </button>
                </template>
              </div>
            </div>
          </div>
          <div class="block mt-6 min-w-0 flex-1">
            <input
              v-if="state.editing"
              required
              autocomplete="off"
              id="name"
              name="name"
              type="text"
              class="textfield"
              ref="editNameTextField"
              :value="state.editingPrincipal.name"
              @input="updatePrincipal('name', $event.target.value)"
            />
            <!-- pb-1.5 is to avoid flicking when entering/existing the editing state -->
            <h1 v-else class="pb-1.5 text-2xl font-bold text-main truncate">
              {{ principal.name }}
            </h1>
          </div>
        </div>
      </div>

      <!-- Description list -->
      <div
        v-if="principal.type == 'END_USER'"
        class="mt-6 mb-2 max-w-5xl mx-auto px-4 sm:px-6 lg:px-8"
      >
        <dl class="grid grid-cols-1 gap-x-4 gap-y-8 sm:grid-cols-2">
          <div class="sm:col-span-1">
            <dt class="text-sm font-medium text-control-light">Role</dt>
            <dd class="mt-1 text-sm text-main">
              <router-link
                :to="'/setting/member'"
                class="normal-link capitalize"
              >
                {{
                  principal.role == "DBA"
                    ? principal.role
                    : principal.role.toLowerCase()
                }}
              </router-link>
              <router-link :to="'/setting/plan'" class="normal-link">
                (Upgrade to Team plan to enable role management)
              </router-link>
            </dd>
          </div>

          <div class="sm:col-span-1">
            <dt class="text-sm font-medium text-control-light">Email</dt>
            <dd class="mt-1 text-sm text-main">
              {{ principal.email }}
            </dd>
          </div>

          <template v-if="state.editing">
            <div class="sm:col-span-1">
              <dt class="text-sm font-medium text-control-light">Password</dt>
              <dd class="mt-1 text-sm text-main">
                <input
                  id="password"
                  name="password"
                  type="text"
                  class="textfield mt-1 w-full"
                  autocomplete="off"
                  placeholder="sensitive - write only"
                  :value="state.editingPrincipal.password"
                  @input="updatePrincipal('password', $event.target.value)"
                />
              </dd>
            </div>

            <div class="sm:col-span-1">
              <dt class="text-sm font-medium text-control-light">
                Confirm
                <span v-if="passwordMismatch" class="text-error">mismatch</span>
              </dt>
              <dd class="mt-1 text-sm text-main">
                <input
                  id="password-confirm"
                  name="password-confirm"
                  type="text"
                  class="textfield mt-1 w-full"
                  autocomplete="off"
                  placeholder="Confirm new password"
                  :value="state.passwordConfirm"
                  @input="state.passwordConfirm = $event.target.value"
                />
              </dd>
            </div>
          </template>
        </dl>
      </div>
    </article>
  </main>
</template>

<script lang="ts">
import { nextTick, computed, onMounted, onUnmounted, reactive, ref } from "vue";
import { useStore } from "vuex";
import cloneDeep from "lodash-es/cloneDeep";
import isEmpty from "lodash-es/isEmpty";
import isEqual from "lodash-es/isEqual";
import PrincipalAvatar from "../components/PrincipalAvatar.vue";
import { Principal, PrincipalPatch } from "../types";
import { isOwner } from "../utils";

interface LocalState {
  editing: boolean;
  editingPrincipal?: PrincipalPatch;
  passwordConfirm?: string;
}

export default {
  name: "ProfileDashboard",
  props: {
    principalID: {
      type: String,
    },
  },
  components: { PrincipalAvatar },
  setup(props, ctx) {
    const editNameTextField = ref();

    const store = useStore();

    const state = reactive<LocalState>({
      editing: false,
    });

    const keyboardHandler = (e: KeyboardEvent) => {
      if (state.editing) {
        if (e.code == "Escape") {
          cancelEdit();
        } else if (e.code == "Enter" && e.metaKey) {
          if (allowSaveEdit.value) {
            saveEdit();
          }
        }
      }
    };

    onMounted(() => {
      document.addEventListener("keydown", keyboardHandler);
    });

    onUnmounted(() => {
      document.removeEventListener("keydown", keyboardHandler);
    });

    const currentUser = computed(
      (): Principal => store.getters["auth/currentUser"]()
    );

    const hasAdminFeature = computed(() =>
      store.getters["plan/feature"]("bb.admin")
    );

    const principal = computed(() => {
      if (props.principalID) {
        return store.getters["principal/principalByID"](
          parseInt(props.principalID)
        );
      }
      return currentUser.value;
    });

    const passwordMismatch = computed(() => {
      return (
        !isEmpty(state.editingPrincipal?.password) &&
        state.editingPrincipal?.password != state.passwordConfirm
      );
    });

    // User can change her own info.
    // Besides, owner can also change anyone's info. This is for resetting password in case user forgets.
    const allowEdit = computed(() => {
      return (
        currentUser.value.id == principal.value.id ||
        isOwner(currentUser.value.role)
      );
    });

    const allowSaveEdit = computed(() => {
      return (
        !isEqual(principal.value, state.editingPrincipal) &&
        (state.passwordConfirm == "" ||
          state.passwordConfirm == state.editingPrincipal?.password)
      );
    });

    const updatePrincipal = (field: string, value: string) => {
      (state.editingPrincipal as any)[field] = value;
    };

    const editUser = () => {
      const clone = cloneDeep(principal.value);
      state.editingPrincipal = {
        name: clone.name,
      };
      state.editing = true;

      nextTick(() => editNameTextField.value.focus());
    };

    const cancelEdit = () => {
      state.editingPrincipal = undefined;
      state.editing = false;
    };

    const saveEdit = () => {
      store
        .dispatch("principal/patchPrincipal", {
          principalID: principal.value.id,
          principalPatch: state.editingPrincipal,
        })
        .then(() => {
          state.editingPrincipal = undefined;
          state.editing = false;
        });
    };

    return {
      editNameTextField,
      state,
      hasAdminFeature,
      principal,
      allowEdit,
      allowSaveEdit,
      passwordMismatch,
      updatePrincipal,
      editUser,
      cancelEdit,
      saveEdit,
    };
  },
};
</script>
