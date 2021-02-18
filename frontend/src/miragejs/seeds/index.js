import faker from "faker";

/*
 * Mirage JS guide on Seeds: https://miragejs.com/docs/data-layer/factories#in-development
 */
const workspacesSeeder = (server) => {
  // Workspace id is ALWAYS 1 for on-premise deployment
  const workspace1 = server.schema.workspaces.find(1);

  // Workspace 2 is just for verifying we are not returning
  // resources from different workspaces.
  const workspace2 = server.schema.workspaces.find(101);

  // Environment
  let environmentList1 = [];
  for (let i = 0; i < 4; i++) {
    environmentList1.push(
      server.create("environment", {
        workspace: workspace1,
      })
    );
  }
  workspace1.update({ environment: environmentList1 });

  let environmentList2 = [];
  for (let i = 0; i < 4; i++) {
    environmentList2.push(
      server.create("environment", {
        workspace: workspace2,
      })
    );
  }
  workspace2.update({ environment: environmentList2 });

  // Instance
  for (let i = 0; i < 4; i++) {
    server.create("instance", {
      workspace: workspace1,
      name: "ws1 instance " + i + " " + faker.fake("{{lorem.word}}"),
      environmentId: environmentList1[i].id,
    });
  }

  for (let i = 0; i < 4; i++) {
    server.create("instance", {
      workspace: workspace2,
      name: "ws2 instance " + i + " " + faker.fake("{{lorem.word}}"),
      environmentId: environmentList2[i].id,
    });
  }

  // Task
  const ws1Owner = server.schema.users.find(1);
  const ws1DBA = server.schema.users.find(2);
  const ws1Dev1 = server.schema.users.find(3);
  const ws1Dev2 = server.schema.users.find(5);

  const ws2DBA = server.schema.users.find(4);
  const ws2Dev = server.schema.users.find(1);

  server.create("task", {
    type: "bytebase.general",
    creator: {
      id: ws1Dev1.id,
      name: ws1Dev1.name,
    },
    assignee: {
      id: ws1Owner.id,
      name: ws1Owner.name,
    },
    subscriberIdList: [ws1DBA.id, ws1Dev2.id],
    stageProgressList: [
      {
        id: "1",
        name: "Request",
        type: "SIMPLE",
        status: "PENDING",
      },
    ],
    workspace: workspace1,
  });

  for (let i = 0; i < 5; i++) {
    server.create("task", {
      creator: {
        id: ws1Dev1.id,
        name: ws1Dev1.name,
      },
      assignee: {
        id: ws1Owner.id,
        name: ws1Owner.name,
      },
      subscriberIdList: [ws1DBA.id, ws1Dev2.id],
      ...fillStage(environmentList1),
      workspace: workspace1,
    });
  }

  for (let i = 0; i < 15; i++) {
    server.create("task", {
      creator: {
        id: ws1Owner.id,
        name: ws1Owner.name,
      },
      assignee: {
        id: ws1DBA.id,
        name: ws1DBA.name,
      },
      subscriberIdList: [ws1Dev2.id],
      ...fillStage(environmentList1),
      workspace: workspace1,
    });
  }

  for (let i = 0; i < 15; i++) {
    server.create("task", {
      creator: {
        id: ws1Dev2.id,
        name: ws1Dev2.name,
      },
      assignee: {
        id: ws1DBA.id,
        name: ws1DBA.name,
      },
      subscriberIdList: [ws1Owner.id, ws1Dev1.id],
      ...fillStage(environmentList1),
      workspace: workspace1,
    });
  }

  server.create("task", {
    creator: {
      id: ws2Dev.id,
      name: ws2Dev.name,
    },
    assignee: {
      id: ws2DBA.id,
      name: ws2DBA.name,
    },
    ...fillStage(environmentList2),
    workspace: workspace2,
  });

  // Naming convention
  // xxx <<workspace #>><<group #>><<project #>>
  // e.g. group 12 (1st workspace and 1st group inside)
  // e.g. project 241 (2nd workspace, 4th group inside and 1st project inside)

  // Group
  // Workspace 1
  const group1 = server.create("group", {
    name: "group 11",
    slug: "grp11",
    namepsace: "",
    workspace: workspace1,
  });
  const group2 = server.create("group", {
    name: "group 12",
    slug: "grp12",
    namepsace: "",
    workspace: workspace1,
  });
  const group3 = server.create("group", {
    name: "group 13",
    slug: "grp13",
    namepsace: "",
    workspace: workspace1,
  });
  // Workspace 2
  const group4 = server.create("group", {
    name: "group 24",
    slug: "grp24",
    namepsace: "",
    workspace: workspace2,
  });

  // Project
  // Workspace 1
  const project1 = server.create("project", {
    name: "proj 111",
    slug: "proj111",
    namespace: "grp11",
    workspace: workspace1,
    group: group1,
  });
  const project2 = server.create("project", {
    name: "proj 122",
    slug: "proj122",
    namespace: "grp12",
    workspace: workspace1,
    group: group2,
  });
  const project3 = server.create("project", {
    name: "proj 131",
    slug: "proj131",
    namespace: "grp13",
    workspace: workspace1,
    group: group3,
  });
  // Workspace 2
  const project4 = server.create("project", {
    name: "proj 241",
    slug: "proj241",
    namespace: "grp24",
    workspace: workspace2,
    group: group4,
  });

  // User 1 is owner of group1, developer of group3 and owner of group4
  const user1 = server.schema.users.find(1);
  server.create("groupRole", {
    role: "OWNER",
    user: user1,
    group: group1,
  });
  server.create("groupRole", {
    role: "DEVELOPER",
    user: user1,
    group: group3,
  });
  server.create("groupRole", "owner", {
    role: "OWNER",
    user: user1,
    group: group4,
  });

  // User 2 is owner of group3 and developer of group4
  const user2 = server.schema.users.find(2);
  server.create("groupRole", "owner", {
    role: "OWNER",
    user: user2,
    group: group3,
  });

  server.create("groupRole", "developer", {
    role: "DEVELOPER",
    user: user2,
    group: group4,
  });

  // Bookmarks
  // Task 1
  const task1 = server.schema.tasks.find(1);
  server.create("bookmark", {
    workspace: workspace1,
    name: "Task #" + task1.id,
    link: `/task/${task1.id}`,
  });

  // Task 2
  const task2 = server.schema.tasks.find(2);
  server.create("bookmark", {
    workspace: workspace1,
    name: "Task #" + task2.id,
    link: `/task/${task2.id}`,
  });

  // Task 3
  const task3 = server.schema.tasks.findBy({
    workspaceId: workspace2.id,
  });
  server.create("bookmark", {
    workspace: workspace2,
    name: "Task #" + task3.id,
    link: `/task/${task3.id}`,
  });
};

