<template>
  <div>
    <div class="flex justify-between items-center">
      <div class="inline-flex items-center space-x-2">
        <h2 class="text-lg leading-7 font-medium text-main">
          {{ $t("datasource.user-list") }}
        </h2>
        <BBButtonAdd v-if="allowEdit" @add="state.showCreateModal = true" />
      </div>
      <BBTableSearch
        ref="searchField"
        :placeholder="$t('datasource.search-user')"
        @change-text="(text) => changeSearchText(text)"
      />
    </div>
    <BBTable
      class="mt-2"
      :column-list="columnList"
      :data-source="dataSource.memberList"
      :show-header="true"
      :row-clickable="false"
    >
      <template #header>
        <BBTableHeaderCell
          :left-padding="4"
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
      <template #body="{ rowData: member }">
        <BBTableCell :left-padding="4" class="table-cell">
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
                  class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
                >
                  {{ $t("settings.members.yourself") }}
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
            v-if="member.issueId"
            :to="`/issue/${member.issueId}`"
            class="normal-link"
            >issue/{{ member.issueId }}
          </router-link>
        </BBTableCell>
        <BBTableCell>
          {{ humanizeTs(member.createdTs) }}
        </BBTableCell>
        <BBTableCell>
          <BBButtonConfirm
            v-if="allowEdit"
            :require-confirm="true"
            :ok-text="'Revoke'"
            :confirm-title="
              $t('datasource.revoke-access', [
                member.principal.name,
                dataSource.name,
              ])
            "
            @confirm="deleteMember(member)"
          />
        </BBTableCell>
      </template>
    </BBTable>
    <BBModal
      v-if="state.showCreateModal"
      :title="$t('datasource.grant-data-source')"
      @close="state.showCreateModal = false"
    >
      <DataSourceMemberForm
        :data-source="dataSource"
        @submit="state.showCreateModal = false"
        @cancel="state.showCreateModal = false"
      />
    </BBModal>
  </div>
</template>

<script lang="ts">
import { reactive, PropType, defineComponent } from "vue";
import { useStore } from "vuex";
import DataSourceMemberForm from "../components/DataSourceMemberForm.vue";
import PrincipalAvatar from "../components/PrincipalAvatar.vue";
import { DataSource, DataSourceMember } from "../types";
import { useI18n } from "vue-i18n";
import { pushNotification, useCurrentUser } from "@/store";

interface LocalState {
  searchText: string;
  showCreateModal: boolean;
}

export default defineComponent({
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
  setup(props) {
    const store = useStore();

    const state = reactive<LocalState>({
      searchText: "",
      showCreateModal: false,
    });

    const { t } = useI18n();

    const currentUser = useCurrentUser();

    const columnList = [
      {
        title: t("common.user"),
      },
      {
        title: t("datasource.requested-issue"),
      },
      {
        title: t("common.added-time"),
      },
      {},
    ];

    const deleteMember = (member: DataSourceMember) => {
      // TODO (yw): there is no action named deleteDataSourceMemberByMemberId
      store
        .dispatch("dataSource/deleteDataSourceMemberByMemberId", {
          databaseId: props.dataSource.database.id,
          dataSourceId: props.dataSource.id,
          memberId: member.principal.id,
        })
        .then(() => {
          pushNotification({
            module: "bytebase",
            style: "INFO",
            title: t(
              "datasource.successfully-revoked-member-principal-name-access-from-props-datasource-name",
              [member.principal.name, props.dataSource.name]
            ),
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
});
</script>
