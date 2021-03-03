<template>
  <div>
    <div class="relative">
      <button
        type="button"
        aria-haspopup="listbox"
        aria-expanded="true"
        aria-labelledby="listbox-label"
        class="btn-select relative w-full pl-3 pr-10 py-2"
        @click.prevent="state.showMenu = !state.showMenu"
      >
        <template v-if="state.selectedPrincipal">
          <span class="flex items-center">
            <BBAvatar
              :size="'small'"
              :username="state.selectedPrincipal.name"
            />
            <span class="ml-3 block truncate">
              {{ state.selectedPrincipal.name }}
            </span>
          </span>
        </template>
        <div v-else>Unassigned</div>
        <span
          class="ml-3 absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none"
        >
          <!-- Heroicon name: solid/selector -->
          <svg
            class="h-5 w-5 text-control-light"
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 20 20"
            fill="currentColor"
            aria-hidden="true"
          >
            <path
              fill-rule="evenodd"
              d="M10 3a1 1 0 01.707.293l3 3a1 1 0 01-1.414 1.414L10 5.414 7.707 7.707a1 1 0 01-1.414-1.414l3-3A1 1 0 0110 3zm-3.707 9.293a1 1 0 011.414 0L10 14.586l2.293-2.293a1 1 0 011.414 1.414l-3 3a1 1 0 01-1.414 0l-3-3a1 1 0 010-1.414z"
              clip-rule="evenodd"
            />
          </svg>
        </span>
      </button>

      <!--
      Select popover, show/hide based on select state.

      Entering: ""
        From: ""
        To: ""
      Leaving: "transition ease-in duration-100"
        From: "opacity-100"
        To: "opacity-0"
    -->
      <transition
        enter-active-class=""
        enter-class=""
        enter-to-class=""
        leave-active-class="transition ease-in duration-100"
        leave-class="opacity-100"
        leave-to-class="opacity-0"
      >
        <div
          v-show="state.showMenu"
          class="absolute mt-1 w-full rounded-md bg-white shadow-lg"
        >
          <ul
            tabindex="-1"
            role="listbox"
            aria-labelledby="listbox-label"
            aria-activedescendant="listbox-item-3"
            class="max-h-56 rounded-md py-1 ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none sm:text-sm"
          >
            <!--
          Select option, manage highlight styles based on mouseenter/mouseleave and keyboard navigation.

          Highlighted: "text-white bg-indigo-600", Not Highlighted: "text-gray-900"
        -->
            <li
              v-for="(item, index) in state.principalList"
              :key="index"
              role="option"
              class="z-10 text-main hover:text-main-text hover:bg-main-hover cursor-default select-none relative py-2 pl-3 pr-9"
              @click.prevent="
                if (item !== state.selectedPrincipal) {
                  $emit('select-principal', item);
                  state.selectedPrincipal = item;
                }
                close();
              "
            >
              <span class="flex items-center">
                <BBAvatar :size="'small'" :username="item.name" />
                <span class="ml-3 block truncate">
                  {{ item.name }}
                </span>
              </span>
              <span
                v-if="item === state.selectedPrincipal"
                class="absolute inset-y-0 right-0 flex items-center pr-4"
              >
                <!-- Heroicon name: solid/check -->
                <svg
                  class="h-5 w-5"
                  xmlns="http://www.w3.org/2000/svg"
                  viewBox="0 0 20 20"
                  fill="currentColor"
                  aria-hidden="true"
                >
                  <path
                    fill-rule="evenodd"
                    d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                    clip-rule="evenodd"
                  />
                </svg>
              </span>
            </li>
          </ul>
        </div>
      </transition>
    </div>
  </div>
</template>

<script lang="ts">
import { watchEffect, reactive } from "vue";
import { useStore } from "vuex";
import { Member, UserDisplay } from "../types";

interface LocalState {
  showMenu: boolean;
  principalList: UserDisplay[];
  selectedPrincipal?: UserDisplay;
}

export default {
  name: "PrincipalSelect",
  emits: ["select-principal"],
  components: {},
  props: {
    selectedId: {
      type: String,
    },
  },
  setup(props, { emit }) {
    const state = reactive<LocalState>({
      showMenu: false,
      principalList: [],
    });
    const store = useStore();

    const preparePrincipalList = () => {
      store
        .dispatch("member/fetchMemberList")
        .then((list: Member[]) => {
          state.principalList = list.map((member: Member) => {
            return {
              id: member.attributes.user.id,
              name: member.attributes.user.name,
            };
          });

          state.selectedPrincipal = state.principalList.find(
            (userDisplay) => userDisplay.id == props.selectedId
          );
        })
        .catch((error) => {
          console.log(error);
        });
    };

    watchEffect(preparePrincipalList);

    const close = () => {
      state.showMenu = false;
    };

    return {
      state,
      close,
    };
  },
};
</script>
