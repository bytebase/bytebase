<template>
  <BBSelect
    :selectedItem="state.selectedPrincipal"
    :itemList="state.principalList"
    :placeholder="'Unassigned'"
    @select-item="(item) => $emit('select-principal', item)"
  >
    <template v-slot:menuItem="{ item }">
      <!--TODO: Have to set a fixed width, otherwise the width would change based on the selected text.
          Likely, there is a better solution, while the author doesn't want to fight with CSS for now.
          The specific value and breakpoint is to make it align with other select in the task sidebar.
          -->
      <span class="flex lg:40 xl:w-44 items-center space-x-2">
        <BBAvatar :size="'small'" :username="item.name" />
        <span class="truncate">{{ item.name }}</span>
      </span>
    </template>
  </BBSelect>
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
