import { create as createProto } from "@bufbuild/protobuf";
import { databaseServiceClientConnect } from "@/connect";
import { UNKNOWN_ID } from "@/types";
import {
  ChangelogView,
  GetChangelogRequestSchema,
  ListChangelogsRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { extractChangelogUID } from "@/utils/v1/changelog";
import type { AppSliceCreator, ChangelogSlice } from "./types";

const changelogCacheKey = (name: string, view: ChangelogView) =>
  `${name}|${view}`;

const normalizeView = (view?: ChangelogView) => view ?? ChangelogView.BASIC;

const changelogListCacheKey = (params: {
  parent: string;
  view?: ChangelogView;
  pageSize?: number;
  pageToken?: string;
  filter?: string;
}) =>
  [
    params.parent,
    normalizeView(params.view),
    params.pageSize ?? 0,
    params.pageToken ?? "",
    params.filter ?? "",
  ].join("|");

export const createChangelogSlice: AppSliceCreator<ChangelogSlice> = (
  set,
  get
) => ({
  changelogsByCacheKey: {},
  changelogsByDatabase: {},
  changelogRequests: {},

  clearChangelogCache: (parent) => {
    set((state) => {
      return {
        changelogsByDatabase: Object.fromEntries(
          Object.entries(state.changelogsByDatabase).filter(
            ([key]) => !key.startsWith(`${parent}|`)
          )
        ),
      };
    });
  },

  listChangelogs: async (params) => {
    const { parent } = params;
    if (!parent) {
      throw new Error('"parent" field is required');
    }

    const view = normalizeView(params.view);
    const listCacheKey = changelogListCacheKey({
      parent,
      pageSize: params.pageSize,
      pageToken: params.pageToken,
      view,
      filter: params.filter,
    });
    const response = await databaseServiceClientConnect.listChangelogs(
      createProto(ListChangelogsRequestSchema, {
        parent,
        pageSize: params.pageSize,
        pageToken: params.pageToken ?? "",
        view,
        filter: params.filter ?? "",
      })
    );
    set((state) => ({
      changelogsByDatabase: {
        ...state.changelogsByDatabase,
        [listCacheKey]: response.changelogs,
      },
      changelogsByCacheKey: {
        ...state.changelogsByCacheKey,
        ...Object.fromEntries(
          response.changelogs.map((changelog) => [
            changelogCacheKey(changelog.name, view),
            changelog,
          ])
        ),
      },
    }));
    return {
      changelogs: response.changelogs,
      nextPageToken: response.nextPageToken,
    };
  },

  getOrFetchChangelogListOfDatabase: async (database, pageSize, view) => {
    const cached =
      get().changelogsByDatabase[
        changelogListCacheKey({
          parent: database,
          pageSize,
          view: normalizeView(view),
        })
      ];
    if (cached) return cached;
    const response = await get().listChangelogs({
      parent: database,
      pageSize,
      view: normalizeView(view),
    });
    return response.changelogs;
  },

  changelogListByDatabase: (database) =>
    Object.entries(get().changelogsByDatabase).find(([key]) => {
      const [parent, view, , pageToken, filter] = key.split("|");
      return (
        parent === database &&
        view === String(ChangelogView.BASIC) &&
        pageToken === "" &&
        filter === ""
      );
    })?.[1] ?? [],

  fetchChangelog: async (params) => {
    if (!params.name) return undefined;
    const view = normalizeView(params.view);
    const changelog = await databaseServiceClientConnect.getChangelog(
      createProto(GetChangelogRequestSchema, {
        name: params.name,
        view,
      })
    );
    set((state) => ({
      changelogsByCacheKey: {
        ...state.changelogsByCacheKey,
        [changelogCacheKey(changelog.name, view)]: changelog,
      },
    }));
    return changelog;
  },

  getOrFetchChangelogByName: async (name, view = ChangelogView.BASIC) => {
    const uid = extractChangelogUID(name);
    if (!uid || uid === String(UNKNOWN_ID)) {
      return undefined;
    }

    const key = changelogCacheKey(name, view);
    const cached = get().changelogsByCacheKey[key];
    if (cached) return cached;
    const pending = get().changelogRequests[key];
    if (pending) return pending;

    const request = get()
      .fetchChangelog({ name, view })
      .finally(() => {
        set((state) => {
          const { [key]: _, ...changelogRequests } = state.changelogRequests;
          return { changelogRequests };
        });
      });
    set((state) => ({
      changelogRequests: { ...state.changelogRequests, [key]: request },
    }));
    return request;
  },

  getChangelogByName: (name, view) => {
    if (view === undefined) {
      return (
        get().changelogsByCacheKey[
          changelogCacheKey(name, ChangelogView.FULL)
        ] ??
        get().changelogsByCacheKey[changelogCacheKey(name, ChangelogView.BASIC)]
      );
    }
    return get().changelogsByCacheKey[changelogCacheKey(name, view)];
  },

  fetchPreviousChangelog: async (name) => {
    const parts = name.split("/changelogs/");
    if (parts.length !== 2) {
      return undefined;
    }
    const database = parts[0];
    const currentUid = extractChangelogUID(name);
    if (!currentUid || currentUid === String(UNKNOWN_ID)) {
      return undefined;
    }

    const { changelogs } = await get().listChangelogs({
      parent: database,
      pageSize: 1000,
      view: ChangelogView.BASIC,
    });
    const index = changelogs.findIndex(
      (changelog) => extractChangelogUID(changelog.name) === currentUid
    );
    if (index < 0 || index === changelogs.length - 1) {
      return undefined;
    }
    return get().getOrFetchChangelogByName(
      changelogs[index + 1].name,
      ChangelogView.FULL
    );
  },
});

export { changelogCacheKey };
