import { expect, it } from "vitest";
import {
  AddColumnContext,
  Column,
  DropColumnContext,
  ModifyColumnContext,
} from "@/types";
import { diffColumnList } from "./diffColumn";

it("diff add column list", () => {
  const testList: {
    originColumnList: Column[];
    targetColumnList: Column[];
    wanted: {
      addColumnList: AddColumnContext[];
      modifyColumnList: ModifyColumnContext[];
      dropColumnList: DropColumnContext[];
    };
  }[] = [
    {
      originColumnList: [],
      targetColumnList: [
        {
          id: -1,
          name: "id",
          type: "int",
          characterSet: "",
          collation: "",
          comment: "",
          nullable: false,
          default: undefined,
        } as Column,
      ],
      wanted: {
        addColumnList: [
          {
            name: "id",
            type: "int",
            characterSet: "",
            collation: "",
            comment: "",
            nullable: false,
            default: undefined,
          },
        ],
        modifyColumnList: [],
        dropColumnList: [],
      },
    },
  ];

  for (const test of testList) {
    const result = diffColumnList(test.originColumnList, test.targetColumnList);
    expect(result).toStrictEqual(test.wanted);
  }
});

it("diff modify column list", () => {
  const testList: {
    originColumnList: Column[];
    targetColumnList: Column[];
    wanted: {
      addColumnList: AddColumnContext[];
      modifyColumnList: ModifyColumnContext[];
      dropColumnList: DropColumnContext[];
    };
  }[] = [
    {
      originColumnList: [
        {
          name: "id",
          type: "int",
          characterSet: "",
          collation: "",
          comment: "",
          nullable: true,
          default: undefined,
        } as Column,
      ],
      targetColumnList: [
        {
          name: "id",
          type: "int",
          characterSet: "",
          collation: "",
          comment: "",
          nullable: false,
          default: undefined,
        } as Column,
      ],
      wanted: {
        addColumnList: [],
        modifyColumnList: [
          {
            name: "id",
            type: "int",
            characterSet: "",
            collation: "",
            comment: "",
            nullable: false,
            default: undefined,
          },
        ],
        dropColumnList: [],
      },
    },
  ];

  for (const test of testList) {
    const result = diffColumnList(test.originColumnList, test.targetColumnList);
    expect(result).toStrictEqual(test.wanted);
  }
});

it("diff drop column list", () => {
  const testList: {
    originColumnList: Column[];
    targetColumnList: Column[];
    wanted: {
      addColumnList: AddColumnContext[];
      modifyColumnList: ModifyColumnContext[];
      dropColumnList: DropColumnContext[];
    };
  }[] = [
    {
      originColumnList: [
        {
          name: "id",
          type: "int",
          characterSet: "",
          collation: "",
          comment: "",
          nullable: true,
          default: undefined,
        } as Column,
      ],
      targetColumnList: [],
      wanted: {
        addColumnList: [],
        modifyColumnList: [],
        dropColumnList: [
          {
            name: "id",
          },
        ],
      },
    },
  ];

  for (const test of testList) {
    const result = diffColumnList(test.originColumnList, test.targetColumnList);
    expect(result).toStrictEqual(test.wanted);
  }
});
