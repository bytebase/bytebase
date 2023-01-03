import { expect, it } from "vitest";
import {
  AddColumnContext,
  DropColumnContext,
  ChangeColumnContext,
  AlterColumnContext,
} from "@/types";
import { Column } from "@/types/schemaEditor/atomType";
import { diffColumnList } from "./diffColumn";

it("diff add column list", () => {
  const testList: {
    originColumnList: Column[];
    columnList: Column[];
    wanted: {
      addColumnList: AddColumnContext[];
      alterColumnList: AlterColumnContext[];
      changeColumnList: ChangeColumnContext[];
      dropColumnList: DropColumnContext[];
    };
  }[] = [
    {
      originColumnList: [],
      columnList: [
        {
          id: "1",
          name: "id",
          type: "int",
          comment: "",
          nullable: false,
          status: "created",
        } as any as Column,
      ],
      wanted: {
        addColumnList: [
          {
            name: "id",
            type: "int",
            comment: "",
            nullable: false,
            characterSet: "",
            collation: "",
            default: undefined,
          },
        ],
        alterColumnList: [],
        changeColumnList: [],
        dropColumnList: [],
      },
    },
  ];

  for (const test of testList) {
    const result = diffColumnList(test.originColumnList, test.columnList);
    expect(result).toStrictEqual(test.wanted);
  }
});

it("diff modify column list", () => {
  const testList: {
    originColumnList: Column[];
    columnList: Column[];
    wanted: {
      addColumnList: AddColumnContext[];
      alterColumnList: AlterColumnContext[];
      changeColumnList: ChangeColumnContext[];
      dropColumnList: DropColumnContext[];
    };
  }[] = [
    {
      originColumnList: [
        {
          id: "1",
          name: "id",
          type: "int",
          comment: "",
          nullable: true,
          status: "normal",
        } as any as Column,
      ],
      columnList: [
        {
          id: "1",
          name: "id",
          type: "varchar",
          comment: "",
          nullable: false,
          status: "normal",
        } as any as Column,
      ],
      wanted: {
        addColumnList: [],
        alterColumnList: [
          {
            oldName: "id",
            newName: "id",
            type: "varchar",
            nullable: false,
            defaultChanged: false,
          },
        ],
        changeColumnList: [
          {
            oldName: "id",
            newName: "id",
            type: "varchar",
            comment: "",
            nullable: false,
            characterSet: "",
            collation: "",
            default: undefined,
          },
        ],
        dropColumnList: [],
      },
    },
  ];

  for (const test of testList) {
    const result = diffColumnList(test.originColumnList, test.columnList);
    expect(result).toStrictEqual(test.wanted);
  }
});

it("diff drop column list", () => {
  const testList: {
    originColumnList: Column[];
    columnList: Column[];
    wanted: {
      addColumnList: AddColumnContext[];
      alterColumnList: AlterColumnContext[];
      changeColumnList: ChangeColumnContext[];
      dropColumnList: DropColumnContext[];
    };
  }[] = [
    {
      originColumnList: [
        {
          id: "1",
          name: "id",
          type: "int",
          comment: "",
          nullable: true,
        } as any as Column,
      ],
      columnList: [
        {
          id: "1",
          name: "id",
          type: "int",
          comment: "",
          nullable: true,
          status: "dropped",
        } as any as Column,
      ],
      wanted: {
        addColumnList: [],
        alterColumnList: [],
        changeColumnList: [],
        dropColumnList: [
          {
            name: "id",
          },
        ],
      },
    },
  ];

  for (const test of testList) {
    const result = diffColumnList(test.originColumnList, test.columnList);
    expect(result).toStrictEqual(test.wanted);
  }
});
