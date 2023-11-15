<template>
  <div class="flex flex-col gap-y-2">
    <IssueSearch
      v-model:params="state.params"
      :show-feature-attention="true"
      :component-props="{
        searchbox: { autofocus },
      }"
      class="px-4 py-2 gap-y-2"
    />

    <div>
      <PagedIssueTableV1
        session-key="issue-dashboard"
        :issue-filter="buildIssueFilterBySearchParams(state.params)"
        :ui-issue-filter="buildUIIssueFilterBySearchParams(state.params)"
        :page-size="50"
      >
        <template #table="{ issueList, loading }">
          <IssueTableV1
            class="border-x-0"
            :show-placeholder="!loading"
            :issue-list="issueList"
            :highlight-text="state.params.query"
          />
        </template>
      </PagedIssueTableV1>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { reactive, computed, watchEffect, watch } from "vue";
import { useRoute } from "vue-router";
import { useRouter } from "vue-router";
import IssueSearch from "@/components/IssueV1/components/IssueSearch";
import IssueTableV1 from "@/components/IssueV1/components/IssueTableV1.vue";
import PagedIssueTableV1 from "@/components/IssueV1/components/PagedIssueTableV1.vue";
import { useProjectV1Store } from "@/store";
import {
  SearchParams,
  buildSearchParamsBySearchText,
  buildIssueFilterBySearchParams,
  buildUIIssueFilterBySearchParams,
  getValueFromSearchParams,
  buildSearchTextBySearchParams,
  maybeApplyDefaultTsRange,
} from "@/utils";

interface LocalState {
  params: SearchParams;
}

const route = useRoute();
const router = useRouter();

const autofocus = computed((): boolean => {
  return !!route.query.autofocus;
});

const initializeSearchParamsFromQuery = (): SearchParams => {
  const { qs } = route.query;
  const params: SearchParams =
    typeof qs === "string" && qs.length > 0
      ? buildSearchParamsBySearchText(qs)
      : {
          query: "",
          scopes: [{ id: "status", value: "OPEN" }],
        };
  maybeApplyDefaultTsRange(params, "created", true /* mutate */);

  return params;
};

const paramsFromQuery = initializeSearchParamsFromQuery();
const state = reactive<LocalState>({
  params: paramsFromQuery,
});

watchEffect(() => {
  const project = getValueFromSearchParams(
    state.params,
    "project",
    "projects/"
  );
  if (project) {
    useProjectV1Store().getOrFetchProjectByName(project);
  }
});

watch(
  () => state.params,
  () => {
    router.replace({
      query: {
        qs: buildSearchTextBySearchParams(state.params),
      },
    });
  },
  { deep: true }
);
</script>
