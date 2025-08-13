<template>
  <ResourceSelect
    v-bind="$attrs"
    :remote="true"
    :loading="state.loading"
    :multiple="multiple"
    :value="user"
    :values="users"
    :size="size"
    :options="options"
    :custom-label="renderLabel"
    :placeholder="$t('settings.members.select-user', multiple ? 2 : 1)"
    :filter="filterByEmail"
    :show-resource-name="false"
    class="bb-user-select"
    @search="handleSearch"
    @update:value="(val) => $emit('update:user', val)"
    @update:values="(val) => $emit('update:users', val)"
  />
</template>

<script lang="tsx" setup>
import { useDebounceFn } from "@vueuse/core";
import { computed, watch, reactive, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import UserIcon from "~icons/heroicons-outline/user";
import UserAvatar from "@/components/User/UserAvatar.vue";
import ServiceAccountTag from "@/components/misc/ServiceAccountTag.vue";
import { extractUserId, useUserStore, type UserFilter } from "@/store";
import {
  DEBOUNCE_SEARCH_DELAY,
  SYSTEM_BOT_USER_NAME,
  UNKNOWN_USER_NAME,
  allUsersUser,
  isValidUserName,
} from "@/types";
import { UserType, type User } from "@/types/proto-es/v1/user_service_pb";
import { getDefaultPagination, ensureUserFullName } from "@/utils";
import ResourceSelect from "./ResourceSelect.vue";

const props = withDefaults(
  defineProps<{
    multiple?: boolean;
    user?: string;
    users?: string[];
    projectName?: string;
    // allUsers is a special user that represents all users in the project.
    includeAllUsers?: boolean;
    includeSystemBot?: boolean;
    includeServiceAccount?: boolean;
    autoReset?: boolean;
    filter?: (user: User, index: number) => boolean;
    size?: "tiny" | "small" | "medium" | "large";
  }>(),
  {
    multiple: false,
    user: undefined,
    users: undefined,
    projectName: undefined,
    includeAllUsers: false,
    includeSystemBot: false,
    includeServiceAccount: false,
    autoReset: true,
    filter: (_1: User, _2: number) => true,
    size: "medium",
  }
);

const emit = defineEmits<{
  (event: "update:user", value: string | undefined): void;
  (event: "update:users", value: string[]): void;
}>();

interface LocalState {
  loading: boolean;
  rawUserList: User[];
}

const { t } = useI18n();
const userStore = useUserStore();
const state = reactive<LocalState>({
  loading: false,
  rawUserList: [],
});

const getFilter = (search: string): UserFilter => {
  const filter = [];
  if (search) {
    filter.push(`(name.matches("${search}") || email.matches("${search}"))`);
  }
  const allowedType = [UserType.USER];
  if (props.includeServiceAccount) {
    allowedType.push(UserType.SERVICE_ACCOUNT);
  }
  if (props.includeSystemBot) {
    allowedType.push(UserType.SYSTEM_BOT);
  }

  return {
    query: search,
    project: props.projectName,
    types: allowedType,
  };
};

const initSelectedUsers = async (userIds: string[]) => {
  for (const userId of userIds) {
    const userName = ensureUserFullName(userId);
    if (isValidUserName(userName)) {
      const user = await userStore.getOrFetchUserByIdentifier(userName);
      if (!state.rawUserList.find((u) => u.name === user.name)) {
        state.rawUserList.unshift(user);
      }
    }
  }
};

const searchUsers = async (search: string) => {
  const { users } = await userStore.fetchUserList({
    filter: getFilter(search),
    pageSize: getDefaultPagination(),
  });
  return users.filter(props.filter);
};

const handleSearch = useDebounceFn(async (search: string) => {
  state.loading = true;
  try {
    const users = await searchUsers(search);
    state.rawUserList = users;
    if (!search) {
      if (props.includeAllUsers) {
        state.rawUserList.unshift(allUsersUser());
      }
      if (props.user) {
        await initSelectedUsers([props.user]);
      }
      if (props.users) {
        await initSelectedUsers(props.users);
      }
    }
  } finally {
    state.loading = false;
  }
}, DEBOUNCE_SEARCH_DELAY);

onMounted(async () => {
  await handleSearch("");
});

const renderAvatar = (user: User) => {
  if (user.name === UNKNOWN_USER_NAME) {
    return (
      <div class="bb-user-select--avatar w-6 h-6 rounded-full border-2 border-current flex justify-center items-center select-none bg-white">
        <UserIcon class="w-4 h-4 text-main text-current" />
      </div>
    );
  } else {
    return (
      <UserAvatar class="bb-user-select--avatar" user={user} size="SMALL" />
    );
  }
};

const renderLabel = (user: User) => {
  const avatar = renderAvatar(user);
  const title =
    user.name === SYSTEM_BOT_USER_NAME
      ? t("settings.members.system-bot")
      : user.title;
  const children = [<span class="truncate">{title}</span>];
  if (user.name !== UNKNOWN_USER_NAME && user.name !== SYSTEM_BOT_USER_NAME) {
    children.push(
      <span class="text-gray-400 truncate">{`(${user.email})`}</span>
    );
  }
  if (user.userType === UserType.SERVICE_ACCOUNT) {
    children.push(<ServiceAccountTag />);
  }
  return (
    <div class="w-full flex items-center gap-x-2">
      {avatar}
      <div class="flex flex-row justify-start items-center gap-x-0.5 truncate">
        {children}
      </div>
    </div>
  );
};

const options = computed(() => {
  return state.rawUserList.map((user) => {
    return {
      resource: user,
      value: extractUserId(user.name),
      label: user.title,
    };
  });
});

const filterByEmail = (pattern: string, user: User) => {
  return user.email.includes(pattern);
};

// The user list might change if props change, and the previous selected id
// might not exist in the new list. In such case, we need to invalidate the selection
// and emit the event.
const resetInvalidSelection = () => {
  if (!props.autoReset || props.multiple) {
    return;
  }
  if (state.loading) {
    return;
  }
  if (
    props.user &&
    !state.rawUserList.find((user) => extractUserId(user.name) === props.user)
  ) {
    emit("update:user", undefined);
  }
};

watch(
  [() => state.loading, () => props.user, state.rawUserList],
  resetInvalidSelection,
  {
    immediate: true,
  }
);
</script>

<style lang="postcss" scoped>
.bb-user-select :deep(.n-base-selection--active .bb-user-select--avatar) {
  opacity: 0.3;
}
</style>
