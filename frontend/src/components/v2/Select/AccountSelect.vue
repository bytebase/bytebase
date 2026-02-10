<template>
  <div>
    <RemoteResourceSelector
      ref="remoteResourceSelectorRef"
      v-bind="$attrs"
      :multiple="multiple"
      :disabled="disabled"
      :size="size"
      :value="value"
      :tag="true"
      :remote="true"
      :additional-options="additionalOptions"
      :render-label="renderLabel"
      :render-tag="renderTag"
      :search="handleSearch"
      @update:value="(val) => $emit('update:value', val)"
    />
    <div v-if="errorMessage" class="text-red-600 mt-0.5">
      {{ errorMessage }}
    </div>
  </div>
</template>

<script lang="tsx" setup>
import { create } from "@bufbuild/protobuf";
import { computedAsync } from "@vueuse/core";
import { computed, ref } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import { useI18n } from "vue-i18n";
import GroupNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupNameCell.vue";
import { HighlightLabelText } from "@/components/v2";
import { UserNameCell } from "@/components/v2/Model/cells";
import {
  ensureGroupIdentifier,
  ensureServiceAccountFullName,
  ensureWorkloadIdentityFullName,
  extractServiceAccountId,
  extractWorkloadIdentityId,
  getUserFullNameByType,
  groupNamePrefix,
  projectNamePrefix,
  serviceAccountToUser,
  useGroupStore,
  useServiceAccountStore,
  useUserStore,
  useWorkloadIdentityStore,
  workloadIdentityToUser,
} from "@/store";
import {
  allUsersUser,
  getUserTypeByEmail,
  getUserTypeByFullname,
  unknownGroup,
  unknownUser,
} from "@/types";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import { ServiceAccountSchema } from "@/types/proto-es/v1/service_account_service_pb";
import { type User, UserType } from "@/types/proto-es/v1/user_service_pb";
import { WorkloadIdentitySchema } from "@/types/proto-es/v1/workload_identity_service_pb";
import { ensureUserFullName, hasWorkspacePermissionV2 } from "@/utils";
import { extractGrpcErrorMessage } from "@/utils/connect";
import RemoteResourceSelector from "./RemoteResourceSelector/index.vue";
import type {
  ResourceSelectOption,
  SelectSize,
} from "./RemoteResourceSelector/types";
import {
  getRenderLabelFunc,
  getRenderTagFunc,
} from "./RemoteResourceSelector/utils";

interface AccountResource {
  type: "user" | "group";
  name: string;
  resource: Group | User;
}

const props = defineProps<{
  multiple?: boolean;
  disabled?: boolean;
  size?: SelectSize;
  value?: string | string[] | undefined; // fullname
  // The parent can be workspace/- or projects/{id}
  parent?: string;
  // allUsers is a special user that represents all users in the project.
  includeAllUsers: boolean;
  includeServiceAccount: boolean;
  includeWorkloadIdentity: boolean;
}>();

defineEmits<{
  // the value is the fullname, for example:
  // users/{email}, groups/{email}, workloadIdentities/{email}, serviceAccounts/{email}
  (event: "update:value", value: string[] | string | undefined): void;
}>();

const { t } = useI18n();
const combinedPageToken = ref<{
  user: string;
  group: string;
}>({
  user: "",
  group: "",
});
const errorMessage = ref<string | undefined>();
const remoteResourceSelectorRef =
  ref<ComponentExposed<typeof RemoteResourceSelector<AccountResource>>>();

const userStore = useUserStore();
const groupStore = useGroupStore();
const serviceAccountStore = useServiceAccountStore();
const workloadIdentityStore = useWorkloadIdentityStore();

const hasListUserPermission = computed(() =>
  hasWorkspacePermissionV2("bb.users.list")
);
const hasListGroupPermission = computed(() =>
  hasWorkspacePermissionV2("bb.groups.list")
);
const hasGetWorkloadIdentityPermission = computed(() =>
  hasWorkspacePermissionV2("bb.workloadIdentities.get")
);
const hasGetServiceAccountPermission = computed(() =>
  hasWorkspacePermissionV2("bb.serviceAccounts.get")
);

const projectFullName = computed(() => {
  if (!props.parent) {
    return undefined;
  }
  if (props.parent.startsWith(projectNamePrefix)) {
    return props.parent;
  }
  return undefined;
});

