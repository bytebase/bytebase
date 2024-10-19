<template>
  <NSelect
    :filterable="true"
    :virtual-scroll="true"
    :multiple="multiple"
    :value="value"
    :options="options"
    :fallback-option="fallbackOption"
    :filter="filterByTitle"
    :render-label="renderLabel"
    :placeholder="$t('settings.members.select-user', multiple ? 2 : 1)"
    :size="size"
    class="bb-user-select"
    @update:value="handleValueUpdated"
  />
</template>

<script lang="tsx" setup>
import type { SelectGroupOption, SelectOption, SelectProps } from "naive-ui";
import { NSelect } from "naive-ui";
import { computed, watch, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import UserIcon from "~icons/heroicons-outline/user";
import {
  getMemberBindingsByRole,
  getMemberBindings,
} from "@/components/Member/utils";
import UserAvatar from "@/components/User/UserAvatar.vue";
import ServiceAccountTag from "@/components/misc/ServiceAccountTag.vue";
import { useProjectV1Store, useUserStore, useWorkspaceV1Store } from "@/store";
import {
  SYSTEM_BOT_USER_NAME,
  UNKNOWN_ID,
  UNKNOWN_USER_NAME,
  allUsersUser,
  unknownUser,
  PresetRoleType,
  isValidProjectName,
  PRESET_WORKSPACE_ROLES,
} from "@/types";
import { UserType, type User } from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";
import { extractUserUID } from "@/utils";

export interface UserSelectOption extends SelectOption {
  value: string;
  user: User;
}

const props = withDefaults(
  defineProps<{
    multiple?: boolean;
    user?: string;
    users?: string[];
    projectName?: string;
    includeAll?: boolean;
    // allUsers is a special user that represents all users in the project.
    includeAllUsers?: boolean;
    includeSystemBot?: boolean;
    includeServiceAccount?: boolean;
    includeArchived?: boolean;
    allowedWorkspaceRoleList?: string[];
    allowedProjectMemberRoleList?: string[];
    autoReset?: boolean;
    filter?: (user: User, index: number) => boolean;
    mapOptions?: (users: User[]) => (UserSelectOption | SelectGroupOption)[];
    fallbackOption?: SelectProps["fallbackOption"];
    size?: "tiny" | "small" | "medium" | "large";
  }>(),
  {
    multiple: false,
    user: undefined,
    users: undefined,
    projectName: undefined,
    includeAll: false,
    includeAllUsers: false,
    includeSystemBot: false,
    includeServiceAccount: false,
    includeArchived: false,
    allowedWorkspaceRoleList: () => [
      PresetRoleType.WORKSPACE_ADMIN,
      PresetRoleType.WORKSPACE_DBA,
      PresetRoleType.WORKSPACE_MEMBER,
    ],
    autoReset: true,
    filter: undefined,
    mapOptions: undefined,
    fallbackOption: false,
    size: "medium",
  }
);

const emit = defineEmits<{
  (event: "update:user", value: string | undefined): void;
  (event: "update:users", value: string[]): void;
}>();

const { t } = useI18n();
const projectV1Store = useProjectV1Store();
const userStore = useUserStore();
const workspaceStore = useWorkspaceV1Store();

const value = computed(() => {
  if (props.multiple) {
    return props.users || [];
  } else {
    return props.user;
  }
});

const prepare = () => {
  if (props.projectName && isValidProjectName(props.projectName)) {
    projectV1Store.getOrFetchProjectByName(props.projectName);
  } else {
    // Need not to fetch the entire member list since it's done in
    // root component
  }
};
watchEffect(prepare);

const getUserListFromProject = (projectName: string) => {
  const project = projectV1Store.getProjectByName(projectName);
  const memberMap = getMemberBindingsByRole({
    policies: [
      {
        level: "WORKSPACE",
        policy: workspaceStore.workspaceIamPolicy,
      },
      {
        level: "PROJECT",
        policy: project.iamPolicy,
      },
    ],
    searchText: "",
    ignoreRoles: new Set(PRESET_WORKSPACE_ROLES),
  });

  return getMemberBindings(memberMap)
    .map((binding) => binding.user)
    .filter((u) => u) as User[];
};

const getUserListFromWorkspace = () => {
  return userStore.userList
    .filter((user) => {
      if (props.includeArchived) return true;
      return user.state === State.ACTIVE;
    })
    .filter((user) => {
      if (props.allowedWorkspaceRoleList.length === 0) {
        // Need not to filter by workspace role
        return true;
      }
      return [...workspaceStore.getWorkspaceRolesByEmail(user.email)].some(
        (role) => props.allowedWorkspaceRoleList.includes(role)
      );
    });
};

const rawUserList = computed(() => {
  const list =
    props.projectName && isValidProjectName(props.projectName)
      ? getUserListFromProject(props.projectName)
      : getUserListFromWorkspace();

  return list.filter((user) => {
    if (
      user.userType === UserType.SERVICE_ACCOUNT &&
      !props.includeServiceAccount
    ) {
      return false;
    }

    if (user.userType === UserType.SYSTEM_BOT && !props.includeSystemBot) {
      return false;
    }

    return true;
  });
});

const combinedUserList = computed(() => {
  let list = [...rawUserList.value];

  list.sort((a, b) => {
    return (
      parseInt(extractUserUID(a.name), 10) -
      parseInt(extractUserUID(b.name), 10)
    );
  });

  if (props.filter) {
    list = list.filter(props.filter);
  }

  if (props.includeSystemBot) {
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
  if (props.includeAllUsers) {
    list.unshift(allUsersUser());
  }
  if (props.user === String(UNKNOWN_ID) || props.includeAll) {
    const dummyAll = {
      ...unknownUser(),
      title: t("common.all"),
    };
    list.unshift(dummyAll);
  }

  return list;
});

const handleValueUpdated = (value: string | string[]) => {
  if (props.multiple) {
    if (!value) {
      // normalize value
      value = [];
    }
    emit("update:users", value as string[]);
  } else {
    if (value === null) {
      // normalize value
      value = "";
    }
    emit("update:user", value as string);
  }
};

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

const renderLabel = (option: SelectOption) => {
  if (option.type === "group") {
    return option.label as string;
  }
  const { user } = option as UserSelectOption;
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
  if (props.mapOptions) {
    return props.mapOptions(combinedUserList.value);
  }
  return combinedUserList.value.map<UserSelectOption>((user) => {
    return {
      user,
      value: extractUserUID(user.name),
      label: user.title,
    };
  });
});

const filterByTitle = (pattern: string, option: SelectOption) => {
  const { user } = option as UserSelectOption;
  pattern = pattern.toLowerCase();
  return (
    user.title.toLowerCase().includes(pattern) ||
    user.email.includes(pattern.toLowerCase())
  );
};

// The user list might change if props change, and the previous selected id
// might not exist in the new list. In such case, we need to invalidate the selection
// and emit the event.
const resetInvalidSelection = () => {
  if (!props.autoReset) return;
  if (
    props.user &&
    !combinedUserList.value.find(
      (user) => extractUserUID(user.name) === props.user
    )
  ) {
    emit("update:user", undefined);
  }
};

watch(
  [() => props.user, () => props.users, combinedUserList],
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
