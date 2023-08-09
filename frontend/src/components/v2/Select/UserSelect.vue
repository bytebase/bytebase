<template>
  <NSelect
    :multiple="multiple"
    :value="value"
    :options="options"
    :filterable="true"
    :filter="filterByTitle"
    :virtual-scroll="true"
    :render-label="renderLabel"
    :fallback-option="false"
    :placeholder="$t('principal.select')"
    class="bb-user-select"
    style="width: 12rem"
    @update:value="handleValueUpdated"
  />
</template>

<script lang="ts" setup>
import { intersection } from "lodash-es";
import { NSelect, SelectOption } from "naive-ui";
import { computed, watch, watchEffect, h } from "vue";
import { useI18n } from "vue-i18n";
import UserIcon from "~icons/heroicons-outline/user";
import UserAvatar from "@/components/User/UserAvatar.vue";
import { useProjectV1Store, useUserStore } from "@/store";
import {
  PresetRoleType,
  SYSTEM_BOT_ID,
  SYSTEM_BOT_USER_NAME,
  UNKNOWN_ID,
  UNKNOWN_USER_NAME,
  unknownUser,
} from "@/types";
import { User, UserRole, UserType } from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";
import { extractUserUID, memberListInProjectV1 } from "@/utils";

interface UserSelectOption extends SelectOption {
  value: string;
  user: User;
}

const props = withDefaults(
  defineProps<{
    multiple?: boolean;
    user?: string;
    users?: string[];
    project?: string;
    includeAll?: boolean;
    includeSystemBot?: boolean;
    includeServiceAccount?: boolean;
    includeArchived?: boolean;
    allowedWorkspaceRoleList?: UserRole[];
    allowedProjectMemberRoleList?: string[];
    autoReset?: boolean;
    filter?: (user: User, index: number) => boolean;
  }>(),
  {
    multiple: false,
    user: undefined,
    users: undefined,
    project: undefined,
    includeAll: false,
    includeSystemBot: false,
    includeServiceAccount: false,
    includeArchived: false,
    allowedWorkspaceRoleList: () => [
      UserRole.OWNER,
      UserRole.DBA,
      UserRole.DEVELOPER,
    ],
    allowedProjectMemberRoleList: () => [
      PresetRoleType.OWNER,
      PresetRoleType.DEVELOPER,
    ],
    autoReset: true,
    filter: undefined,
  }
);

const emit = defineEmits<{
  (event: "update:user", value: string | undefined): void;
  (event: "update:users", value: string[]): void;
}>();

const { t } = useI18n();
const projectV1Store = useProjectV1Store();
const userStore = useUserStore();

const value = computed(() => {
  if (props.multiple) {
    return props.users || [];
  } else {
    return props.user;
  }
});

const prepare = () => {
  if (props.project && String(props.project) !== String(UNKNOWN_ID)) {
    projectV1Store.getOrFetchProjectByUID(props.project);
  } else {
    // Need not to fetch the entire member list since it's done in
    // root component
  }
};
watchEffect(prepare);

const getUserListFromProject = (projectUID: string) => {
  const project = projectV1Store.getProjectByUID(projectUID);
  const memberList = memberListInProjectV1(project, project.iamPolicy);
  const filteredUserList = memberList
    .filter((member) => {
      if (props.allowedProjectMemberRoleList.length === 0) {
        // Need not to filter by project member role
        return true;
      }
      return (
        intersection(member.roleList, props.allowedProjectMemberRoleList)
          .length > 0
      );
    })
    .map((member) => member.user);

  return filteredUserList;
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
      return props.allowedWorkspaceRoleList.includes(user.userRole);
    });
};

const rawUserList = computed(() => {
  const list =
    props.project && props.project !== String(UNKNOWN_ID)
      ? getUserListFromProject(props.project)
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

  if (props.user === String(SYSTEM_BOT_ID) || props.includeSystemBot) {
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
    emit("update:users", value as string[]);
  } else {
    emit("update:user", value as string);
  }
};

const renderAvatar = (user: User) => {
  if (user.name === UNKNOWN_USER_NAME) {
    return h(
      "div",
      {
        class:
          "bb-user-select--avatar w-6 h-6 rounded-full border-2 border-current flex justify-center items-center select-none bg-white",
      },
      h(UserIcon, {
        class: "w-4 h-4 text-main text-current",
      })
    );
  } else {
    return h(UserAvatar, {
      class: "bb-user-select--avatar",
      user,
      size: "SMALL",
    });
  }
};

const renderLabel = (option: SelectOption) => {
  const { user } = option as UserSelectOption;
  const avatar = renderAvatar(user);
  const text = h("span", { class: "truncate" }, user.title);
  return h(
    "div",
    {
      class: "flex items-center gap-x-2",
    },
    [avatar, text]
  );
};

const options = computed(() => {
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

<style>
.bb-user-select .n-base-selection--active .bb-user-select--avatar {
  opacity: 0.3;
}
</style>
