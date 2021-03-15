<template>
  <h2 class="text-xl leading-7 font-bold text-main">User list</h2>
  <BBTable
    class="mt-2"
    :columnList="columnList"
    :dataSource="memberList"
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
      <BBTableHeaderCell class="w-8 table-cell" :title="columnList[2].title" />
    </template>
    <template v-slot:body="{ rowData: member }">
      <BBTableCell :leftPadding="4" class="table-cell">
        <div class="flex flex-row items-center space-x-2">
          <BBAvatar :username="member.principal.name" />
          <div class="flex flex-col">
            <div class="flex flex-row items-center space-x-2">
              <router-link :to="`/u/${member.principal.id}`" class="normal-link"
                >{{ member.principal.name }}
              </router-link>
              <span
                v-if="currentUser.id == member.principal.id"
                class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
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
        <router-link :to="`/task/${member.taskId}`" class="normal-link"
          >task/{{ member.taskId }}
        </router-link>
      </BBTableCell>
      <BBTableCell>
        {{ humanizeTs(member.createdTs) }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { computed, watchEffect } from "vue";
import { useStore } from "vuex";

const columnList = [
  {
    title: "User",
  },
  {
    title: "Requested task",
  },
  {
    title: "Added time",
  },
];

export default {
  name: "DataSourceMemberTable",
  components: {},
  props: {
    instanceId: {
      required: true,
      type: String,
    },
    dataSourceId: {
      required: true,
      type: String,
    },
  },
  setup(props, ctx) {
    const store = useStore();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const prepareMemberList = () => {
      store
        .dispatch("dataSource/fetchMemberListById", {
          instanceId: props.instanceId,
          dataSourceId: props.dataSourceId,
        })
        .catch((error) => {
          console.log(error);
        });
    };

    watchEffect(prepareMemberList);

    const memberList = computed(() =>
      store.getters["dataSource/memberListById"](props.dataSourceId)
    );

    return {
      columnList,
      currentUser,
      memberList,
    };
  },
};
</script>
