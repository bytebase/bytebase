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
              class="mt-6 sm:flex-1 sm:min-w-0 sm:flex sm:items-center sm:justify-end sm:space-x-6 sm:pb-1"
            >
              <div class="mt-6 flex flex-row justify-stretch space-x-4">
                <button v-if="false" type="button" class="btn-normal">
                  <!-- Heroicon name: solid/mail -->
                  <heroicons-solid:mail
                    class="-ml-1 mr-2 h-5 w-5 text-control-light"
                  />
                  <span>Message</span>
                </button>
                <template v-if="allowEdit">
                  <template v-if="state.editing">
                    <button
                      type="button"
                      class="btn-normal"
                      @click.prevent="cancelEdit"
                    >
                      {{ $t("common.cancel") }}
                    </button>
                    <button
                      type="button"
                      class="btn-normal"
                      :disabled="!allowSaveEdit"
                      @click.prevent="saveEdit"
                    >
                      <!-- Heroicon name: solid/save -->
                      <heroicons-solid:save
                        class="-ml-1 mr-2 h-5 w-5 text-control-light"
                      />
                      <span>{{ $t("common.save") }}</span>
                    </button>
                  </template>
                  <button
                    v-else
                    type="button"
                    class="btn-normal"
                    @click.prevent="editUser"
                  >
                    <!-- Heroicon name: solid/pencil -->
                    <heroicons-solid:pencil
                      class="-ml-1 mr-2 h-5 w-5 text-control-light"
                    />
                    <span>{{ $t("common.edit") }}</span>
                  </button>
                </template>
              </div>
            </div>
          </div>
          <div class="block mt-6 min-w-0 flex-1">
            <input
              v-if="state.editing"
              id="name"
              ref="editNameTextField"
              required
              autocomplete="off"
              name="name"
              type="text"
              class="textfield"
              :value="state.editingPrincipal?.name"
              @input="(e: any)=>updatePrincipal('name', e.target.value)"
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
            <dt class="text-sm font-medium text-control-light">
              {{ $t("settings.profile.role") }}
            </dt>
            <dd class="mt-1 text-sm text-main">
              <router-link
                :to="'/setting/member'"
                class="normal-link capitalize"
                >{{
                  $t(`common.role.${principal.role.toLowerCase()}`)
                }}</router-link
              >
              <router-link
                v-if="!hasRBACFeature"
                :to="'/setting/subscription'"
                class="normal-link"
              >
                {{ $t("settings.profile.subscription") }}
              </router-link>
            </dd>
          </div>

          <div class="sm:col-span-1">
            <dt class="text-sm font-medium text-control-light">
              {{ $t("settings.profile.email") }}
            </dt>
            <dd class="mt-1 text-sm text-main">{{ principal.email }}</dd>
          </div>

          <template v-if="state.editing">
            <div class="sm:col-span-1">
              <dt class="text-sm font-medium text-control-light">
                {{ $t("settings.profile.password") }}
              </dt>
              <dd class="mt-1 text-sm text-main">
                <input
                  id="password"
                  name="password"
                  type="text"
                  class="textfield mt-1 w-full"
                  autocomplete="off"
                  :placeholder="$t('common.sensitive-placeholder')"
                  :value="state.editingPrincipal?.password"
                  @input="(e: any) => updatePrincipal('password', e.target.value)"
                />
              </dd>
            </div>

            <div class="sm:col-span-1">
              <dt class="text-sm font-medium text-control-light">
                {{ $t("settings.profile.password-confirm") }}
                <span v-if="passwordMismatch" class="text-error">{{
                  $t("settings.profile.password-mismatch")
                }}</span>
              </dt>
              <dd class="mt-1 text-sm text-main">
                <input
                  id="password-confirm"
                  name="password-confirm"
                  type="text"
                  class="textfield mt-1 w-full"
                  autocomplete="off"
                  :placeholder="
                    $t('settings.profile.password-confirm-placeholder')
                  "
                  :value="state.passwordConfirm"
                  @input="(e: any) => state.passwordConfirm = e.target.value"
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
import {
  nextTick,
  computed,
  onMounted,
  onUnmounted,
  reactive,
  ref,
  defineComponent,
} from "vue";
import cloneDeep from "lodash-es/cloneDeep";
import isEmpty from "lodash-es/isEmpty";
import isEqual from "lodash-es/isEqual";
import PrincipalAvatar from "../components/PrincipalAvatar.vue";
import { PrincipalPatch } from "../types";
import { isOwner } from "../utils";
import { featureToRef, useCurrentUser, usePrincipalStore } from "@/store";

interface LocalState {
  editing: boolean;
  editingPrincipal?: PrincipalPatch;
  passwordConfirm?: string;
}

export default defineComponent({
  name: "ProfileDashboard",
  components: { PrincipalAvatar },
  props: {
    principalId: {
      type: String,
      default: undefined,
    },
  },
  setup(props) {
    const editNameTextField = ref();

    const principalStore = usePrincipalStore();

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

    const currentUser = useCurrentUser();

    const hasRBACFeature = featureToRef("bb.feature.rbac");

    const principal = computed(() => {
      if (props.principalId) {
        return principalStore.principalById(parseInt(props.principalId));
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
      principalStore
        .patchPrincipal({
          principalId: principal.value.id,
          principalPatch: state.editingPrincipal!,
        })
        .then(() => {
          state.editingPrincipal = undefined;
          state.editing = false;
        });
    };

    return {
      editNameTextField,
      state,
      hasRBACFeature,
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
});
</script>
