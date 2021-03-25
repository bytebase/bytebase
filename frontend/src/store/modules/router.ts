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
    {
      // /task/:taskSlug
      // Total 2 elements, 2nd element is the task slug
      const taskComponents = currentRoute.path.match(
        "/task/([0-9a-zA-Z_-]+)"
      ) || ["/", undefined];
      if (taskComponents[1]) {
        return {
          taskSlug: taskComponents[1],
        };
      }
    }

    {
      // /instance/:instanceSlug/db/:databaseSlug
      // Total 3 elements, 2nd element is the instance slug, 3rd element is the data source slug
      const databaseComponents = currentRoute.path.match(
        "/instance/([0-9a-zA-Z_-]+)/db/([0-9a-zA-Z_-]+)"
      ) || ["/", undefined, undefined];
      if (databaseComponents[1] && databaseComponents[2]) {
        return {
          instanceSlug: databaseComponents[1],
          databaseSlug: databaseComponents[2],
        };
      }
    }

    {
      // /instance/:instanceSlug/ds/:dataSourceSlug
      // Total 3 elements, 2nd element is the instance slug, 3rd element is the data source slug
      const dataSourceComponents = currentRoute.path.match(
        "/instance/([0-9a-zA-Z_-]+)/ds/([0-9a-zA-Z_-]+)"
      ) || ["/", undefined, undefined];
      if (dataSourceComponents[1] && dataSourceComponents[2]) {
        return {
          instanceSlug: dataSourceComponents[1],
          dataSourceSlug: dataSourceComponents[2],
        };
      }
    }

    {
      // /instance/:instanceSlug
      // Total 2 elements, 2nd element is the instance slug
      const instanceComponents = currentRoute.path.match(
        "/instance/([0-9a-zA-Z_-]+)"
      ) || ["/", undefined];
      if (instanceComponents[1]) {
        return {
          instanceSlug: instanceComponents[1],
        };
      }
    }

    {
      // /u/:principalId
      // Total 2 elements, 2nd element is the principal id
      const profileComponents = currentRoute.path.match(
        "/u/([0-9a-zA-Z_-]+)"
      ) || ["/", undefined];
      if (profileComponents[1]) {
        return {
          principalId: profileComponents[1],
        };
      }
    }

    return {};
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
