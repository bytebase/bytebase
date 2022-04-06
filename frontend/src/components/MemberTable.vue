<template>
  <BBTable
    class="mt-2"
    :column-list="COLUMN_LIST"
    :section-data-source="dataSource"
    :compact-section="true"
    :show-header="true"
    :row-clickable="false"
  >
    <template #header>
      <BBTableHeaderCell
        :left-padding="4"
        class="w-auto table-cell"
        :title="$t(COLUMN_LIST[0].title)"
      />
      <BBTableHeaderCell
        class="w-8 table-cell"
        :title="$t(COLUMN_LIST[1].title)"
      />
      <BBTableHeaderCell
        class="w-72 table-cell"
        :title="$t(COLUMN_LIST[2].title)"
      />
      <BBTableHeaderCell
        class="w-auto table-cell"
        :title="$t(COLUMN_LIST[3].title)"
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
            <div class="flex flex-col">
              <div class="flex flex-row items-center space-x-2">
                <router-link
                  :to="`/u/${member.principal.id}`"
                  class="normal-link"
                  >{{ member.principal.name }}</router-link
                >
                <span
                  v-if="currentUser.id == member.principal.id"
                  class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
                  >{{ $t("settings.members.yourself") }}</span
                >
              </div>
              <span class="textlabel">{{ member.principal.email }}</span>
            </div>
          </template>
        </div>
      </BBTableCell>
      <BBTableCell class="whitespace-nowrap tooltip-wrapper">
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
      <BBTableCell class="table-cell">
        <div class="flex flex-row items-center space-x-1">
          <span>{{ humanizeTs(member.updatedTs) }}</span>
          <span>by</span>
          <router-link :to="`/u/${member.updater.id}`" class="normal-link">{{
            member.updater.name
          }}</router-link>
        </div>
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
</template>

<script lang="ts">
import { computed, defineComponent, PropType, reactive } from "vue";
import { useStore } from "vuex";
import { useI18n } from "vue-i18n";
import RoleSelect from "../components/RoleSelect.vue";
import PrincipalAvatar from "../components/PrincipalAvatar.vue";
import { MemberId, RoleType, MemberPatch, Member, RowStatus } from "../types";
import { BBTableColumn, BBTableSectionDataSource } from "../bbkit/types";
import { isOwner } from "../utils";
import { featureToRef, useCurrentUser } from "@/store";

const COLUMN_LIST: BBTableColumn[] = [
  {
    title: "settings.members.table.account",
  },
  {
    title: "settings.members.table.role",
  },
  {
    title: "settings.members.table.update-time",
  },
  {
    title: "",
  },
];

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface LocalState {}

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
    const store = useStore();

    const currentUser = useCurrentUser();

    const hasRBACFeature = featureToRef("bb.feature.rbac");

    const state = reactive<LocalState>({});

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

    const allowEdit = computed(() => {
      return isOwner(currentUser.value.role);
    });

    const allowChangeRole = (member: Member) => {
      return (
        hasRBACFeature.value &&
        allowEdit.value &&
        member.rowStatus == "NORMAL" &&
        (member.role != "OWNER" || dataSource.value[0].list.length > 1)
      );
    };

    const changeRoleTooltip = (member: Member): string => {
      if (allowChangeRole(member)) {
        return "";
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
      return (
        allowEdit.value &&
        member.rowStatus == "NORMAL" &&
        (member.role != "OWNER" || dataSource.value[0].list.length > 1)
      );
    };

    const allowActivateMember = (member: Member) => {
      return allowEdit.value && member.rowStatus == "ARCHIVED";
    };

    const changeRole = (id: MemberId, role: RoleType) => {
      const memberPatch: MemberPatch = {
        role,
      };
      store.dispatch("member/patchMember", {
        id,
        memberPatch,
      });
    };

    const changeRowStatus = (id: MemberId, rowStatus: RowStatus) => {
      const memberPatch: MemberPatch = {
        rowStatus,
      };
      store.dispatch("member/patchMember", {
        id,
        memberPatch,
      });
    };

    return {
      COLUMN_LIST,
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
    };
  },
});
</script>
