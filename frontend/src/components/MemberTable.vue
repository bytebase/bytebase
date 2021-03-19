<template>
  <p class="text-lg font-medium leading-7 text-main">Current members</p>
  <BBTable
    class="mt-2"
    :columnList="columnList"
    :sectionDataSource="dataSource"
    :showHeader="true"
    :rowClickable="false"
  >
    <template v-slot:header>
      <BBTableHeaderCell
        :leftPadding="4"
        class="w-auto table-cell"
        :title="columnList[0].title"
      />
      <BBTableHeaderCell class="w-8 table-cell" :title="columnList[1].title" />
      <BBTableHeaderCell class="w-72 table-cell" :title="columnList[2].title" />
      <BBTableHeaderCell
        class="w-auto table-cell"
        :title="columnList[3].title"
      />
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
      <BBTableCell class="">
        <RoleSelect
          :selectedRole="roleMappingUI.role"
          :disabled="!allowEdit"
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
        <BBButtonTrash
          v-if="allowEdit"
          :requireConfirm="true"
          :okText="'Revoke'"
          :confirmTitle="`Are you sure to revoke '${roleMappingUI.role}' from '${roleMappingUI.principal.name}'`"
          @confirm="deleteRole(roleMappingUI.id)"
        />
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { computed, reactive, watchEffect } from "vue";
import { useStore } from "vuex";
import RoleSelect from "../components/RoleSelect.vue";
import { Principal, RoleMapping, RoleMappingId, RoleType } from "../types";
import { BBTableColumn } from "../bbkit/types";

type RoleMappingUI = RoleMapping & {
  principal: Principal;
  updater: Principal;
};

const columnList: BBTableColumn[] = [
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
];

interface LocalState {}

export default {
  name: "MemberTable",
  components: { RoleSelect },
  props: {},
  setup(props, ctx) {
    const store = useStore();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const state = reactive<LocalState>({});

    const prepareRoleMappingList = () => {
      store.dispatch("roleMapping/fetchRoleMappingList").catch((error) => {
        console.log(error);
      });
    };

    watchEffect(prepareRoleMappingList);

    const dataSource = computed(() => {
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
        if (roleMappingUI.role === "OWNER") {
          ownerList.push(roleMappingUI);
        } else if (roleMappingUI.role === "DBA") {
          dbaList.push(roleMappingUI);
        } else if (roleMappingUI.role === "DEVELOPER") {
          developerList.push(roleMappingUI);
        }
      }
      const dataSource = [];
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
      return currentUser.value.role == "OWNER";
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
      columnList,
      dataSource,
      allowEdit,
      changeRole,
      deleteRole,
    };
  },
};
</script>
