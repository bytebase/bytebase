<template>
  <div class="flex flex-col">
    <div class="px-4 py-2 flex justify-between items-center">
      <!-- eslint-disable vue/attribute-hyphenation -->
      <EnvironmentTabFilter
        :selectedId="selectedEnvironment?.id"
        @select-environment="selectEnvironment"
      />
      <BBTableSearch
        ref="searchField"
        :placeholder="$t('issue.search-issue-name')"
        @change-text="(text: string) => changeSearchText(text)"
      />
    </div>

    <!-- show OPEN Assigned issues with pageSize=10 -->
    <PagedIssueTable
      session-key="home-assigned"
      :issue-find="{
        statusList: ['OPEN'],
        assigneeId: currentUser.id,
      }"
      :page-size="OPEN_ISSUE_LIST_PAGE_SIZE"
    >
      <template #table="{ issueList, loading }">
        <IssueTable
          :left-bordered="false"
          :right-bordered="false"
          :show-placeholder="!loading"
          :title="$t('common.assigned')"
          :issue-list="issueList.filter(keywordAndEnvironmentFilter)"
        />
      </template>
    </PagedIssueTable>

    <!-- show OPEN Created issues with pageSize=10 -->
    <PagedIssueTable
      session-key="home-created"
      :issue-find="{
        statusList: ['OPEN'],
        creatorId: currentUser.id,
      }"
      :page-size="OPEN_ISSUE_LIST_PAGE_SIZE"
    >
      <template #table="{ issueList, loading }">
        <IssueTable
          class="-mt-px"
          :left-bordered="false"
          :right-bordered="false"
          :show-placeholder="!loading"
          :title="$t('common.created')"
          :issue-list="issueList.filter(keywordAndEnvironmentFilter)"
        />
      </template>
    </PagedIssueTable>

    <!-- show OPEN Subscribed issues with pageSize=10 -->
    <PagedIssueTable
      session-key="home-subscribed"
      :issue-find="{
        statusList: ['OPEN'],
        subscriberId: currentUser.id,
      }"
      :page-size="OPEN_ISSUE_LIST_PAGE_SIZE"
    >
      <template #table="{ issueList, loading }">
        <IssueTable
          class="-mt-px"
          :left-bordered="false"
          :right-bordered="false"
          :show-placeholder="!loading"
          :title="$t('common.subscribed')"
          :issue-list="issueList.filter(keywordAndEnvironmentFilter)"
        />
      </template>
    </PagedIssueTable>

    <!-- show the first 5 DONE or CANCELED issues -->
    <!-- But won't show "Load more", since we have a "View all closed" link below -->
    <PagedIssueTable
      session-key="home-closed"
      :issue-find="{
        statusList: ['DONE', 'CANCELED'],
        principalId: currentUser.id,
      }"
      :page-size="MAX_CLOSED_ISSUE"
      :hide-load-more="true"
    >
      <template #table="{ issueList, loading }">
        <IssueTable
          class="-mt-px"
          :left-bordered="false"
          :right-bordered="false"
          :show-placeholder="!loading"
          :title="$t('project.overview.recently-closed')"
          :issue-list="issueList.filter(keywordAndEnvironmentFilter)"
        />
      </template>
    </PagedIssueTable>
  </div>
  <div class="w-full flex justify-end mt-2 px-4">
    <router-link to="/issue?status=closed" class="normal-link">
      {{ $t("project.overview.view-all-closed") }}
    </router-link>
  </div>

  <BBModal
    :title="
      $t('subscription.trial-start-modal.title', {
        plan: $t(
          `subscription.plan.${planTypeToString(
            subscriptionStore.subscription.plan
          ).toLowerCase()}.title`
        ),
      })
    "
    @close="onTrialingModalClose"
    v-if="state.showTrialStartModal && subscriptionStore.subscription"
  >
    <div class="min-w-0 md:min-w-400 max-w-2xl">
      <div class="flex justify-center items-center">
        <img :src="planImage" class="w-56 p-4" />
        <div class="text-lg space-y-2">
          <p>
            <i18n-t keypath="subscription.trial-start-modal.content">
              <template #plan>
                <strong>
                  {{
                    $t(
                      `subscription.plan.${planTypeToString(
                        subscriptionStore.subscription.plan
                      ).toLowerCase()}.title`
                    )
                  }}
                </strong>
              </template>
              <template #date>
                <strong>{{ subscriptionStore.expireAt }}</strong>
              </template>
            </i18n-t>
          </p>
          <p>
            <i18n-t keypath="subscription.trial-start-modal.subscription">
              <template #page>
                <router-link
                  to="/setting/subscription"
                  class="normal-link"
                  exact-active-class=""
                >
                  {{ $t("subscription.trial-start-modal.subscription-page") }}
                </router-link>
              </template>
            </i18n-t>
          </p>
        </div>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { onMounted, reactive, ref, computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import EnvironmentTabFilter from "../components/EnvironmentTabFilter.vue";
import { IssueTable } from "../components/Issue";
import { activeEnvironment } from "../utils";
import { Environment, Issue, planTypeToString, PlanType } from "../types";
import {
  useCurrentUser,
  useEnvironmentStore,
  useSubscriptionStore,
  useOnboardingStateStore,
} from "@/store";
import PagedIssueTable from "@/components/Issue/PagedIssueTable.vue";

interface LocalState {
  searchText: string;
  showTrialStartModal: boolean;
}

const OPEN_ISSUE_LIST_PAGE_SIZE = 10;
const MAX_CLOSED_ISSUE = 5;

const searchField = ref();

const environmentStore = useEnvironmentStore();
const subscriptionStore = useSubscriptionStore();
const onboardingStateStore = useOnboardingStateStore();
const router = useRouter();
const route = useRoute();

const state = reactive<LocalState>({
  searchText: "",
  showTrialStartModal: onboardingStateStore.getStateByKey(
    "show-trialing-modal"
  ),
});

const currentUser = useCurrentUser();

const onTrialingModalClose = () => {
  state.showTrialStartModal = false;
  onboardingStateStore.consume("show-trialing-modal");
};

const planImage = computed(() => {
  return new URL(
    `../assets/plan-${planTypeToString(
      subscriptionStore.subscription?.plan ?? PlanType.FREE
    ).toLowerCase()}.png`,
    import.meta.url
  ).href;
});

const selectedEnvironment = computed(() => {
  const { environment } = route.query;
  return environment
    ? environmentStore.getEnvironmentById(parseInt(environment as string, 10))
    : undefined;
});

const keywordAndEnvironmentFilter = (issue: Issue) => {
  if (selectedEnvironment.value) {
    if (activeEnvironment(issue.pipeline).id !== selectedEnvironment.value.id) {
      return false;
    }
  }
  const keyword = state.searchText.trim();
  if (keyword) {
    if (!issue.name.toLowerCase().includes(keyword)) {
      return false;
    }
  }
  return true;
};
const selectEnvironment = (environment: Environment) => {
  if (environment) {
    router.replace({
      name: "workspace.home",
      query: {
        ...route.query,
        environment: environment.id,
      },
    });
  } else {
    router.replace({
      name: "workspace.home",
      query: {
        ...route.query,
        environment: undefined,
      },
    });
  }
};

const changeSearchText = (searchText: string) => {
  state.searchText = searchText;
};

onMounted(() => {
  // Focus on the internal search field when mounted
  searchField.value.$el.querySelector("#search").focus();
});
</script>
