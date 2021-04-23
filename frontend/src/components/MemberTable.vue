<template>
  <p class="text-lg font-medium leading-7 text-main">Current members</p>
  <BBTable
    class="mt-2"
    :columnList="columnList"
    :sectionDataSource="dataSource"
    :compactSection="true"
    :showHeader="true"
    :rowClickable="false"
  >
    <template v-slot:header>
      <BBTableHeaderCell
        :leftPadding="4"
        class="w-auto table-cell"
        :title="columnList[0].title"
      />
      <template v-if="hasAdminFeature">
        <BBTableHeaderCell
          class="w-8 table-cell"
          :title="columnList[1].title"
        />
        <BBTableHeaderCell
          class="w-72 table-cell"
          :title="columnList[2].title"
        />
        <BBTableHeaderCell
          class="w-auto table-cell"
          :title="columnList[3].title"
        />
      </template>
      <template v-else>
        <BBTableHeaderCell
          class="w-72 table-cell"
          :title="columnList[1].title"
        />
        <BBTableHeaderCell
          class="w-auto table-cell"
          :title="columnList[2].title"
        />
      </template>
    </template>
    <template v-slot:body="{ rowData: memberUI }">
      <BBTableCell :leftPadding="4" class="table-cell">
        <div class="flex flex-row items-center space-x-2">
          <template v-if="'INVITED' == memberUI.principal.status">
            <span
              class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-main text-main-text"
            >
              Invited
            </span>
            <span class="textlabel">
              {{ memberUI.principal.email }}
            </span>
          </template>
          <template v-else>
            <BBAvatar :username="memberUI.principal.name" />
            <div class="flex flex-col">
              <div class="flex flex-row items-center space-x-2">
                <router-link
                  :to="`/u/${memberUI.principal.id}`"
                  class="normal-link"
                  >{{ memberUI.principal.name }}
                </router-link>
                <span
                  v-if="currentUser.id == memberUI.principal.id"
                  class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
                >
                  You
                </span>
              </div>
              <span class="textlabel">
                {{ memberUI.principal.email }}
              </span>
            </div>
          </template>
        </div>
      </BBTableCell>
      <BBTableCell v-if="hasAdminFeature" class="">
        <RoleSelect
          :selectedRole="memberUI.role"
          :disabled="!allowChangeRole(memberUI.role)"
          @change-role="
            (role) => {
              changeRole(memberUI.id, role);
            }
          "
        />
      </BBTableCell>
      <BBTableCell class="table-cell">
        <div class="flex flex-row items-center space-x-1">
          <span>
            {{ humanizeTs(memberUI.updatedTs) }}
          </span>
          <span>by</span>
          <router-link :to="`/u/${memberUI.updater.id}`" class="normal-link"
            >{{ memberUI.updater.name }}
          </router-link>
        </div>
      </BBTableCell>
      <BBTableCell>
        <BBButtonConfirm
          v-if="allowChangeRole(memberUI.role)"
          :requireConfirm="true"
          :okText="'Revoke'"
          :confirmTitle="`Are you sure to revoke '${memberUI.role}' from '${memberUI.principal.name}'`"
          @confirm="deleteRole(memberUI.id)"
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
import { computed, reactive, watchEffect } from "vue";
import { useStore } from "vuex";
import RoleSelect from "../components/RoleSelect.vue";
import { Principal, Member, MemberId, RoleType, MemberPatch } from "../types";
import { BBTableColumn, BBTableSectionDataSource } from "../bbkit/types";
import { isOwner } from "../utils";

type MemberUI = Member & {
  principal: Principal;
  updater: Principal;
};

interface LocalState {}

export default {
  name: "MemberTable",
  components: { RoleSelect },
  props: {},
  setup(props, ctx) {
    const store = useStore();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const hasAdminFeature = computed(() =>
      store.getters["plan/feature"]("bb.admin")
    );

    const state = reactive<LocalState>({});

    const prepareMemberList = () => {
      store.dispatch("member/fetchMemberList").catch((error) => {
        console.log(error);
      });
    };

    watchEffect(prepareMemberList);

    const dataSource = computed((): BBTableSectionDataSource<MemberUI>[] => {
      const ownerList: MemberUI[] = [];
      const dbaList: MemberUI[] = [];
      const developerList: MemberUI[] = [];
      for (const member of store.getters["member/memberList"]()) {
        const memberUI = {
          ...member,
          principal: store.getters["principal/principalById"](
            member.principalId
          ),
          updater: store.getters["principal/principalById"](member.updaterId),
        };

        if (memberUI.role == "OWNER") {
          ownerList.push(memberUI);
        }

        if (memberUI.role == "DBA") {
          dbaList.push(memberUI);
        }

        if (memberUI.role == "DEVELOPER") {
          developerList.push(memberUI);
        }
      }

      const dataSource: BBTableSectionDataSource<MemberUI>[] = [];
      if (hasAdminFeature.value) {
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
      } else {
        ownerList.push(...dbaList);
        ownerList.push(...developerList);

        dataSource.push({
          title: "Member",
          list: ownerList,
        });
      }
      return dataSource;
    });

    const columnList = computed((): BBTableColumn[] => {
      return hasAdminFeature.value
        ? [
            {
              title: "Account",
            },
            {
              title: "Role",
            },
            {
              title: "Granted Time",
            },
            {
              title: "",
            },
          ]
        : [
            {
              title: "Account",
            },
            {
              title: "Granted Time",
            },
            {
              title: "",
            },
          ];
    });

    const allowEdit = computed(() => {
      return isOwner(currentUser.value.role);
    });

    const allowChangeRole = (role: RoleType) => {
      return (
        allowEdit.value &&
        (role != "OWNER" || dataSource.value[0].list.length > 1)
      );
    };

    const showUpgradeInfo = computed(() => {
      return !hasAdminFeature.value && isOwner(currentUser.value.role);
    });

    const changeRole = (id: MemberId, role: RoleType) => {
      const memberPatch: MemberPatch = {
        updaterId: currentUser.value.id,
        role,
      };
      store
        .dispatch("member/patchMember", {
          id,
          memberPatch,
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const deleteRole = (id: MemberId) => {
      store.dispatch("member/deleteMemberById", id).catch((error) => {
        console.log(error);
      });
    };

    return {
      state,
      currentUser,
      hasAdminFeature,
      showUpgradeInfo,
      columnList,
      dataSource,
      allowEdit,
      allowChangeRole,
      changeRole,
      deleteRole,
    };
  },
};
</script>
