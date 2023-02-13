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
      #sectionHeader="{
        section,
      }: {
        section: BBTableSectionDataSource<Member>,
      }"
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
    </template>
    <template #body="{ rowData: member }">
      <BBTableCell :left-padding="4" class="table-cell">
        <div class="flex flex-row items-center space-x-2">
          <template v-if="'INVITED' == member.principal.status">
            <span
              class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-main text-main-text"
              >{{ $t("settings.members.invited") }}</span
            >
            <span class="textlabel">{{ member.principal.email }}</span>
          </template>
          <template v-else>
            <PrincipalAvatar :principal="member.principal" />
            <div class="flex flex-row">
              <div class="flex flex-col">
                <div class="flex flex-row items-center space-x-2">
                  <router-link
                    :to="`/u/${member.principal.id}`"
                    class="normal-link"
                  >
                    {{ member.principal.name }}
                  </router-link>
                  <span
                    v-if="currentUser.id == member.principal.id"
                    class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
                    >{{ $t("settings.members.yourself") }}</span
                  >
                  <span
                    v-if="member.principal.id === SYSTEM_BOT_ID"
                    class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
                  >
                    {{ $t("settings.members.system-bot") }}
                  </span>
                  <span
                    v-if="member.principal.type === 'SERVICE_ACCOUNT'"
                    class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
                  >
                    {{ $t("settings.members.service-account") }}
                  </span>
                </div>
                <span
                  v-if="member.principal.id !== SYSTEM_BOT_ID"
                  class="textlabel"
                  >{{ member.principal.email }}</span
                >
              </div>
              <template
                v-if="member.principal.type === 'SERVICE_ACCOUNT' && allowEdit"
              >
                <button
                  v-if="member.principal.serviceKey"
                  class="inline-flex gap-x-1 text-xs ml-3 my-1 px-2 rounded bg-gray-100 text-gray-500 hover:text-gray-700 hover:bg-gray-200 items-center"
                  @click.prevent="
                    () => copyServiceKey(member.principal.serviceKey)
                  "
                >
                  <heroicons-outline:clipboard class="w-4 h-4" />
                  {{ $t("settings.members.copy-service-key") }}
                </button>
                <button
                  v-else
                  class="inline-flex gap-x-1 text-xs ml-3 my-1 px-2 rounded bg-gray-100 text-gray-500 hover:text-gray-700 hover:bg-gray-200 items-center"
                  @click.prevent="() => tryResetServiceKey(member)"
                >
                  <heroicons-outline:reply class="w-4 h-4" />
                  {{ $t("settings.members.reset-service-key") }}
                </button>
              </template>
            </div>
          </template>
        </div>
      </BBTableCell>
      <BBTableCell class="whitespace-nowrap tooltip-wrapper w-36">
        <span v-if="changeRoleTooltip(member)" class="tooltip">{{
          changeRoleTooltip(member)
        }}</span>
        <RoleSelect
          :selected-role="member.role"
          :disabled="!allowChangeRole(member)"
          @change-role="
            (role) => {
              changeRole(member.id, role);
            }
          "
        />
      </BBTableCell>
      <BBTableCell>
        <BBButtonConfirm
          v-if="allowDeactivateMember(member)"
          :style="'ARCHIVE'"
          :require-confirm="true"
          :ok-text="$t('settings.members.action.deactivate')"
          :confirm-title="`${$t(
            'settings.members.action.deactivate-confirm-title'
          )} '${member.principal.name}'?`"
          :confirm-description="
            $t('settings.members.action.deactivate-confirm-description')
          "
          @confirm="changeRowStatus(member.id, 'ARCHIVED')"
        />
        <BBButtonConfirm
          v-else-if="allowActivateMember(member)"
          :style="'RESTORE'"
          :require-confirm="true"
          :ok-text="$t('settings.members.action.reactivate')"
          :confirm-title="`${$t(
            'settings.members.action.reactivate-confirm-title'
          )} '${member.principal.name}'?`"
          :confirm-description="''"
          @confirm="changeRowStatus(member.id, 'NORMAL')"
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
</template>

<script lang="ts">
import { computed, defineComponent, PropType, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import RoleSelect from "../components/RoleSelect.vue";
import PrincipalAvatar from "../components/PrincipalAvatar.vue";
import {
  MemberId,
  RoleType,
  MemberPatch,
  Member,
  RowStatus,
  SYSTEM_BOT_ID,
  Principal,
} from "../types";
import { BBTableSectionDataSource } from "../bbkit/types";
import { hasWorkspacePermission } from "../utils";
import {
  featureToRef,
  useCurrentUser,
  useMemberStore,
  usePrincipalStore,
  pushNotification,
} from "@/store";

const columnList = computed(() => [
  {
    title: "settings.members.table.account",
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
  targetServiceAccount?: Member;
}

export default defineComponent({
  name: "MemberTable",
  components: { RoleSelect, PrincipalAvatar },
  props: {
    memberList: {
      required: true,
      type: Array as PropType<Member[]>,
    },
  },
  setup(props) {
    const { t } = useI18n();
    const currentUser = useCurrentUser();
    const memberStore = useMemberStore();

    const hasRBACFeature = featureToRef("bb.feature.rbac");

    const state = reactive<LocalState>({
      showResetKeyAlert: false,
    });

    const dataSource = computed((): BBTableSectionDataSource<Member>[] => {
      const ownerList: Member[] = [];
      const dbaList: Member[] = [];
      const developerList: Member[] = [];
      for (const member of props.memberList) {
        if (member.role == "OWNER") {
          ownerList.push(member);
        }

        if (member.role == "DBA") {
          dbaList.push(member);
        }

        if (member.role == "DEVELOPER") {
          developerList.push(member);
        }
      }

      const dataSource: BBTableSectionDataSource<Member>[] = [];
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

    const hasMoreThanOneOwner = computed(
      () =>
        dataSource.value[0].list.filter(
          (m) => m.principal.type !== "SYSTEM_BOT"
        ).length > 1
    );

    const allowEdit = computed(() => {
      return hasWorkspacePermission(
        "bb.permission.workspace.manage-member",
        currentUser.value.role
      );
    });

    const allowChangeRole = (member: Member) => {
      if (member.principal.id === SYSTEM_BOT_ID) {
        return false;
      }
      return (
        hasRBACFeature.value &&
        allowEdit.value &&
        member.rowStatus == "NORMAL" &&
        (member.role != "OWNER" || hasMoreThanOneOwner.value)
      );
    };

    const changeRoleTooltip = (member: Member): string => {
      if (allowChangeRole(member)) {
        return "";
      }
      if (member.principal.id === SYSTEM_BOT_ID) {
        return t("settings.members.tooltip.cannot-change-role-of-systembot");
      }

      if (!hasRBACFeature.value) {
        return t("settings.members.tooltip.upgrade");
      }

      if (!allowEdit.value) {
        return t("settings.members.tooltip.not-allow-edit");
      }

      return t("settings.members.tooltip.not-allow-remove");
    };

    const allowDeactivateMember = (member: Member) => {
      if (member.principal.id === SYSTEM_BOT_ID) {
        return false;
      }
      return (
        allowEdit.value &&
        member.rowStatus == "NORMAL" &&
        (member.role != "OWNER" || hasMoreThanOneOwner.value)
      );
    };

    const allowActivateMember = (member: Member) => {
      return allowEdit.value && member.rowStatus == "ARCHIVED";
    };

    const changeRole = (id: MemberId, role: RoleType) => {
      const memberPatch: MemberPatch = {
        role,
      };
      memberStore.patchMember({
        id,
        memberPatch,
      });
    };

    const changeRowStatus = (id: MemberId, rowStatus: RowStatus) => {
      const memberPatch: MemberPatch = {
        rowStatus,
      };
      memberStore.patchMember({
        id,
        memberPatch,
      });
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

    const tryResetServiceKey = (member: Member) => {
      state.showResetKeyAlert = true;
      state.targetServiceAccount = member;
    };

    const resetServiceKey = () => {
      state.showResetKeyAlert = false;
      if (!state.targetServiceAccount) {
        return;
      }

      const memberId = state.targetServiceAccount.id;
      usePrincipalStore()
        .patchPrincipal({
          principalId: state.targetServiceAccount.principal.id,
          principalPatch: {
            type: state.targetServiceAccount.principal.type,
            refreshKey: true,
          },
        })
        .then((principal: Principal) => {
          memberStore.updatePrincipal(memberId, principal);
          return toClipboard(principal.serviceKey);
        })
        .then(() => {
          pushNotification({
            module: "bytebase",
            style: "INFO",
            title: t("settings.members.service-key-copied"),
          });
        });
    };

    return {
      SYSTEM_BOT_ID,
      columnList,
      state,
      currentUser,
      hasRBACFeature,
      dataSource,
      allowEdit,
      allowChangeRole,
      changeRoleTooltip,
      allowDeactivateMember,
      allowActivateMember,
      changeRole,
      changeRowStatus,
      copyServiceKey,
      resetServiceKey,
      tryResetServiceKey,
    };
  },
});
</script>