const fillStage = (environmentList) => {
  const i = Math.floor(Math.random() * 5);
  if (i % 5 == 0) {
    return {
      status: "OPEN",
      stageProgressList: [
        {
          id: "1",
          name: environmentList[0].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[0].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "PENDING",
        },
        {
          id: "2",
          name: environmentList[1].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[1].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "PENDING",
        },
      ],
    };
  } else if (i % 5 == 1) {
    return {
      status: "OPEN",
      stageProgressList: [
        {
          id: "1",
          name: environmentList[0].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[0].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "DONE",
        },
        {
          id: "2",
          name: environmentList[1].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[1].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "DONE",
        },
        {
          id: "3",
          name: environmentList[2].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[2].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "RUNNING",
        },
        {
          id: "4",
          name: environmentList[3].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[3].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "PENDING",
        },
      ],
    };
  } else if (i % 5 == 2) {
    return {
      status: "DONE",
      stageProgressList: [
        {
          id: "1",
          name: environmentList[0].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[0].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "DONE",
        },
        {
          id: "2",
          name: environmentList[1].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[1].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "SKIPPED",
        },
        {
          id: "3",
          name: environmentList[2].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[2].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "DONE",
        },
        {
          id: "4",
          name: environmentList[3].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[3].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "DONE",
        },
      ],
    };
  } else if (i % 5 == 3) {
    return {
      status: "OPEN",
      stageProgressList: [
        {
          id: "1",
          name: environmentList[0].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[0].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "DONE",
        },
        {
          id: "2",
          name: environmentList[1].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[1].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "FAILED",
        },
        {
          id: "3",
          name: environmentList[2].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[2].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "PENDING",
        },
        {
          id: "4",
          name: environmentList[3].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[3].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "PENDING",
        },
      ],
    };
  } else {
    return {
      status: "CANCELED",
      stageProgressList: [
        {
          id: "1",
          name: environmentList[0].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[0].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "DONE",
        },
        {
          id: "2",
          name: environmentList[1].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[1].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "SKIPPED",
        },
        {
          id: "3",
          name: environmentList[2].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[2].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "DONE",
        },
        {
          id: "4",
          name: environmentList[3].name,
          type: "ENVIRONMENT",
          environmentId: environmentList[3].id,
          runnable: {
            auto: true,
            run: () => {},
          },
          status: "CANCELED",
        },
      ],
    };
  }
};

export default function seeds(server) {
  server.loadFixtures();
  workspacesSeeder(server);
}