const invalidAccountProjectMessage = ({
  project,
  email,
}: {
  email: string;
  project?: string;
}): string | undefined => {
  if (!project) {
    // workspace-level account can be added to both workspace or project level IAM policy.
    return;
  }
  if (projectFullName.value === project) {
    // Otherwise the account project must match
    return;
  }
  const message = [t("settings.members.invalid-account", { email })];
  if (!projectFullName.value) {
    message.push(t("settings.members.not-support-project-level-account"));
  } else {
    message.push(t("settings.members.project-not-match"));
  }
  return message.join(" ");
};

const getUserOption = (user: User): ResourceSelectOption<AccountResource> => ({
  resource: {
    type: "user",
    name: getUserFullNameByType(user),
    resource: user,
  },
  value: getUserFullNameByType(user),
  label: user.title,
});

const getGroupOption = (
  group: Group
): ResourceSelectOption<AccountResource> => ({
  resource: {
    type: "group",
    name: group.name,
    resource: group,
  },
  value: group.name,
  label: group.title,
});

const additionalOptions = computedAsync(async () => {
  const options: ResourceSelectOption<AccountResource>[] = [];
  if (props.includeAllUsers) {
    options.push(getUserOption(allUsersUser()));
  }

  let values: string[] = [];
  if (Array.isArray(props.value)) {
    values = props.value;
  } else if (props.value) {
    values = [props.value];
  }

  const groupNames: string[] = [];
  const userNames: string[] = [];
  for (const fullname of values) {
    if (fullname.startsWith(groupNamePrefix)) {
      groupNames.push(fullname);
      continue;
    }
    const userType = getUserTypeByFullname(fullname);
    switch (userType) {
      case UserType.SERVICE_ACCOUNT:
        if (!hasGetServiceAccountPermission.value) {
          continue;
        }
        const sa = await serviceAccountStore.getOrFetchServiceAccount(
          fullname,
          true
        );
        options.push(getUserOption(serviceAccountToUser(sa)));
        break;
      case UserType.WORKLOAD_IDENTITY:
        if (!hasGetWorkloadIdentityPermission.value) {
          continue;
        }
        const wi = await workloadIdentityStore.getOrFetchWorkloadIdentity(
          fullname,
          true
        );
        options.push(getUserOption(workloadIdentityToUser(wi)));
        break;
      default:
        userNames.push(fullname);
        break;
    }
  }

  const groups = await groupStore.batchGetOrFetchGroups(groupNames);
  for (const group of groups) {
    if (group) {
      options.push(getGroupOption(group));
    }
  }

  // Ensure users are fetched into store
  await userStore.batchGetOrFetchUsers(userNames);
  // Get all users from store
  for (const email of userNames) {
    const user = userStore.getUserByIdentifier(email);
    if (user) {
      options.push(getUserOption(user));
    }
  }

  return options;
}, []);

const handleSearch = async (params: {
  search: string;
  pageToken: string;
  pageSize: number;
}) => {
  errorMessage.value = undefined;
  if (!params.pageToken) {
    combinedPageToken.value = {
      user: "",
      group: "",
    };
  }

  try {
    const resp = await fetchAccounts(params);
    return resp;
  } catch (error) {
    errorMessage.value = extractGrpcErrorMessage(error);
    return {
      nextPageToken: "",
      options: [],
    };
  } finally {
    if (errorMessage.value) {
      setTimeout(() => remoteResourceSelectorRef.value?.close(), 500);
    }
  }
};

