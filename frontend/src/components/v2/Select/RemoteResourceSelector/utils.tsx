import { NCheckbox, NTag } from "naive-ui";
import { type VNodeChild } from "vue";
import EllipsisText from "@/components/EllipsisText.vue";
import { HighlightLabelText } from "@/components/v2";
import { useUserStore } from "@/store";
import { unknownUser } from "@/types";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import {
  ensureUserFullName,
  hasWorkspacePermissionV2,
  isValidEmail,
} from "@/utils";
import type { ResourceSelectOption, SelectSize } from "./types";

export const getRenderLabelFunc =
  <T extends { name: string }>(params: {
    showResourceName?: boolean;
    multiple?: boolean;
    customLabel?: (resource: T, keyword: string) => VNodeChild;
  }) =>
  (option: ResourceSelectOption<T>, selected: boolean, searchText: string) => {
    const { resource, label } = option;
    const node = (
      <div class="py-1">
        {params.customLabel && resource ? (
          params.customLabel(resource, searchText)
        ) : (
          <HighlightLabelText keyword={searchText} text={label} />
        )}
        {params.showResourceName && resource && (
          <div>
            <EllipsisText class="opacity-60 textinfolabel">
              <HighlightLabelText keyword={searchText} text={resource.name} />
            </EllipsisText>
          </div>
        )}
      </div>
    );
    if (params.multiple) {
      return (
        <div class="flex items-center gap-x-2 py-2">
          <NCheckbox checked={selected} size="small" />
          {node}
        </div>
      );
    }

    return node;
  };

export const getRenderTagFunc =
  <T,>(params: {
    multiple?: boolean;
    size?: SelectSize;
    disabled?: boolean;
    customLabel?: (resource: T, keyword: string) => VNodeChild;
  }) =>
  ({
    option,
    handleClose,
  }: {
    option: ResourceSelectOption<T>;
    handleClose: () => void;
  }) => {
    const { resource, label } = option;
    const node =
      params.customLabel && resource ? params.customLabel(resource, "") : label;
    if (params.multiple) {
      return (
        <NTag
          size={params.size}
          closable={!params.disabled}
          onClose={handleClose}
        >
          {node}
        </NTag>
      );
    }
    return node;
  };

export interface UserResource extends User {
  // True for emails not yet registered as Bytebase users.
  isExternal?: boolean;
}

export const searchUsersWithFallback = async (params: {
  search: string;
  project?: string;
  pageToken: string;
  pageSize: number;
  allowArbitraryEmail?: boolean;
}): Promise<{
  users: UserResource[];
  nextPageToken: string;
}> => {
  const hasListUserPermission = hasWorkspacePermissionV2("bb.users.list");

  if (!hasListUserPermission) {
    return {
      users: [unknownUser(ensureUserFullName(params.search))],
      nextPageToken: "",
    };
  }

  const { nextPageToken, users } = await useUserStore().fetchUserList({
    filter: {
      query: params.search,
      project: params.project,
    },
    pageToken: params.pageToken,
    pageSize: params.pageSize,
  });

  // In SaaS mode (allowArbitraryEmail), if no existing user matches and the
  // search looks like an email, offer it as a selectable option so admins
  // can add emails for users who haven't signed up yet.
  if (
    params.allowArbitraryEmail &&
    users.length === 0 &&
    !params.pageToken &&
    isValidEmail(params.search)
  ) {
    const fullname = ensureUserFullName(params.search);
    const user: UserResource = {
      ...unknownUser(fullname),
      email: params.search,
      title: params.search,
      isExternal: true,
    };

    return {
      users: [user],
      nextPageToken: "",
    };
  }

  return {
    nextPageToken,
    users,
  };
};
