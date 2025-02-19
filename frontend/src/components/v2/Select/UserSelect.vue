<template>
  <ResourceSelect
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
    @update:value="(val) => $emit('update:user', val)"
    @update:values="(val) => $emit('update:users', val)"
  />
</template>

<script lang="tsx" setup>
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
import { State } from "@/types/proto/v1/common";
import { UserType, type User } from "@/types/proto/v1/user_service";
import { extractUserUID } from "@/utils";
import ResourceSelect from "./ResourceSelect.vue";

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
    autoReset?: boolean;
    filter?: (user: User, index: number) => boolean;
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
  return combinedUserList.value.map((user) => {
    return {
      resource: user,
      value: extractUserUID(user.name),
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
