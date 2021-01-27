import { RouteLocationNormalized } from "vue-router";
import { RouterSlug } from "../../types";

export interface RouterState {}

const getters = {
  backPath: (state: RouterState) => (): string => {
    return localStorage.getItem("ui.backPath") || "/";
  },
  routeSlug: (state: RouterState) => (
    currentRoute: RouteLocationNormalized
  ): RouterSlug => {
    // /pipeline/<<pipeline-id>>
    // Total 2 elements, 2nd element is the pipeline id
    const pipelineComponents = currentRoute.path.match(
      "/pipeline/([0-9a-zA-Z_]+)"
    ) || ["/", undefined];
    if (pipelineComponents[1]) {
      return {
        pipelineId: pipelineComponents[1],
      };
    }

    // /<<group-slug>>/<<project-slug>>/pipeline/<<pipelineId>>
    // Total 7 elements, 1st element is always the fullpath
    //
    // case 1: "" or /
    //    [
    //      "/",
    //    ]
    // case 2: <<group-slug>> or /<<group-slug>>/project
    //    [
    //      "<<fullpath>>",
    //      "<<group-slug>>"
    //    ]
    // case 3: /<<group-slug>>/<<project-slug>>
    //    [
    //      "<<fullpath>>",
    //      "<<group-slug>>",
    //      "/project/<<project-slug>>",
    //      "<<project-slug>>",
    //      "<<project-slug>>"
    //    ]
    // case 4: /<<group-slug>>/<<project-slug>>/pipeline
    //    [
    //      "<<fullpath>>",
    //      "<<group-slug>>",
    //      "/project/<<project-slug>>",
    //      "<<project-slug>>",
    //      "<<project-slug>>"
    //    ]
    // case 5: /<<group-slug>>/<<project-slug>>/pipeline/<<pipelineId>>
    //    [
    //      "<<fullpath>>",
    //      "<<group-slug>>",
    //      "/project/<<project-slug>>/pipeline/<<pipelineId>>",
    //      "<<project-slug>>/pipeline/<<pipelineId",
    //      "<<project-slug>>",
    //      "/pipeline/<<pipelineId>>",
    //      "<<pipelineId>>",
    //    ]
    const pathComponents = currentRoute.path.match(
      "/([0-9a-zA-Z_]+)(/(([0-9a-zA-Z_]+)(/pipeline/([0-9a-zA-Z_]+)?)?)?)?"
    ) || [
      "/",
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
    ];
    return {
      groupSlug: pathComponents[1],
      projectSlug: pathComponents[4],
    };
  },
};

const actions = {
  setBackPath({ commit }: any, backPath: string) {
    localStorage.setItem("ui.backPath", backPath);
    return backPath;
  },
};

export default {
  namespaced: true,
  getters,
  actions,
};
