<template>
  <BBTable
    class="mt-2"
    :columnList="COLUMN_LIST"
    :sectionDataSource="dataSource"
    :compactSection="true"
    :showHeader="true"
    :rowClickable="false"
  >
    <template v-slot:header>
      <BBTableHeaderCell
        :leftPadding="4"
        class="w-auto table-cell"
        :title="COLUMN_LIST[0].title"
      />
      <BBTableHeaderCell class="w-8 table-cell" :title="COLUMN_LIST[1].title" />
      <BBTableHeaderCell
        class="w-72 table-cell"
        :title="COLUMN_LIST[2].title"
      />
      <BBTableHeaderCell
        class="w-auto table-cell"
        :title="COLUMN_LIST[3].title"
      />
    </template>
    <template v-slot:body="{ rowData: member }">
      <BBTableCell :leftPadding="4" class="table-cell">
        <div class="flex flex-row items-center space-x-2">
          <template v-if="'INVITED' == member.principal.status">
            <span
              class="
                inline-flex
                items-center
                px-2
                py-0.5
                rounded-lg
                text-xs
                font-semibold
                bg-main
                text-main-text
              "
            >
              Invited
            </span>
            <span class="textlabel">
              {{ member.principal.email }}
            </span>
          </template>
          <template v-else>
            <PrincipalAvatar :principal="member.principal" />
            <div class="flex flex-col">
              <div class="flex flex-row items-center space-x-2">
                <router-link
                  :to="`/u/${member.principal.id}`"
                  class="normal-link"
                  >{{ member.principal.name }}
                </router-link>
                <span
                  v-if="currentUser.id == member.principal.id"
                  class="
                    inline-flex
                    items-center
                    px-2
                    py-0.5
                    rounded-lg
                    text-xs
                    font-semibold
                    bg-green-100
                    text-green-800
                  "
                >
                  You
                </span>
              </div>
              <span class="textlabel">
                {{ member.principal.email }}
              </span>
            </div>
          </template>
        </div>
      </BBTableCell>
      <BBTableCell class="tooltip-wrapper">
        <span class="tooltip">{{ changeRoleTooltip(member) }}</span>
        <RoleSelect
          :selectedRole="member.role"
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
          <span>
            {{ humanizeTs(member.updatedTs) }}
          </span>
          <span>by</span>
          <router-link :to="`/u/${member.updater.id}`" class="normal-link"
            >{{ member.updater.name }}
          </router-link>
        </div>
      </BBTableCell>
      <BBTableCell>
        <BBButtonConfirm
          v-if="allowDeactivateMember(member)"
          :style="'ARCHIVE'"
          :requireConfirm="true"
          :okText="'Deactivate'"
          :confirmTitle="`Are you sure to deactivate '${member.principal.name}'`"
          :confirmDescription="'You can still reactivate later'"
          @confirm="changeRowStatus(member.id, 'ARCHIVED')"
        />
        <BBButtonConfirm
          v-else-if="allowActivateMember(member)"
          :style="'RESTORE'"
          :requireConfirm="true"
          :okText="'Reactivate'"
          :confirmTitle="`Are you sure to reactivate '${member.principal.name}'`"
          :confirmDescription="''"
          @confirm="changeRowStatus(member.id, 'NORMAL')"
        />
      </BBTableCell>
    </template>
  </BBTable>
  <div v-if="showUpgradeInfo" class="mt-6 border-t pt-4 border-block-border">
    <div class="flex flex-row items-center space-x-1">
      <svg
        class="w-6 h-6 text-accent"
        fill="currentColor"
        viewBox="0 0 20 20"
        xmlns="http://www.w3.org/2000/svg"
      >
        <path
          fill-rule="evenodd"
          d="M5 2a1 1 0 011 1v1h1a1 1 0 010 2H6v1a1 1 0 01-2 0V6H3a1 1 0 010-2h1V3a1 1 0 011-1zm0 10a1 1 0 011 1v1h1a1 1 0 110 2H6v1a1 1 0 11-2 0v-1H3a1 1 0 110-2h1v-1a1 1 0 011-1zM12 2a1 1 0 01.967.744L14.146 7.2 17.5 9.134a1 1 0 010 1.732l-3.354 1.935-1.18 4.455a1 1 0 01-1.933 0L9.854 12.8 6.5 10.866a1 1 0 010-1.732l3.354-1.935 1.18-4.455A1 1 0 0112 2z"
          clip-rule="evenodd"
        ></path>
      </svg>
      <router-link to="/setting/plan" class="text-lg accent-link"
        >Upgrade to unlock Owner and DBA roles</router-link
      >
    </div>
    <img class="w-full" src="../assets/role_management_screenshot.png" />
  </div>
</template>

<script lang="ts">
import { computed, PropType, reactive } from "vue";
import { useStore } from "vuex";
import RoleSelect from "../components/RoleSelect.vue";
import PrincipalAvatar from "../components/PrincipalAvatar.vue";
import { MemberID, RoleType, MemberPatch, Member, RowStatus } from "../types";
import { BBTableColumn, BBTableSectionDataSource } from "../bbkit/types";
import { isOwner } from "../utils";

const COLUMN_LIST: BBTableColumn[] = [
  {
    title: "Account",
  },
  {
    title: "Role",
  },
  {
    title: "Updated Time",
  },
  {
    title: "",
  },
];

interface LocalState {}

export default {
  name: "MemberTable",
  components: { RoleSelect, PrincipalAvatar },
  props: {
    memberList: {
      required: true,
      type: Object as PropType<Member[]>,
    },
  },
  setup(props, ctx) {
    const store = useStore();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const hasAdminFeature = computed(() =>
      store.getters["plan/feature"]("bb.admin")
    );

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
        title: "Owner",
        list: ownerList,
      });

      dataSource.push({
        title: "DBA",
        list: dbaList,
      });

      dataSource.push({
        title: "Developer",
        list: developerList,
      });

      return dataSource;
    });

    const allowEdit = computed(() => {
      return isOwner(currentUser.value.role);
    });

    const allowChangeRole = (member: Member) => {
      return (
        hasAdminFeature.value &&
        allowEdit.value &&
        member.rowStatus == "NORMAL" &&
        (member.role != "OWNER" || dataSource.value[0].list.length > 1)
      );
    };

    const changeRoleTooltip = (member: Member): string => {
      if (allowChangeRole(member)) {
        return "";
      }

      if (!hasAdminFeature.value) {
        return "Upgrade to Team plan to enable role management";
      }

      if (!allowEdit.value) {
        return "Only Owner can change the role";
      }

      return "Can not remove the last Owner";
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

    const showUpgradeInfo = computed(() => {
      return false;
      // return !hasAdminFeature.value && isOwner(currentUser.value.role);
    });

    const changeRole = (id: MemberID, role: RoleType) => {
      const memberPatch: MemberPatch = {
        role,
      };
      store.dispatch("member/patchMember", {
        id,
        memberPatch,
      });
    };

    const changeRowStatus = (id: MemberID, rowStatus: RowStatus) => {
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
      hasAdminFeature,
      showUpgradeInfo,
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
};
</script>
