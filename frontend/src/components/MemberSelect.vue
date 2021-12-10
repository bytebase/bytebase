<template>
  <BBSelect
    :selected-item="selectedPrincipal"
    :item-list="principalList"
    :placeholder="placeholder"
    :disabled="disabled"
    @select-item="
      (principal) => {
        state.selectedId = principal.id;
        $emit('select-principal-id', parseInt(principal.id));
      }
    "
  >
    <template #menuItem="{ item }">
      <!--TODO(tianzhou): Have to set a fixed width, otherwise the width would change based on the selected text.
          Likely, there is a better solution, while the author doesn't want to fight with CSS for now.
          The specific value and breakpoint is to make it align with other select in the issue sidebar.
          -->
      <span class="flex lg:40 xl:w-44 items-center space-x-2">
        <PrincipalAvatar :principal="item" :size="'SMALL'" />
        <span class="truncate">{{ item.name }}</span>
      </span>
    </template>
    <template #placeholder="{ placeholder }">
      <!--TODO(tianzhou): Have to set a fixed width, otherwise the width would change based on the selected text.
          Likely, there is a better solution, while the author doesn't want to fight with CSS for now.
          The specific value and breakpoint is to make it align with other select in the issue sidebar.
          -->
      <!-- Add my-0.5 padding to avoid flickering when switching to assignee -->
      <span class="flex my-0.5 lg:40 xl:w-44">
        <span class="truncate" :class="required ? 'text-error' : ''">{{
          $t(placeholder)
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
  PrincipalId,
  RoleType,
  SYSTEM_BOT_ID,
} from "../types";
import { isDBA, isDeveloper, isOwner } from "../utils";

interface LocalState {
  selectedId?: PrincipalId;
  showMenu: boolean;
}

export default {
  name: "MemberSelect",
  components: { PrincipalAvatar },
  props: {
    selectedId: {
      type: Number,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
    allowedRoleList: {
      default: () => ["OWNER", "DBA", "DEVELOPER"],
      type: Array as PropType<RoleType[]>,
    },
    placeholder: {
      default: "settings.members.not-assigned",
      type: String,
    },
    required: {
      default: true,
      type: Boolean,
    },
  },
  emits: ["select-principal-id"],
  setup(props) {
    const state = reactive<LocalState>({
      selectedId: props.selectedId,
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
      if (props.selectedId == SYSTEM_BOT_ID) {
        list.unshift(store.getters["principal/principalById"](SYSTEM_BOT_ID));
      }
      return list.filter((item: Principal) => {
        // The previously selected item might no longer be applicable.
        // e.g. The select limits to DBA only and the selected principal is no longer a DBA
        // in such case, we still show the item.
        if (item.id == props.selectedId) {
          return true;
        }

        return (
          // We write this way instead of props.allowedRoleList.includes(item.role)
          // is because isOwner/isDBA/isDeveloper has feature gate logic.
          (props.allowedRoleList.includes("OWNER") && isOwner(item.role)) ||
          (props.allowedRoleList.includes("DBA") && isDBA(item.role)) ||
          (props.allowedRoleList.includes("DEVELOPER") &&
            isDeveloper(item.role))
        );
      });
    });

    watch(
      () => props.selectedId,
      (cur) => {
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
