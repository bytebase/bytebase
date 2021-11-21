<template>
  <BBSelect
    :selectedItem="selectedPrincipal"
    :itemList="principalList"
    :placeholder="placeholder"
    :disabled="disabled"
    @select-item="
      (principal) => {
        state.selectedID = principal.id;
        $emit('select-principal-id', parseInt(principal.id));
      }
    "
  >
    <template v-slot:menuItem="{ item }">
      <!--TODO(tianzhou): Have to set a fixed width, otherwise the width would change based on the selected text.
          Likely, there is a better solution, while the author doesn't want to fight with CSS for now.
          The specific value and breakpoint is to make it align with other select in the issue sidebar.
          -->
      <span class="flex lg:40 xl:w-44 items-center space-x-2">
        <PrincipalAvatar :principal="item" :size="'SMALL'" />
        <span class="truncate">{{ item.name }}</span>
      </span>
    </template>
    <template v-slot:placeholder="{ placeholder }">
      <!--TODO(tianzhou): Have to set a fixed width, otherwise the width would change based on the selected text.
          Likely, there is a better solution, while the author doesn't want to fight with CSS for now.
          The specific value and breakpoint is to make it align with other select in the issue sidebar.
          -->
      <!-- Add my-0.5 padding to avoid flickering when switching to assignee -->
      <span class="flex my-0.5 lg:40 xl:w-44">
        <span class="truncate" :class="required ? 'text-error' : ''">{{
          placeholder
        }}</span>
      </span>
    </template>
  </BBSelect>
</template>

<script lang="ts">
import { reactive, computed, watch, PropType } from "vue";
import { useStore } from "vuex";
import PrincipalAvatar from "./PrincipalAvatar.vue";
import {
  Member,
  Principal,
  PrincipalID,
  RoleType,
  SYSTEM_BOT_ID,
} from "../types";
import { isDBA, isDeveloper, isOwner } from "../utils";

interface LocalState {
  selectedID?: PrincipalID;
  showMenu: boolean;
}

export default {
  name: "MemberSelect",
  emits: ["select-principal-id"],
  components: { PrincipalAvatar },
  props: {
    selectedID: {
      type: Number,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
    allowedRoleList: {
      default: ["OWNER", "DBA", "DEVELOPER"],
      type: Object as PropType<RoleType[]>,
    },
    placeholder: {
      default: "Not assigned",
      type: String,
    },
    required: {
      default: true,
      type: Boolean,
    },
  },
  setup(props, { emit }) {
    const state = reactive<LocalState>({
      selectedID: props.selectedID,
      showMenu: false,
    });
    const store = useStore();

    const principalList = computed((): Principal[] => {
      const list = store.getters["member/memberList"]()
        .filter((member: Member) => {
          return member.status == "ACTIVE";
        })
        .map((member: Member) => {
          return member.principal;
        });
      // If system bot is the selected ID (e.g. when issue is created by the bot on observing new sql file),
      // Then we add system bot to the list so it can display properly.
      if (props.selectedID == SYSTEM_BOT_ID) {
        list.unshift(store.getters["principal/principalByID"](SYSTEM_BOT_ID));
      }
      return list.filter((item: Principal) => {
        // The previouly selected item might no longer be applicable.
        // e.g. The select limits to DBA only and the selected principal is no longer a DBA
        // in such case, we still show the item.
        if (item.id == props.selectedID) {
          return true;
        }

        return (
          // We write this way instead of props.allowedRoleList.includes(item.role)
          // is becaues isOwner/isDBA/isDeveloper has feature gate logic.
          (props.allowedRoleList.includes("OWNER") && isOwner(item.role)) ||
          (props.allowedRoleList.includes("DBA") && isDBA(item.role)) ||
          (props.allowedRoleList.includes("DEVELOPER") &&
            isDeveloper(item.role))
        );
      });
    });

    watch(
      () => props.selectedID,
      (cur, _) => {
        state.selectedID = cur;
      }
    );

    const selectedPrincipal = computed(() =>
      principalList.value.find(
        (principal: Principal) => principal.id == state.selectedID
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
