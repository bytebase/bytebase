import { orderBy } from "lodash-es";
import type { Ref, RenderFunction, VNode } from "vue";
import { computed, h, ref } from "vue";
import { useI18n } from "vue-i18n";
import BBAvatar from "@/bbkit/BBAvatar.vue";
import GitIcon from "@/components/GitIcon.vue";
import SystemBotTag from "@/components/misc/SystemBotTag.vue";
import YouTag from "@/components/misc/YouTag.vue";
import { ProjectV1Name } from "@/components/v2";
import ALL_METHODS from "@/grpcweb/methods";
import { useCurrentUserV1, useUserStore, useProjectV1List } from "@/store";
import { SYSTEM_BOT_EMAIL } from "@/types";
import { AuditLog_Severity } from "@/types/proto/v1/audit_log_service";
import { Workflow } from "@/types/proto/v1/project_service";
import type { SearchParams, SearchScopeId } from "@/utils";
import { extractProjectResourceName } from "@/utils";

export type ScopeOption = {
  id: SearchScopeId;
  title: string;
  description: string;
  options: ValueOption[];
};

export type ValueOption = {
  value: string;
  keywords: string[];
  custom?: boolean;
  render: RenderFunction;
};

export const useSearchScopeOptions = (
  params: Ref<SearchParams>,
  supportOptionIdList: Ref<SearchScopeId[]>
) => {
  const { t } = useI18n();
  const me = useCurrentUserV1();
  const userStore = useUserStore();
  const { projectList } = useProjectV1List();

  const principalSearchValueOptions = computed(() => {
    // Put "you" to the top
    const sortedUsers = orderBy(
      userStore.activeUserList,
      (user) => (user.name === me.value.name ? -1 : 1),
      "asc"
    );
    return sortedUsers.map<ValueOption>((user) => {
      return {
        value: user.email,
        keywords: [user.email, user.title],
        render: () => {
          const children = [
            h(BBAvatar, { size: "TINY", username: user.title }),
            renderSpan(user.title),
          ];
          if (user.name === me.value.name) {
            children.push(h(YouTag));
          }
          if (user.email === SYSTEM_BOT_EMAIL) {
            children.push(h(SystemBotTag));
          }
          return h("div", { class: "flex items-center gap-x-1" }, children);
        },
      };
    });
  });

  // fullScopeOptions provides full search scopes and options.
  // we need this as the source of truth.
  const fullScopeOptions = computed((): ScopeOption[] => {
    const scopes: ScopeOption[] = [
      {
        id: "project",
        title: t("issue.advanced-search.scope.project.title"),
        description: t("issue.advanced-search.scope.project.description"),
        options: projectList.value.map<ValueOption>((proj) => {
          const name = extractProjectResourceName(proj.name);
          return {
            value: name,
            keywords: [name, proj.title, proj.key],
            render: () => {
              const children: VNode[] = [
                h(ProjectV1Name, { project: proj, link: false }),
              ];
              if (proj.workflow === Workflow.VCS) {
                children.push(h(GitIcon, { class: "h-4" }));
              }
              return h("div", { class: "flex items-center gap-x-2" }, children);
            },
          };
        }),
      },
      {
        id: "actor",
        title: t("audit-log.advanced-search.scope.actor.title"),
        description: t("audit-log.advanced-search.scope.actor.description"),
        options: principalSearchValueOptions.value,
      },
      {
        id: "method",
        title: t("audit-log.advanced-search.scope.method.title"),
        description: t("audit-log.advanced-search.scope.method.description"),
        options: ALL_METHODS.map((method) => {
          return {
            value: method,
            keywords: [method],
            render: () => "",
          };
        }),
      },
      {
        id: "level",
        title: t("audit-log.advanced-search.scope.level.title"),
        description: t("audit-log.advanced-search.scope.level.description"),
        options: Object.values(AuditLog_Severity).map((severity) => {
          return {
            value: severity,
            keywords: [severity],
            render: () => "",
          };
        }),
      },
    ];
    const supportOptionIdSet = new Set(supportOptionIdList.value);
    return scopes.filter((scope) => supportOptionIdSet.has(scope.id));
  });

  // filteredScopeOptions will filter search options by chosen scope.
  // For example, if users select a specific project, we should only allow them select instances related with this project.
  const filteredScopeOptions = computed((): ScopeOption[] => {
    const clone = fullScopeOptions.value.map((scope) => ({
      ...scope,
      options: scope.options.map((option) => ({
        ...option,
      })),
    }));

    return clone;
  });

  // availableScopeOptions will hide chosen search scope.
  // For example, if uses already select the instance, we should NOT show the instance scope in the dropdown.
  const availableScopeOptions = computed((): ScopeOption[] => {
    const existedScopes = new Set<SearchScopeId>(
      params.value.scopes.map((scope) => scope.id)
    );

    return filteredScopeOptions.value.filter((scope) => {
      if (existedScopes.has(scope.id)) {
        return false;
      }
      return true;
    });
  });

  const menuView = ref<"scope" | "value">();
  const currentScope = ref<SearchScopeId>();
  const currentScopeOption = computed(() => {
    if (currentScope.value) {
      return filteredScopeOptions.value.find(
        (opt) => opt.id === currentScope.value
      );
    }
    return undefined;
  });
  const scopeOptions = computed(() => {
    if (menuView.value === "scope") return availableScopeOptions.value;
    return [];
  });
  const valueOptions = computed(() => {
    if (menuView.value === "value" && currentScopeOption.value) {
      return currentScopeOption.value.options;
    }
    return [];
  });

  return {
    fullScopeOptions,
    filteredScopeOptions,
    availableScopeOptions,
    menuView,
    currentScope,
    currentScopeOption,
    scopeOptions,
    valueOptions,
  };
};

const renderSpan = (content: string) => {
  return h("span", {}, content);
};
