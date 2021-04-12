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
    <template v-slot:body="{ rowData: roleMappingUI }">
      <BBTableCell :leftPadding="4" class="table-cell">
        <div class="flex flex-row items-center space-x-2">
          <template v-if="'INVITED' == roleMappingUI.principal.status">
            <span
              class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-main text-main-text"
            >
              Invited
            </span>
            <span class="textlabel">
              {{ roleMappingUI.principal.email }}
            </span>
          </template>
          <template v-else>
            <BBAvatar :username="roleMappingUI.principal.name" />
            <div class="flex flex-col">
              <div class="flex flex-row items-center space-x-2">
                <router-link
                  :to="`/u/${roleMappingUI.principal.id}`"
                  class="normal-link"
                  >{{ roleMappingUI.principal.name }}
                </router-link>
                <span
                  v-if="currentUser.id == roleMappingUI.principal.id"
                  class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
                >
                  You
                </span>
              </div>
              <span class="textlabel">
                {{ roleMappingUI.principal.email }}
              </span>
            </div>
          </template>
        </div>
      </BBTableCell>
      <BBTableCell v-if="hasAdminFeature" class="">
        <RoleSelect
          :selectedRole="roleMappingUI.role"
          :disabled="!allowChangeRole(roleMappingUI.role)"
          @change-role="
            (role) => {
              changeRole(roleMappingUI.id, role);
            }
          "
        />
      </BBTableCell>
      <BBTableCell class="table-cell">
        <div class="flex flex-row items-center space-x-1">
          <span>
            {{ humanizeTs(roleMappingUI.lastUpdatedTs) }}
          </span>
          <span>by</span>
          <router-link
            :to="`/u/${roleMappingUI.updater.id}`"
            class="normal-link"
            >{{ roleMappingUI.updater.name }}
          </router-link>
        </div>
      </BBTableCell>
      <BBTableCell>
        <BBButtonConfirm
          v-if="allowChangeRole(roleMappingUI.role)"
          :requireConfirm="true"
          :okText="'Revoke'"
          :confirmTitle="`Are you sure to revoke '${roleMappingUI.role}' from '${roleMappingUI.principal.name}'`"
          @confirm="deleteRole(roleMappingUI.id)"
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
import { Principal, RoleMapping, RoleMappingId, RoleType } from "../types";
import { BBTableColumn, BBTableSectionDataSource } from "../bbkit/types";
import { isOwner } from "../utils";

type RoleMappingUI = RoleMapping & {
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
      store.getters["plan/feature"]("bytebase.admin")
    );

    const state = reactive<LocalState>({});

    const prepareRoleMappingList = () => {
      store.dispatch("roleMapping/fetchRoleMappingList").catch((error) => {
        console.log(error);
      });
    };

    watchEffect(prepareRoleMappingList);

    const dataSource = computed(
      (): BBTableSectionDataSource<RoleMappingUI>[] => {
        const ownerList: RoleMappingUI[] = [];
        const dbaList: RoleMappingUI[] = [];
        const developerList: RoleMappingUI[] = [];
        for (const roleMapping of store.getters[
          "roleMapping/roleMappingList"
        ]()) {
          const roleMappingUI = {
            ...roleMapping,
            principal: store.getters["principal/principalById"](
              roleMapping.principalId
            ),
            updater: store.getters["principal/principalById"](
              roleMapping.updaterId
            ),
          };

          if (roleMappingUI.role == "OWNER") {
            ownerList.push(roleMappingUI);
          }

          if (roleMappingUI.role == "DBA") {
            dbaList.push(roleMappingUI);
          }

          if (roleMappingUI.role == "DEVELOPER") {
            developerList.push(roleMappingUI);
          }
        }

        const dataSource: BBTableSectionDataSource<RoleMappingUI>[] = [];
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
      }
    );

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

    const changeRole = (id: RoleMappingId, role: RoleType) => {
      store
        .dispatch("roleMapping/patchRoleMapping", {
          id,
          role,
          updaterId: currentUser.value.id,
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const deleteRole = (id: RoleMappingId) => {
      store.dispatch("roleMapping/deleteRoleMappingById", id).catch((error) => {
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
