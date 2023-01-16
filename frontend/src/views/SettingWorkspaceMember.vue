<template>
  <div class="w-full overflow-x-hidden">
    <div v-if="allowAddOrInvite" class="w-full flex justify-center mb-6">
      <MemberAddOrInvite />
    </div>
    <FeatureAttention
      v-if="!hasRBACFeature"
      custom-class="my-5"
      feature="bb.feature.rbac"
      :description="$t('subscription.features.bb-feature-rbac.desc')"
    />
    <div class="flex justify-between items-center">
      <div class="flex-1 flex space-x-2">
        <p class="text-lg font-medium leading-7 text-main">
          <span>{{ $t("settings.members.active") }}</span>
          <span class="ml-1 font-normal text-control-light">
            ({{ activeMemberList.length }})
          </span>
        </p>
        <div
          v-if="showUpgradeInfo"
          class="flex flex-row items-center space-x-1"
        >
          <heroicons-solid:sparkles class="w-6 h-6 text-accent" />
          <router-link to="/setting/subscription" class="text-lg accent-link">{{
            $t("settings.members.upgrade")
          }}</router-link>
        </div>
      </div>

      <div>
        <BBTableSearch
          :value="state.activeMemberFilterText"
          @change-text="(text: string) => state.activeMemberFilterText = text"
        />
      </div>
    </div>
    <MemberTable :member-list="activeMemberList" />
    <div
      v-if="inactiveMemberList.length > 0 || state.inactiveMemberFilterText"
      class="mt-8"
    >
      <div class="flex justify-between items-center">
        <p class="text-lg font-medium leading-7 text-control-light">
          <span>{{ $t("settings.members.inactive") }}</span>
          <span class="ml-1 font-normal text-control-light">
            ({{ inactiveMemberList.length }})
          </span>
        </p>

        <div>
          <BBTableSearch
            :value="state.inactiveMemberFilterText"
            @change-text="(text: string) => state.inactiveMemberFilterText = text"
          />
        </div>
      </div>
      <MemberTable :member-list="inactiveMemberList" />
    </div>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, reactive } from "vue";
import MemberAddOrInvite from "../components/MemberAddOrInvite.vue";
import MemberTable from "../components/MemberTable.vue";
import { hasWorkspacePermission } from "../utils";
import { Member, SYSTEM_BOT_ID, unknown } from "../types";
import {
  featureToRef,
  useCurrentUser,
  useMemberList,
  usePrincipalStore,
} from "@/store";

type LocalState = {
  activeMemberFilterText: string;
  inactiveMemberFilterText: string;
};

export default defineComponent({
  name: "SettingWorkspaceMember",
  components: { MemberAddOrInvite, MemberTable },
  setup() {
    const state = reactive<LocalState>({
      activeMemberFilterText: "",
      inactiveMemberFilterText: "",
    });

    const currentUser = useCurrentUser();

    const hasRBACFeature = featureToRef("bb.feature.rbac");

    const memberList = useMemberList();

    const activeMemberList = computed(() => {
      const systemBotMember: Member = {
        ...unknown("MEMBER"),
        id: SYSTEM_BOT_ID,
        role: "OWNER",
        principal: usePrincipalStore().principalById(SYSTEM_BOT_ID),
      };
      let list = [
        systemBotMember,
        ...memberList.value.filter(
          (member: Member) => member.rowStatus == "NORMAL"
        ),
      ];
      const keyword = state.activeMemberFilterText.trim().toLowerCase();
      if (keyword) {
        list = list.filter((member) =>
          member.principal.name.toLowerCase().includes(keyword)
        );
      }

      return list;
    });

    const inactiveMemberList = computed(() => {
      let list = memberList.value.filter(
        (member: Member) => member.rowStatus == "ARCHIVED"
      );
      const keyword = state.inactiveMemberFilterText.trim().toLowerCase();
      if (keyword) {
        list = list.filter((member) =>
          member.principal.name.toLowerCase().includes(keyword)
        );
      }

      return list;
    });

    const allowAddOrInvite = computed(() => {
      // TODO(tianzhou): Implement invite mode for DBA and developer
      // If current user has manage user permission, MemberAddOrInvite is in Add mode.
      // Otherwise, MemberAddOrInvite is in Invite mode.
      return hasWorkspacePermission(
        "bb.permission.workspace.manage-member",
        currentUser.value.role
      );
    });

    const showUpgradeInfo = computed(() => {
      return (
        !hasRBACFeature.value &&
        hasWorkspacePermission(
          "bb.permission.workspace.manage-general",
          currentUser.value.role
        )
      );
    });

    return {
      state,
      activeMemberList,
      inactiveMemberList,
      allowAddOrInvite,
      showUpgradeInfo,
      hasRBACFeature,
    };
  },
});
</script>
