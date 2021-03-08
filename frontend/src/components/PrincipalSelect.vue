<template>
  <BBSelect
    :selectedItem="selectedPrincipal"
    :itemList="principalList"
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
import { watchEffect, reactive, computed } from "vue";
import { useStore } from "vuex";
import { RoleMapping, Principal } from "../types";

interface LocalState {
  showMenu: boolean;
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
    });
    const store = useStore();

    const principalList = computed(() =>
      store.getters["roleMapping/roleMappingList"]().map(
        (roleMapping: RoleMapping) => {
          return roleMapping.principal;
        }
      )
    );

    const selectedPrincipal = computed(() =>
      principalList.value.find(
        (principal: Principal) => principal.id == props.selectedId
      )
    );

    const close = () => {
      state.showMenu = false;
    };

    return {
      state,
      principalList,
      selectedPrincipal,
      close,
    };
  },
};
</script>
