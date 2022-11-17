import { AddColumnContext, Column } from "@/types";
import { expect, it } from "vitest";
import { diffColumnList } from "./diffColumn";

it("diff add column list", () => {
  const testList: {
    originColumnList: Column[];
    updatedColumnList: Column[];
    wanted: {
      addColumnList: AddColumnContext[];
    };
  }[] = [
    {
      originColumnList: [],
      updatedColumnList: [],
      wanted: {
        addColumnList: [],
      },
    },
    {
      originColumnList: [],
      updatedColumnList: [
        {
          name: "id",
          type: "int",
          nullable: false,
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
      },
    },
  ];

  for (const test of testList) {
    const result = diffColumnList(
      test.originColumnList,
      test.updatedColumnList
    );
    expect(result).toStrictEqual(test.wanted);
  }
});
