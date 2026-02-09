/**
 * CSS styles for Electron window dragging.
 * Apply dragStyle to header/titlebar containers.
 * Apply noDragStyle to all interactive children (buttons, links, inputs).
 */
export const dragStyle = { WebkitAppRegion: "drag" } as React.CSSProperties;
export const noDragStyle = { WebkitAppRegion: "no-drag" } as React.CSSProperties;

/**
 * WindowDragRegion - Not used. Apply dragStyle/noDragStyle directly to elements.
 */
export function WindowDragRegion() {
  return null;
}
