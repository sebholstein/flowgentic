import { createFileRoute, Outlet } from "@tanstack/react-router";

export const Route = createFileRoute("/app/threads/$threadId/tasks")({
  component: TasksLayout,
});

function TasksLayout() {
  return <Outlet />;
}
