<template>
  <div class="space-y-4">
    <div class="space-y-2">
      <div class="flex flex-row justify-between py-0.5 select-none space-x-4">
        <div class="w-64">
          <PrincipalSelect
            id="user"
            name="user"
            :required="false"
            :placeholder="'Select user'"
            :selectedId="state.principalId"
            @select-principal-id="
              (principalId) => {
                state.principalId = principalId;
                clearValidationError();
                validateMember();
              }
            "
          />
        </div>
        <div v-if="hasAdminFeature" class="flex flex-row radio-set">
          <div class="radio">
            <input
              :name="`member_role`"
              tabindex="-1"
              type="radio"
              class="btn"
              value="OWNER"
              v-model="state.role"
            />
            <label class="label"> Owner </label>
          </div>
          <div class="radio">
            <input
              :name="`member_role`"
              tabindex="-1"
              type="radio"
              class="btn"
              value="DEVELOPER"
              v-model="state.role"
            />
            <label class="label"> Developer </label>
          </div>
        </div>
      </div>
    </div>

    <div class="flex justify-between">
      <span class="flex items-center">
        <p v-if="state.error" class="text-sm text-error" id="state-error">
          {{ state.error }}
        </p>
      </span>

      <button
        type="button"
        class="btn-primary"
        :disabled="!hasValidMember"
        @click.prevent="addMember"
      >
        <svg
          class="mr-2 w-5 h-5"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M18 9v3m0 0v3m0-3h3m-3 0h-3m-2-5a4 4 0 11-8 0 4 4 0 018 0zM3 20a6 6 0 0112 0v1H3v-1z"
          ></path>
        </svg>
        Add member
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, PropType, reactive } from "vue";
import { useStore } from "vuex";
import PrincipalSelect from "../components/PrincipalSelect.vue";
import {
  PrincipalId,
  Project,
  ProjectMember,
  ProjectMemberNew,
  ProjectRoleType,
  UNKNOWN_ID,
} from "../types";

interface LocalState {
  principalId: PrincipalId;
  role: ProjectRoleType;
  error: string;
}

export default {
  name: "ProjectMemberInvite",
  components: { PrincipalSelect },
  props: {
    project: {
      required: true,
      type: Object as PropType<Project>,
    },
  },
  setup(props, ctx) {
    const store = useStore();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const hasAdminFeature = computed(() =>
      store.getters["plan/feature"]("bytebase.admin")
    );

    const state = reactive<LocalState>({
      principalId: UNKNOWN_ID,
      role: "DEVELOPER",
      error: "",
    });

    const hasValidMember = computed(() => {
      return (
        state.principalId != UNKNOWN_ID && validateInviteInternal().length == 0
      );
    });

    const validateInviteInternal = (): string => {
      if (state.principalId != UNKNOWN_ID) {
        if (
          props.project.memberList.find((item: ProjectMember) => {
            return item.principal.id == state.principalId;
          })
        ) {
          return "Already a project member";
        }
      }
      return "";
    };

    const validateMember = () => {
      state.error = validateInviteInternal();
    };

    const clearValidationError = () => {
      state.error = "";
    };

    const addMember = () => {
      // If admin feature is NOT enabled, then we set every member to OWNER role.
      const projectMember: ProjectMemberNew = {
        principalId: state.principalId,
        role: hasAdminFeature.value ? state.role : "OWNER",
        creatorId: currentUser.value.id,
      };
      store
        .dispatch("project/createdMember", {
          projectId: props.project.id,
          projectMember,
        })
        .catch((err) => {
          console.error(err);
        });
      state.principalId = UNKNOWN_ID;
      state.role = "DEVELOPER";
      state.error = "";
    };

    return {
      state,
      hasAdminFeature,
      validateMember,
      clearValidationError,
      hasValidMember,
      addMember,
    };
  },
};
</script>
