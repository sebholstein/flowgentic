import { useState, useCallback, useRef } from "react";

interface UseResizePanelOptions {
  initial?: number;
  min?: number;
  max?: number;
}

export function useResizePanel({
  initial = 55,
  min = 30,
  max = 70,
}: UseResizePanelOptions = {}) {
  const [percent, setPercent] = useState(initial);
  const containerRef = useRef<HTMLDivElement>(null);

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      const startX = e.clientX;
      const startPercent = percent;
      const containerWidth = containerRef.current?.offsetWidth ?? 1;

      const handleMouseMove = (moveEvent: MouseEvent) => {
        const deltaX = moveEvent.clientX - startX;
        const deltaPercent = (deltaX / containerWidth) * 100;
        setPercent(Math.min(max, Math.max(min, startPercent + deltaPercent)));
      };

      const handleMouseUp = () => {
        document.removeEventListener("mousemove", handleMouseMove);
        document.removeEventListener("mouseup", handleMouseUp);
      };

      document.addEventListener("mousemove", handleMouseMove);
      document.addEventListener("mouseup", handleMouseUp);
    },
    [percent, min, max],
  );

  return { percent, handleMouseDown, containerRef };
}
