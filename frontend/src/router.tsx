import { createRouter, createHashHistory } from "@tanstack/react-router";

// Import the generated route tree
import { routeTree } from "./routeTree.gen";

// Use hash history for Electron compatibility (file:// protocol)
const hashHistory = createHashHistory();

// Create a new router instance
export const getRouter = () => {
  const router = createRouter({
    routeTree,
    context: {},
    history: hashHistory,
    scrollRestoration: true,
    defaultPreloadStaleTime: 0,
  });

  return router;
};
