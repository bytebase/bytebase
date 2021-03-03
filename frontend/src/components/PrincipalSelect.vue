<template>
  <BBSelect
    :selectedItem="state.selectedPrincipal"
    :itemList="state.principalList"
    :placeholder="'Unassigned'"
    @select-item="(item) => $emit('select-principal', item)"
  >
    <template v-slot:menuItem="{ item }">
      <span class="flex items-center">
        <BBAvatar :size="'small'" :username="item.name" />
        <span class="ml-3 block truncate">
          {{ item.name }}
        </span>
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
