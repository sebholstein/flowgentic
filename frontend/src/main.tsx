import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { RouterProvider } from "@tanstack/react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { getRouter } from "./router";
import "./styles.css";

const router = getRouter();
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // Discard inactive query cache after 30s (default: 5 min).
      // With aggressive polling this prevents stale protobuf response
      // objects from piling up in memory.
      gcTime: 30_000,
      // Treat data as fresh for 5s — prevents re-renders when a poll
      // returns identical data and structural sharing produces the same ref.
      staleTime: 5_000,
    },
  },
});

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  </StrictMode>,
);
