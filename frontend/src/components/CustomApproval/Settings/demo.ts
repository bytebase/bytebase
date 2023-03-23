const groups = {
  id: 0,
  callExpr: {
    function: "_&&_",
    args: [
      {
        id: 0,
        callExpr: {
          function: "_&&_",
          args: [
            {
              id: 0,
              callExpr: {
                target: {
                  id: 0,
                  identExpr: {
                    name: "environment",
                  },
                },
                function: "contains",
                args: [
                  {
                    id: 0,
                    constExpr: {
                      stringValue: "1",
                    },
                  },
                ],
              },
            },
            {
              id: 0,
              callExpr: {
                target: {
                  id: 0,
                  identExpr: {
                    name: "environment",
                  },
                },
                function: "contains",
                args: [
                  {
                    id: 0,
                    constExpr: {
                      stringValue: "2",
                    },
                  },
                ],
              },
            },
          ],
        },
      },
      {
        id: 0,
        callExpr: {
          function: "_&&_",
          args: [
            {
              id: 0,
              callExpr: {
                target: {
                  id: 0,
                  identExpr: {
                    name: "environment",
                  },
                },
                function: "contains",
                args: [
                  {
                    id: 0,
                    constExpr: {
                      stringValue: "3",
                    },
                  },
                ],
              },
            },
            {
              id: 0,
              callExpr: {
                target: {
                  id: 0,
                  identExpr: {
                    name: "environment",
                  },
                },
                function: "contains",
                args: [
                  {
                    id: 0,
                    constExpr: {
                      stringValue: "4",
                    },
                  },
                ],
              },
            },
          ],
        },
      },
      {
        id: 0,
        callExpr: {
          function: "_&&_",
          args: [
            {
              id: 0,
              callExpr: {
                target: {
                  id: 0,
                  identExpr: {
                    name: "environment",
                  },
                },
                function: "contains",
                args: [
                  {
                    id: 0,
                    constExpr: {
                      stringValue: "5",
                    },
                  },
                ],
              },
            },
            {
              id: 0,
              callExpr: {
                target: {
                  id: 0,
                  identExpr: {
                    name: "environment",
                  },
                },
                function: "contains",
                args: [
                  {
                    id: 0,
                    constExpr: {
                      stringValue: "6",
                    },
                  },
                ],
              },
            },
          ],
        },
      },
    ],
  },
};

const single = {
  id: 0,
  callExpr: {
    target: {
      id: 0,
      identExpr: {
        name: "environment",
      },
    },
    function: "contains",
    args: [
      {
        id: 0,
        constExpr: {
          stringValue: "d",
        },
      },
    ],
  },
};

const collection = {
  id: 0,
  callExpr: {
    function: "@in",
    args: [
      {
        id: 0,
        identExpr: {
          name: "db_engine",
        },
      },
      {
        id: 0,
        listExpr: {
          elements: [
            {
              id: 0,
              constExpr: {
                stringValue: "MYSQL",
              },
            },
            {
              id: 0,
              constExpr: {
                stringValue: "POSTGRES",
              },
            },
          ],
          optionalIndices: [],
        },
      },
    ],
  },
};

const realDDL = {
  id: 0,
  callExpr: {
    function: "_&&_",
    args: [
      {
        id: 0,
        callExpr: {
          function: "_||_",
          args: [
            {
              id: 0,
              callExpr: {
                function: "@in",
                args: [
                  {
                    id: 0,
                    identExpr: {
                      name: "environment",
                    },
                  },
                  {
                    id: 0,
                    listExpr: {
                      elements: [
                        {
                          id: 0,
                          constExpr: {
                            stringValue: "prod",
                          },
                        },
                      ],
                      optionalIndices: [],
                    },
                  },
                ],
              },
            },
            {
              id: 0,
              callExpr: {
                target: {
                  id: 0,
                  identExpr: {
                    name: "database_name",
                  },
                },
                function: "contains",
                args: [
                  {
                    id: 0,
                    constExpr: {
                      stringValue: "cms",
                    },
                  },
                ],
              },
            },
          ],
        },
      },
      {
        id: 0,
        callExpr: {
          function: "_||_",
          args: [
            {
              id: 0,
              callExpr: {
                target: {
                  id: 0,
                  identExpr: {
                    name: "sql_type",
                  },
                },
                function: "startsWith",
                args: [
                  {
                    id: 0,
                    constExpr: {
                      stringValue: "CREATE",
                    },
                  },
                ],
              },
            },
            {
              id: 0,
              callExpr: {
                target: {
                  id: 0,
                  identExpr: {
                    name: "sql_type",
                  },
                },
                function: "startsWith",
                args: [
                  {
                    id: 0,
                    constExpr: {
                      stringValue: "DROP",
                    },
                  },
                ],
              },
            },
            {
              id: 0,
              callExpr: {
                target: {
                  id: 0,
                  identExpr: {
                    name: "sql_type",
                  },
                },
                function: "startsWith",
                args: [
                  {
                    id: 0,
                    constExpr: {
                      stringValue: "ALTER",
                    },
                  },
                ],
              },
            },
          ],
        },
      },
    ],
  },
};

const realDML = {
  id: 0,
  callExpr: {
    function: "_||_",
    args: [
      {
        id: 0,
        callExpr: {
          function: "@in",
          args: [
            {
              id: 0,
              identExpr: {
                name: "environment",
              },
            },
            {
              id: 0,
              listExpr: {
                elements: [
                  {
                    id: 0,
                    constExpr: {
                      stringValue: "preview",
                    },
                  },
                  {
                    id: 0,
                    constExpr: {
                      stringValue: "prod",
                    },
                  },
                ],
                optionalIndices: [],
              },
            },
          ],
        },
      },
      {
        id: 0,
        callExpr: {
          function: "_&&_",
          args: [
            {
              id: 0,
              callExpr: {
                function: "_>=_",
                args: [
                  {
                    id: 0,
                    identExpr: {
                      name: "insert_rows",
                    },
                  },
                  {
                    id: 0,
                    constExpr: {
                      int64Value: 100,
                    },
                  },
                ],
              },
            },
            {
              id: 0,
              callExpr: {
                function: "_<_",
                args: [
                  {
                    id: 0,
                    identExpr: {
                      name: "insert_rows",
                    },
                  },
                  {
                    id: 0,
                    constExpr: {
                      int64Value: 1000,
                    },
                  },
                ],
              },
            },
          ],
        },
      },
    ],
  },
};

type Demo = { key: string; expr: object };

const common: Demo[] = [
  { key: "分组接分组", expr: groups },
  { key: "简单条件", expr: single },
  { key: "@in 运算符", expr: collection },
];
export default {
  common,
  DDL: [...common, { key: "真实 DDL", expr: realDDL }],
  DML: [...common, { key: "真实 DML", expr: realDML }],
};