const fetchAccounts = async (params: {
  search: string;
  pageToken: string;
  pageSize: number;
}): Promise<{
  nextPageToken: string;
  options: ResourceSelectOption<AccountResource>[];
}> => {
  let userType = getUserTypeByFullname(params.search);
  if (userType === UserType.USER) {
    // If the params.search doesn't have the prefix,
    // we will try to guess the type by email suffix.
    userType = getUserTypeByEmail(params.search);
  }
  switch (userType) {
    case UserType.SERVICE_ACCOUNT:
      if (!props.includeServiceAccount) {
        return {
          nextPageToken: "",
          options: [],
        };
      }
      if (!hasGetServiceAccountPermission.value) {
        const email = extractServiceAccountId(params.search);
        return {
          nextPageToken: "",
          options: [
            getUserOption(
              serviceAccountToUser(
                create(ServiceAccountSchema, {
                  name: ensureServiceAccountFullName(params.search),
                  email: email,
                  title: email.split("@")[0],
                })
              )
            ),
          ],
        };
      }

      const sa = await serviceAccountStore.getOrFetchServiceAccount(
        params.search,
        true
      );
      errorMessage.value = invalidAccountProjectMessage(sa);
      if (errorMessage.value) {
        return {
          nextPageToken: "",
          options: [],
        };
      }
      return {
        nextPageToken: "",
        options: [getUserOption(serviceAccountToUser(sa))],
      };
    case UserType.WORKLOAD_IDENTITY:
      if (!props.includeWorkloadIdentity) {
        return {
          nextPageToken: "",
          options: [],
        };
      }
      if (!hasGetWorkloadIdentityPermission.value) {
        const email = extractWorkloadIdentityId(params.search);
        return {
          nextPageToken: "",
          options: [
            getUserOption(
              workloadIdentityToUser(
                create(WorkloadIdentitySchema, {
                  name: ensureWorkloadIdentityFullName(params.search),
                  email: email,
                  title: email.split("@")[0],
                })
              )
            ),
          ],
        };
      }

      const wi = await workloadIdentityStore.getOrFetchWorkloadIdentity(
        params.search,
        true
      );
      errorMessage.value = invalidAccountProjectMessage(wi);
      if (errorMessage.value) {
        return {
          nextPageToken: "",
          options: [],
        };
      }
      return {
        nextPageToken: "",
        options: [getUserOption(workloadIdentityToUser(wi))],
      };
    default:
  }

  const requests: Promise<ResourceSelectOption<AccountResource>[]>[] = [];

  if (!params.pageToken) {
    requests.push(
      handleSearchUser({
        search: params.search,
        pageToken: "",
        pageSize: params.pageSize,
      }),
      handleSearchGroup({
        search: params.search,
        pageToken: "",
        pageSize: params.pageSize,
      })
    );
  } else {
    if (combinedPageToken.value.user) {
      requests.push(
        handleSearchUser({
          search: params.search,
          pageToken: combinedPageToken.value.user,
          pageSize: params.pageSize,
        })
      );
    }
    if (combinedPageToken.value.group) {
      requests.push(
        handleSearchGroup({
          search: params.search,
          pageToken: combinedPageToken.value.group,
          pageSize: params.pageSize,
        })
      );
    }
  }

  const responses = await Promise.all(requests);
  return {
    nextPageToken:
      combinedPageToken.value.user || combinedPageToken.value.group,
    options: responses.reduce((resp, list) => {
      resp.push(...list);
      return resp;
    }, []),
  };
};

const handleSearchUser = async (params: {
  search: string;
  pageToken: string;
  pageSize: number;
}) => {
  if (!hasListUserPermission.value) {
    return [getUserOption(unknownUser(ensureUserFullName(params.search)))];
  }

  const { nextPageToken, users } = await userStore.fetchUserList({
    filter: {
      query: params.search,
      project: projectFullName.value,
    },
    pageToken: params.pageToken,
    pageSize: params.pageSize,
  });
  combinedPageToken.value.user = nextPageToken;
  return users.map(getUserOption);
};

const handleSearchGroup = async (params: {
  search: string;
  pageToken: string;
  pageSize: number;
}) => {
  if (
    !hasListGroupPermission.value &&
    params.search.startsWith(groupNamePrefix)
  ) {
    return [getGroupOption(unknownGroup(ensureGroupIdentifier(params.search)))];
  }

  const { nextPageToken, groups } = await groupStore.fetchGroupList({
    filter: {
      query: params.search,
      project: projectFullName.value,
    },
    pageToken: params.pageToken,
    pageSize: params.pageSize,
  });
  combinedPageToken.value.group = nextPageToken;
  return groups.map(getGroupOption);
};

const customLabel = (
  resource: AccountResource,
  keyword: string,
  showName: boolean
) => {
  if (resource.type === "group") {
    return (
      <GroupNameCell
        showName={showName}
        group={resource.resource as Group}
        showIcon={false}
        link={false}
        keyword={keyword}
      />
    );
  }

  const user = resource.resource as User;
  return (
    <UserNameCell
      user={user}
      allowEdit={false}
      showMfaEnabled={false}
      showSource={false}
      showEmail={showName}
      link={false}
      size="small"
      keyword={keyword}
      onClickUser={() => {}}
    >
      {{
        suffix: () =>
          !showName && (
            <span class="textinfolabel truncate">
              (<HighlightLabelText keyword={keyword} text={user.email} />)
            </span>
          ),
      }}
    </UserNameCell>
  );
};

const renderLabel = computed(() => {
  return getRenderLabelFunc({
    multiple: props.multiple,
    customLabel: (resource: AccountResource, keyword: string) =>
      customLabel(resource, keyword, true),
    showResourceName: false,
  });
});

const renderTag = computed(() => {
  return getRenderTagFunc({
    multiple: props.multiple,
    disabled: props.disabled,
    size: props.size,
    customLabel: (resource: AccountResource, keyword: string) =>
      customLabel(resource, keyword, false),
  });
});
</script>
