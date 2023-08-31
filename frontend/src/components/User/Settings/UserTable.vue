<template>
  <BBTable
    class="mt-2"
    :column-list="columnList"
    :section-data-source="dataSource"
    :compact-section="true"
    :show-header="true"
    :row-clickable="false"
  >
    <template
      #sectionHeader="{ section }: { section: BBTableSectionDataSource<User> }"
    >
      <span>{{ section.title }}</span>
      <span
        v-if="section.list.length > 0"
        class="ml-0.5 font-normal text-control-light"
      >
        ({{ section.list.length }})
      </span>
    </template>

    <template #header>
      <BBTableHeaderCell
        :left-padding="4"
        class="w-auto table-cell"
        :title="$t(columnList[0].title)"
      />
      <BBTableHeaderCell
        class="w-20 table-cell"
        :title="$t(columnList[1].title)"
      />
      <BBTableHeaderCell
        class="w-20 table-cell"
        :title="$t(columnList[2].title)"
      />
      <BBTableHeaderCell
        class="w-20 table-cell"
        :title="$t(columnList[3].title)"
      />
    </template>
    <template #body="{ rowData: user }: { rowData: User }">
      <BBTableCell :left-padding="4" class="table-cell">
        <div class="flex flex-row items-center space-x-2">
          <template v-if="false">
            <span
              class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-main text-main-text"
              >{{ $t("settings.members.invited") }}</span
            >
            <span class="textlabel">{{ user.email }}</span>
          </template>
          <template v-else>
            <UserAvatar :user="user" />

            <div class="flex flex-row">
              <div class="flex flex-col">
                <div class="flex flex-row items-center space-x-2">
                  <router-link
                    :to="`/u/${extractUserUID(user.name)}`"
                    class="normal-link"
                  >
                    {{ user.title }}
                  </router-link>
                  <span
                    v-if="currentUserV1.name === user.name"
                    class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
                  >
                    {{ $t("settings.members.yourself") }}
                  </span>
                  <span
                    v-if="user.name === SYSTEM_BOT_USER_NAME"
                    class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
                  >
                    {{ $t("settings.members.system-bot") }}
                  </span>
                  <span
                    v-if="user.userType === UserType.SERVICE_ACCOUNT"
                    class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
                  >
                    {{ $t("settings.members.service-account") }}
                  </span>
                </div>
                <span
                  v-if="user.name !== SYSTEM_BOT_USER_NAME"
                  class="textlabel"
                >
                  {{ user.email }}
                </span>
              </div>
              <template
                v-if="user.userType === UserType.SERVICE_ACCOUNT && allowEdit"
              >
                <button
                  v-if="user.serviceKey"
                  class="inline-flex gap-x-1 text-xs ml-3 my-1 px-2 rounded bg-gray-100 text-gray-500 hover:text-gray-700 hover:bg-gray-200 items-center"
                  @click.prevent="() => copyServiceKey(user.serviceKey)"
                >
                  <heroicons-outline:clipboard class="w-4 h-4" />
                  {{ $t("settings.members.copy-service-key") }}
                </button>
                <button
                  v-else
                  class="inline-flex gap-x-1 text-xs ml-3 my-1 px-2 rounded bg-gray-100 text-gray-500 hover:text-gray-700 hover:bg-gray-200 items-center"
                  @click.prevent="() => tryResetServiceKey(user)"
                >
                  <heroicons-outline:reply class="w-4 h-4" />
                  {{ $t("settings.members.reset-service-key") }}
                </button>
              </template>
            </div>
          </template>
        </div>
      </BBTableCell>
      <BBTableCell class="whitespace-nowrap tooltip-wrapper w-auto">
        <span
          v-if="is2FAEnabled(user)"
          class="text-xs p-1 px-2 rounded-lg bg-green-600 text-white"
        >
          {{ $t("two-factor.enabled") }}
        </span>
      </BBTableCell>
      <BBTableCell class="whitespace-nowrap tooltip-wrapper w-36">
        <span v-if="changeRoleTooltip(user)" class="tooltip">{{
          changeRoleTooltip(user)
        }}</span>
        <RoleSelect
          :role="user.userRole"
          :disabled="!allowChangeRole(user)"
          @update:role="changeRole(user, $event)"
        />
      </BBTableCell>
      <BBTableCell>
        <BBButtonConfirm
          v-if="allowDeactivateMember(user)"
          :style="'ARCHIVE'"
          :require-confirm="true"
          :ok-text="$t('settings.members.action.deactivate')"
          :confirm-title="deactivateConfirmation(user).title"
          :confirm-description="deactivateConfirmation(user).description"
          @confirm="changeRowStatus(user, State.DELETED)"
        />
        <BBButtonConfirm
          v-else-if="allowActivateMember(user)"
          :style="'RESTORE'"
          :require-confirm="true"
          :ok-text="$t('settings.members.action.reactivate')"
          :confirm-title="`${$t(
            'settings.members.action.reactivate-confirm-title'
          )} '${user.title}'?`"
          :confirm-description="''"
          @confirm="changeRowStatus(user, State.ACTIVE)"
        />
      </BBTableCell>
    </template>
  </BBTable>
  <BBAlert
    v-if="state.showResetKeyAlert"
    :style="'CRITICAL'"
    :ok-text="$t('settings.members.reset-service-key')"
    :title="$t('settings.members.reset-service-key')"
    :description="$t('settings.members.reset-service-key-alert')"
    @ok="resetServiceKey"
    @cancel="state.showResetKeyAlert = false"
  />
  <BBAlertDialog
    ref="removeSelfOwnerDialog"
    :style="'CRITICAL'"
    :ok-text="$t('common.confirm')"
    :title="$t('settings.members.remove-self-owner.title')"
    :description="$t('settings.members.remove-self-owner.description')"
  />
</template>

<script lang="ts" setup>
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import { cloneDeep } from "lodash-es";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { BBAlertDialog } from "@/bbkit";
import { BBTableSectionDataSource } from "@/bbkit/types";
import { RoleSelect } from "@/components/v2";
import {
  featureToRef,
  useCurrentUserV1,
  pushNotification,
  useUserStore,
} from "@/store";
import { SYSTEM_BOT_USER_NAME } from "@/types";
import { User, UserRole, UserType } from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";
import { hasWorkspacePermissionV1, extractUserUID } from "@/utils";
import UserAvatar from "../UserAvatar.vue";
import { copyServiceKeyToClipboardIfNeeded } from "./common";

const columnList = computed(() => [
  {
    title: "settings.members.table.account",
  },
  {
    title: "",
  },
  {
    title: "settings.members.table.role",
  },
  {
    title: "",
  },
]);

interface LocalState {
  showResetKeyAlert: boolean;
  targetServiceAccount?: User;
}

const props = defineProps<{
  userList: User[];
}>();

const { t } = useI18n();
const currentUserV1 = useCurrentUserV1();
const userStore = useUserStore();
const removeSelfOwnerDialog = ref<InstanceType<typeof BBAlertDialog>>();

const hasRBACFeature = featureToRef("bb.feature.rbac");

const state = reactive<LocalState>({
  showResetKeyAlert: false,
});

const dataSource = computed((): BBTableSectionDataSource<User>[] => {
  const ownerList: User[] = [];
  const dbaList: User[] = [];
  const developerList: User[] = [];
  for (const member of props.userList) {
    if (member.userRole === UserRole.OWNER) {
      ownerList.push(member);
    }

    if (member.userRole === UserRole.DBA) {
      dbaList.push(member);
    }

    if (member.userRole === UserRole.DEVELOPER) {
      developerList.push(member);
    }
  }

  const dataSource: BBTableSectionDataSource<User>[] = [];
  dataSource.push({
    title: t("common.role.owner"),
    list: ownerList,
  });

  dataSource.push({
    title: t("common.role.dba"),
    list: dbaList,
  });

  dataSource.push({
    title: t("common.role.developer"),
    list: developerList,
  });

  return dataSource;
});

const hasMoreThanOneOwner = computed(() => {
  return (
    dataSource.value[0].list.filter((m) => m.userType === UserType.USER)
      .length > 1
  );
});

const allowEdit = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-member",
    currentUserV1.value.userRole
  );
});

const allowChangeRole = (user: User) => {
  if (
    user.name === SYSTEM_BOT_USER_NAME ||
    user.userType === UserType.SERVICE_ACCOUNT
  ) {
    return false;
  }

  return (
    hasRBACFeature.value &&
    allowEdit.value &&
    user.state === State.ACTIVE &&
    (user.userRole !== UserRole.OWNER || hasMoreThanOneOwner.value)
  );
};

const changeRoleTooltip = (user: User): string => {
  if (allowChangeRole(user)) {
    return "";
  }
  // Non-actived user cannot be changed role, so the tooltip should be empty.
  if (user.state !== State.ACTIVE) {
    return "";
  }

  if (
    user.name === SYSTEM_BOT_USER_NAME ||
    user.userType === UserType.SERVICE_ACCOUNT
  ) {
    return t(
      "settings.members.tooltip.cannot-change-role-of-systembot-or-service-account"
    );
  }

  if (!hasRBACFeature.value) {
    return t("settings.members.tooltip.upgrade");
  }

  if (!allowEdit.value) {
    return t("settings.members.tooltip.not-allow-edit");
  }

  return t("settings.members.tooltip.not-allow-remove");
};

const is2FAEnabled = (user: User): boolean => {
  return user.mfaEnabled;
};

const allowDeactivateMember = (user: User) => {
  if (user.name === SYSTEM_BOT_USER_NAME) {
    return false;
  }
  return (
    allowEdit.value &&
    user.state === State.ACTIVE &&
    (user.userRole !== UserRole.OWNER || hasMoreThanOneOwner.value)
  );
};

const allowActivateMember = (user: User) => {
  return allowEdit.value && user.state === State.DELETED;
};

const deactivateConfirmation = (user: User) => {
  const me = currentUserV1.value;
  if (user.name === me.name && user.userRole === UserRole.OWNER) {
    return {
      title: t("settings.members.remove-self-owner.title"),
      description: t("settings.members.remove-self-owner.description"),
    };
  }
  return {
    title: `${t("settings.members.action.deactivate-confirm-title")} '${
      user.title
    }'?`,
    description: t("settings.members.action.deactivate-confirm-description"),
  };
};

