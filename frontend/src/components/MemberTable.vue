<template>
  <p class="text-xl font-bold leading-7 text-main">Current members</p>
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
    <template v-slot:body="{ rowData: roleMapping }">
      <BBTableCell :leftPadding="4" class="table-cell">
        <div class="flex flex-row items-center space-x-2">
          <template v-if="'INVITED' == roleMapping.principal.status">
            <span
              class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-main text-main-text"
            >
              Invited
            </span>
            <span class="textlabel">
              {{ roleMapping.principal.email }}
            </span>
          </template>
          <template v-else>
            <BBAvatar :username="roleMapping.principal.name" />
            <div class="flex flex-col">
              <div class="flex flex-row items-center space-x-2">
                <router-link
                  :to="`/u/${roleMapping.principal.id}`"
                  class="normal-link"
                  >{{ roleMapping.principal.name }}
                </router-link>
                <span
                  v-if="currentUser.id == roleMapping.principal.id"
                  class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
                >
                  You
                </span>
              </div>
              <span class="textlabel">
                {{ roleMapping.principal.email }}
              </span>
            </div>
          </template>
        </div>
      </BBTableCell>
      <BBTableCell class="">
        <RoleSelect
          :selectedRole="roleMapping.role"
          :disabled="!allowEdit"
          @change-role="
            (role) => {
              changeRole(roleMapping.id, role);
            }
          "
        />
      </BBTableCell>
      <BBTableCell class="table-cell">
        <div class="flex flex-row items-center space-x-1">
          <span>
            {{ humanizeTs(roleMapping.lastUpdatedTs) }}
          </span>
          <router-link :to="`/u/${roleMapping.updater.id}`" class="normal-link"
            >by {{ roleMapping.updater.name }}
          </router-link>
        </div>
      </BBTableCell>
      <BBTableCell>
        <button
          v-if="allowEdit"
          class="btn-icon"
          @click.prevent="deleteRole(roleMapping.id)"
        >
          <svg
            class="w-4 h-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
            ></path>
          </svg>
        </button>
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { computed, reactive, watchEffect } from "vue";
import { useStore } from "vuex";
import RoleSelect from "../components/RoleSelect.vue";
import { RoleMapping, RoleMappingId, RoleType } from "../types";
import { BBTableColumn } from "../bbkit/types";

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
      const ownerList = [];
      const dbaList = [];
      const developerList = [];
      for (const roleMapping of store.getters[
        "roleMapping/roleMappingList"
      ]()) {
        if (roleMapping.role === "OWNER") {
          ownerList.push(roleMapping);
        } else if (roleMapping.role === "DBA") {
          dbaList.push(roleMapping);
        } else if (roleMapping.role === "DEVELOPER") {
          developerList.push(roleMapping);
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
      const myRoleMapping = store.getters[
        "roleMapping/roleMappingByPrincipalId"
      ](currentUser.value.id);
      if (myRoleMapping.role != "OWNER") {
        return false;
      }
      return true;
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
