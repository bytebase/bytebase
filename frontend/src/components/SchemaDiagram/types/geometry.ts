export type Size = {
  width: number;
  height: number;
};

export type Position = {
  x: number;
  y: number;
};

export type Rect = Position & Size;

export type Path = Position[];