const changeRole = async (user: User, role: UserRole) => {
  const me = currentUserV1.value;
  if (user.name === me.name) {
    if (user.userRole === UserRole.OWNER && role !== UserRole.OWNER) {
      const dialog = removeSelfOwnerDialog.value;
      if (!dialog) {
        throw new Error("dialog is not loaded");
      }
      const result = await dialog.open();
      if (!result) {
        return;
      }
    }
  }

  const userPatch = cloneDeep(user);
  userPatch.userRole = role;
  userStore.updateUser({
    user: userPatch,
    updateMask: ["role"],
    regenerateTempMfaSecret: false,
    regenerateRecoveryCodes: false,
  });
};

const changeRowStatus = (user: User, state: State) => {
  if (state === State.ACTIVE) {
    userStore.restoreUser(user);
  } else {
    userStore.archiveUser(user);
  }
};

const copyServiceKey = (serviceKey: string) => {
  toClipboard(serviceKey).then(() => {
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: t("settings.members.service-key-copied"),
    });
  });
};

const tryResetServiceKey = (user: User) => {
  state.showResetKeyAlert = true;
  state.targetServiceAccount = user;
};

const resetServiceKey = () => {
  state.showResetKeyAlert = false;
  const user = state.targetServiceAccount;

  if (!user) {
    return;
  }
  userStore
    .updateUser({
      user,
      updateMask: ["service_key"],
      regenerateRecoveryCodes: false,
      regenerateTempMfaSecret: false,
    })
    .then((updatedUser) => {
      copyServiceKeyToClipboardIfNeeded(updatedUser);
    });
};
</script>
