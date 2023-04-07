import dayjs from "dayjs";
import { random } from "lodash-es";
import { defineStore } from "pinia";

import { databaseServiceClient } from "@/grpcweb";
import {
  ListSlowQueriesRequest,
  SlowQueryDetails,
  SlowQueryLog,
} from "@/types/proto/v1/database_service";
import { instanceSupportSlowQuery, randomString } from "@/utils";
import { useDatabaseStore } from "./database";
import { useEnvironmentStore } from "./environment";
import { useInstanceStore } from "./instance";

export const useSlowQueryStore = defineStore("slow-query", () => {
  const fetchSlowQueryLogList = async (
    request: Partial<ListSlowQueriesRequest> = {}
  ) => {
    console.log(
      `fetchSlowQueryLogList(${JSON.stringify(request, null, "  ")})`
    );
    // await sleep(500);
    try {
      const _res = await databaseServiceClient.listSlowQueries(request);
    } catch (ex) {
      // console.log("listSlowQueries", ex);
    }
    return generateMockSlowQueryList();
  };

  return { fetchSlowQueryLogList };
});

const _sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

const generateMockSlowQueryList = async () => {
  const list: SlowQueryLog[] = [];

  const environmentList = await useEnvironmentStore().fetchEnvironmentList([
    "NORMAL",
  ]);
  await useInstanceStore().fetchInstanceList();
  await useDatabaseStore().fetchDatabaseList();

  for (let i = 0; i < 10; i++) {
    const environment = environmentList[random(0, environmentList.length - 1)];
    const instanceList = useInstanceStore()
      .getInstanceListByEnvironmentId(environment.id)
      .filter(instanceSupportSlowQuery);
    const instance = instanceList[random(0, instanceList.length - 1)];
    if (!instance) continue;
    const databaseList = useDatabaseStore().getDatabaseListByInstanceId(
      instance.id
    );
    const database = databaseList[random(0, databaseList.length - 1)];
    if (!database) continue;
    const project = `projects/${database.project.resourceId}`;
    const resource = `environments/${environment.resourceId}/instances/${instance.resourceId}/databases/${database.id}`;
    list.push(
      SlowQueryLog.fromJSON({
        project,
        resource,
        statistics: {
          sqlFingerprint: generateMockSQL(),
          count: random(1, 10000),
          latestLogTime: dayjs()
            .subtract(random(1, 30 * 86400), "seconds")
            .toDate(),
          nightyFifthPercentileQueryTime: {
            seconds: random(0.0001, 2, true),
          },
          averageQueryTime: {
            seconds: random(0.0001, 2, true),
          },
          nightyFifthPercentileRowsExamined: random(0, 100000),
          averageRowsExamined: random(0, 100000),
          nightyFifthPercentileRowsSent: random(0, 100000),
          averageRowsSent: random(0, 100000),
          samples: generateMockSlowQueryDetails(random(5, 20)),
        },
      })
    );
  }

  list.sort((a, b) => -(a.statistics!.count - b.statistics!.count));
  return list;
};

const generateMockSlowQueryDetails = (n: number) => {
  const details: SlowQueryDetails[] = [];
  for (let i = 0; i < n; i++) {
    details.push(
      SlowQueryDetails.fromJSON({
        startTime: dayjs()
          .subtract(random(1, 30 * 86400), "seconds")
          .toDate(),
        queryTime: {
          seconds: random(0.0001, 1, true),
        },
        lockTime: {
          seconds: random(0.0001, 1, true),
        },
        rowsExamined: random(0, 100000),
        rowsSent: random(0, 100000),
        sqlText: generateMockSQL(),
      })
    );
  }
  return details;
};

const generateMockSQL = () => {
  const lines: string[] = [];
  const r = random(1, 10);
  for (let i = 0; i < r; i++) {
    const n = random(3, 10);
    const words: string[] = [];
    for (let j = 0; j < n; j++) {
      words.push(randomString(random(1, 10)));
    }
    lines.push(words.join(" "));
  }
  return lines.join("\n");
};
