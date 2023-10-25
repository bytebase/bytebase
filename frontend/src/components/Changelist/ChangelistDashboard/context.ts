import Emittery from "emittery";
import { InjectionKey, Ref, inject, provide, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";

export type ChangelistDashboardFilter = {
  project: string;
  keyword: string;
};

export type ChangelistDashboardEvents = Emittery<{
  refresh: undefined;
}>;

export type ChangelistDashboardContext = {
  filter: Ref<ChangelistDashboardFilter>;
  showCreatePanel: Ref<boolean>;
  events: ChangelistDashboardEvents;
};

export const KEY = Symbol(
  "bb.changelist.dashboard"
) as InjectionKey<ChangelistDashboardContext>;

export const useChangelistDashboardContext = () => {
  return inject(KEY)!;
};

export const provideChangelistDashboardContext = (
  project: string = "projects/-"
) => {
  const route = useRoute();
  const router = useRouter();

  const context: ChangelistDashboardContext = {
    filter: ref({
      project,
      keyword: "",
    }),
    showCreatePanel: ref(false),
    events: new Emittery(),
  };

  watch(
    () => route.query.project as string,
    (project) => {
      if (project) {
        context.filter.value.project = project;
      }
    },
    { immediate: true }
  );

  watch(
    () => context.filter.value.project,
    (project) => {
      const projectInQuery = (route.query.project as string) ?? "";
      const projectParam = project === "projects/-" ? "" : project;
      const query = {
        ...route.query,
        project: projectParam,
      };
      if (query.project === "") {
        delete (query as any)["project"];
      }
      if (projectInQuery !== projectParam) {
        router.replace({
          ...route,
          query,
        });
      }
    },
    { immediate: true }
  );

  provide(KEY, context);

  return context;
};
