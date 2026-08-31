import { create as createProto, toJson } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { changelogServiceClientConnect } from "@/api";
import { UNKNOWN_ID } from "@/types";
import {
  ChangelogView,
  GetChangelogRequestSchema,
  ListChangelogsRequestSchema,
} from "@/types/proto-es/v1/changelog_service_pb";
import { celString } from "@/utils/v1/celLiteral";
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
    const response = await changelogServiceClientConnect.listChangelogs(
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
    const changelog = await changelogServiceClientConnect.getChangelog(
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

    const current = await get().getOrFetchChangelogByName(
      name,
      ChangelogView.FULL
    );
    if (!current?.createTime) {
      return undefined;
    }
    const createTime = toJson(TimestampSchema, current.createTime);
    const { changelogs } = await get().listChangelogs({
      parent: database,
      pageSize: 1,
      view: ChangelogView.FULL,
      filter: `has_schema_snapshot == true && create_time < ${celString(createTime)}`,
    });
    return changelogs[0];
  },
});

export { changelogCacheKey };
