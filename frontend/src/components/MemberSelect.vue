<template>
  <BBComboBox
    :value="selectedPrincipal"
    :options="userList"
    :filter="filter"
    :placeholder="placeholder"
    :disabled="disabled"
    v-bind="$attrs"
    @update:value="
      (user: User) => {
        state.selectedId = extractUserUID(user.name);
        $emit('select-user-id', state.selectedId);
      }
    "
  >
    <template #menuItem="{ item }: { item: User }">
      <!--TODO(tianzhou): Have to set a fixed width, otherwise the width would change based on the selected text.
          Likely, there is a better solution, while the author doesn't want to fight with CSS for now.
          The specific value and breakpoint is to make it align with other select in the issue sidebar.
          -->
      <span class="flex lg:40 xl:w-44 items-center space-x-2">
        <!-- Show a special avatar for "All" -->
        <div
          v-if="showAll && extractUserUID(item.name) === String(UNKNOWN_ID)"
          class="w-6 h-6 rounded-full border-2 border-current flex justify-center items-center select-none bg-white"
        >
          <heroicons-outline:user class="w-4 h-4 text-main" />
        </div>
        <!-- Show the initial letters by default -->
        <UserAvatar v-else :user="item" size="SMALL" />
        <span class="truncate">{{ item.title }}</span>
      </span>
    </template>
    <template #placeholder>
      <span class="leading-6 truncate" :class="required ? 'text-error' : ''">{{
        $t(placeholder)
      }}</span>
    </template>
  </BBComboBox>
</template>

<script lang="ts" setup>
import { reactive, computed, watch, PropType } from "vue";
import { useI18n } from "vue-i18n";
import { hasFeature, useUserStore } from "@/store";
import { User, UserRole, UserType } from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";
import { BBComboBox } from "../bbkit";
import {
  SYSTEM_BOT_ID,
  UNKNOWN_ID,
  SYSTEM_BOT_USER_NAME,
  unknownUser,
  filterUserListByKeyword,
} from "../types";
import { extractUserUID } from "../utils";
import UserAvatar from "./User/UserAvatar.vue";

interface LocalState {
  selectedId: string | undefined;
  showMenu: boolean;
}

const props = defineProps({
  selectedId: {
    type: String,
    default: undefined,
  },
  disabled: {
    default: false,
    type: Boolean,
  },
  showAll: {
    type: Boolean,
    default: false,
  },
  showSystemBot: {
    type: Boolean,
    default: false,
  },
  allowedRoleList: {
    default: () => [UserRole.OWNER, UserRole.DBA, UserRole.DEVELOPER],
    type: Array as PropType<UserRole[]>,
  },
  placeholder: {
    default: "settings.members.not-assigned",
    type: String,
  },
  required: {
    default: true,
    type: Boolean,
  },
  customFilter: {
    type: Function as PropType<(user: User) => boolean>,
    default: undefined,
  },
});

defineEmits<{
  (event: "select-user-id", uid: string): void;
}>();

const state = reactive<LocalState>({
  selectedId: props.selectedId,
  showMenu: false,
});
const { t } = useI18n();
const userStore = useUserStore();

const userList = computed((): User[] => {
  const list = userStore.userList.filter((user) => {
    return user.state === State.ACTIVE && user.userType === UserType.USER;
  });
  // If system bot is the selected ID (e.g. when issue is created by the bot on observing new sql file),
  // Then we add system bot to the list so it can display properly.
  if (props.selectedId === String(SYSTEM_BOT_ID) || props.showSystemBot) {
    const systemBotIndex = list.findIndex(
      (user) => user.name === SYSTEM_BOT_USER_NAME
    );
    if (systemBotIndex >= 0) {
      const systemBotUser = list[systemBotIndex];
      list.splice(systemBotIndex, 1);
      list.unshift(systemBotUser);
    } else {
      list.unshift(userStore.getUserByName(SYSTEM_BOT_USER_NAME)!);
    }
  }
  // If `showAll` is true, we insert a virtual user before the list.
  if (props.showAll) {
    const dummyAll = {
      ...unknownUser(),
      title: t("common.all"),
    };
    list.unshift(dummyAll);
  }
  const hasRBACFeature = hasFeature("bb.feature.rbac");
  return list.filter((user) => {
    // The previously selected item might no longer be applicable.
    // e.g. The select limits to DBA only and the selected principal is no longer a DBA
    // in such case, we still show the item.
    if (extractUserUID(user.name) === props.selectedId) {
      return true;
    }

    if (typeof props.customFilter === "function") {
      return props.customFilter(user);
    }

    if (!hasRBACFeature) return true;

    if (props.allowedRoleList.length === 0) {
      // Need not to filter by workspace role.
      return true;
    }

    return props.allowedRoleList.includes(user.userRole);
  });
});

watch(
  () => props.selectedId,
  (cur) => {
    state.selectedId = cur;
  }
);

const selectedPrincipal = computed(() =>
  userList.value.find((user) => extractUserUID(user.name) === state.selectedId)
);

const filter = (options: User[], query: string): User[] => {
  return filterUserListByKeyword(options, query);
};
</script>
