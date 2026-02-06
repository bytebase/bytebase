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
  getUserTypeByFullname,
  groupNamePrefix,
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
  projectName?: string;
  // allUsers is a special user that represents all users in the project.
  includeAllUsers?: boolean;
}>();

defineEmits<{
  // the value is the fullname, for example:
  // users/{email}, groups/{email}, workloadIdentities/{email}, serviceAccounts/{email}
  (event: "update:value", value: string[] | string | undefined): void;
}>();

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
  // Try fullname prefix first (e.g., serviceAccounts/foo@bar.com),
  // then fall back to email suffix check (e.g., foo@service.bytebase.com).
  // This supports both standard SA emails and legacy non-standard SA fullnames.
  const userType =
    getUserTypeByFullname(params.search) !== UserType.USER
      ? getUserTypeByFullname(params.search)
      : getUserTypeByEmail(params.search);
  switch (userType) {
    case UserType.SERVICE_ACCOUNT:
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
      return {
        nextPageToken: "",
        options: [getUserOption(serviceAccountToUser(sa))],
      };
    case UserType.WORKLOAD_IDENTITY:
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
      }),
      // Also search service accounts and workload identities to support
      // legacy accounts with non-standard email formats.
      handleSearchServiceAccount(params.search),
      handleSearchWorkloadIdentity(params.search)
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
      project: props.projectName,
      types: [UserType.USER],
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
      project: props.projectName,
    },
    pageToken: params.pageToken,
    pageSize: params.pageSize,
  });
  combinedPageToken.value.group = nextPageToken;
  return groups.map(getGroupOption);
};

const handleSearchServiceAccount = async (
  search: string
): Promise<ResourceSelectOption<AccountResource>[]> => {
  if (!hasGetServiceAccountPermission.value || !search) {
    return [];
  }
  const resp = await serviceAccountStore.listServiceAccounts({
    parent: props.projectName ?? "workspaces/-",
    pageSize: 10,
    pageToken: undefined,
    showDeleted: false,
    filter: { query: search },
  });
  return resp.serviceAccounts.map((sa) =>
    getUserOption(serviceAccountToUser(sa))
  );
};

const handleSearchWorkloadIdentity = async (
  search: string
): Promise<ResourceSelectOption<AccountResource>[]> => {
  if (!hasGetWorkloadIdentityPermission.value || !search) {
    return [];
  }
  const resp = await workloadIdentityStore.listWorkloadIdentities({
    parent: props.projectName ?? "workspaces/-",
    pageSize: 10,
    pageToken: undefined,
    showDeleted: false,
    filter: { query: search },
  });
  return resp.workloadIdentities.map((wi) =>
    getUserOption(workloadIdentityToUser(wi))
  );
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
