<template>
  <BBSelect
    :selectedItem="selectedPrincipal"
    :itemList="principalList"
    :placeholder="'Unassigned'"
    :disabled="disabled"
    @select-item="
      (principal) => {
        state.selectedId = principal.id;
        $emit('select-principal-id', principal.id);
      }
    "
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
    <template v-slot:placeholder="{ placeholder }">
      <!--TODO: Have to set a fixed width, otherwise the width would change based on the selected text.
          Likely, there is a better solution, while the author doesn't want to fight with CSS for now.
          The specific value and breakpoint is to make it align with other select in the task sidebar.
          -->
      <span class="flex lg:40 xl:w-44">
        <span class="truncate">{{ placeholder }}</span>
      </span>
    </template>
  </BBSelect>
</template>

<script lang="ts">
import { reactive, computed, watch } from "vue";
import { useStore } from "vuex";
import { RoleMapping, Principal, PrincipalId } from "../types";
import { feature } from "../utils";

interface LocalState {
  selectedId?: PrincipalId;
  showMenu: boolean;
}

export default {
  name: "PrincipalSelect",
  emits: ["select-principal-id"],
  props: {
    selectedId: {
      type: String,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
    allowAllRoles: {
      default: true,
      type: Boolean,
    },
  },
  setup(props, { emit }) {
    const state = reactive<LocalState>({
      selectedId: props.selectedId,
      showMenu: false,
    });
    const store = useStore();

    const principalList = computed(() => {
      const list: RoleMapping[] = store.getters[
        "roleMapping/roleMappingList"
      ]().filter((item: RoleMapping) => {
        return (
          props.allowAllRoles ||
          !feature("bytebase.admin") ||
          item.role == "DBA" ||
          item.role == "OWNER"
        );
      });

      return list.map((roleMapping: RoleMapping) => {
        return store.getters["principal/principalById"](
          roleMapping.principalId
        );
      });
    });

    watch(
      () => props.selectedId,
      (cur, _) => {
        state.selectedId = cur;
      }
    );

    const selectedPrincipal = computed(() =>
      principalList.value.find(
        (principal: Principal) => principal.id == state.selectedId
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
