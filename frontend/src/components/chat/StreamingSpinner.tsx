import { useState, useRef, useEffect, memo } from "react";
import { Loader2 } from "lucide-react";

export const StreamingSpinner = memo(function StreamingSpinner() {
  const [elapsed, setElapsed] = useState(0);
  const startRef = useRef(Date.now());

  useEffect(() => {
    const interval = setInterval(() => {
      setElapsed(Date.now() - startRef.current);
    }, 100);
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="flex items-center gap-1.5">
      <Loader2 className="size-4 animate-spin text-muted-foreground" />
      <span className="text-[12px] font-mono text-muted-foreground tabular-nums">
        {(elapsed / 1000).toFixed(1)}s
      </span>
    </div>
  );
});
