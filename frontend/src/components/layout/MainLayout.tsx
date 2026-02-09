"use client";

import { FC, PropsWithChildren } from "react";
import { CommandPalette } from "@/components/command-palette/CommandPalette";
import { WindowDragRegion } from "@/components/layout/WindowDragRegion";

export const MainLayout: FC<PropsWithChildren> = ({ children }) => {
  return (
    <div className="h-svh overflow-hidden flex flex-col">
      <WindowDragRegion />
      {children}
      <CommandPalette />
    </div>
  );
};
