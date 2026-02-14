import { useState, useEffect, useRef } from "react";

/**
 * Animates text changes with a typewriter effect.
 * When `target` changes, removes old text char-by-char, then types new text.
 */
export function useTypewriter(target: string, speed = 30): string {
  const [display, setDisplay] = useState(target);
  const prevTarget = useRef(target);

  useEffect(() => {
    if (target === prevTarget.current) return;

    const old = prevTarget.current;
    prevTarget.current = target;

    let cancelled = false;

    (async () => {
      // Remove old text char by char
      for (let i = old.length; i >= 0; i--) {
        if (cancelled) return;
        setDisplay(old.slice(0, i));
        await sleep(speed / 2);
      }

      // Type new text char by char
      for (let i = 1; i <= target.length; i++) {
        if (cancelled) return;
        setDisplay(target.slice(0, i));
        await sleep(speed);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [target, speed]);

  return display;
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
