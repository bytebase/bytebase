<template>
  <div>
    <div class="flex justify-between items-center">
      <div class="inline-flex items-center space-x-2">
        <h2 class="text-lg leading-7 font-medium text-main">User list</h2>
        <BBButtonAdd v-if="allowEdit" @add="state.showCreateModal = true" />
      </div>
      <BBTableSearch
        ref="searchField"
        :placeholder="'Search user'"
        @change-text="(text) => changeSearchText(text)"
      />
    </div>
    <BBTable
      class="mt-2"
      :columnList="columnList"
      :dataSource="dataSource.memberList"
      :showHeader="true"
      :rowClickable="false"
    >
      <template v-slot:header>
        <BBTableHeaderCell
          :leftPadding="4"
          class="w-auto table-cell"
          :title="columnList[0].title"
        />
        <BBTableHeaderCell
          class="w-8 table-cell"
          :title="columnList[1].title"
        />
        <BBTableHeaderCell
          class="w-8 table-cell"
          :title="columnList[2].title"
        />
        <BBTableHeaderCell
          v-if="allowEdit"
          class="w-8 table-cell"
          :title="columnList[3].title"
        />
      </template>
      <template v-slot:body="{ rowData: member }">
        <BBTableCell :leftPadding="4" class="table-cell">
          <div class="flex flex-row items-center space-x-2">
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
          </div>
        </BBTableCell>
        <BBTableCell>
          <router-link
            v-if="member.issueID"
            :to="`/issue/${member.issueID}`"
            class="normal-link"
            >issue/{{ member.issueID }}
          </router-link>
        </BBTableCell>
        <BBTableCell>
          {{ humanizeTs(member.createdTs) }}
        </BBTableCell>
        <BBTableCell>
          <BBButtonConfirm
            v-if="allowEdit"
            :requireConfirm="true"
            :okText="'Revoke'"
            :confirmTitle="`Are you sure to revoke '${member.principal.name}' access from '${dataSource.name}'?`"
            @confirm="deleteMember(member)"
          />
        </BBTableCell>
      </template>
    </BBTable>
    <BBModal
      v-if="state.showCreateModal"
      :title="'Grant data source'"
      @close="state.showCreateModal = false"
    >
      <DataSourceMemberForm
        :dataSource="dataSource"
        @submit="state.showCreateModal = false"
        @cancel="state.showCreateModal = false"
      />
    </BBModal>
  </div>
</template>

<script lang="ts">
import { computed, reactive, PropType } from "vue";
import { useStore } from "vuex";
import DataSourceMemberForm from "../components/DataSourceMemberForm.vue";
import PrincipalAvatar from "../components/PrincipalAvatar.vue";
import { DataSource, DataSourceMember } from "../types";

const columnList = [
  {
    title: "User",
  },
  {
    title: "Requested issue",
  },
  {
    title: "Added time",
  },
  {},
];

interface LocalState {
  searchText: string;
  showCreateModal: boolean;
}

export default {
  name: "DataSourceMemberTable",
  components: { DataSourceMemberForm, PrincipalAvatar },
  props: {
    dataSource: {
      required: true,
      type: Object as PropType<DataSource>,
    },
    allowEdit: {
      required: true,
      type: Boolean,
    },
  },
  setup(props, ctx) {
    const store = useStore();

    const state = reactive<LocalState>({
      searchText: "",
      showCreateModal: false,
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const deleteMember = (member: DataSourceMember) => {
      store
        .dispatch("dataSource/deleteDataSourceMemberByMemberID", {
          databaseID: props.dataSource.database.id,
          dataSourceID: props.dataSource.id,
          memberID: member.principal.id,
        })
        .then(() => {
          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "INFO",
            title: `Successfully revoked '${member.principal.name}' access from '${props.dataSource.name}'.`,
          });
        });
    };

    const changeSearchText = (searchText: string) => {
      state.searchText = searchText;
    };

    return {
      state,
      columnList,
      currentUser,
      deleteMember,
      changeSearchText,
    };
  },
};
</script>
