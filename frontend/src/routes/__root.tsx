import { Outlet, createRootRoute } from "@tanstack/react-router";
import { ThemeProvider } from "@/components/theme-provider";
import { ConnectProvider } from "@/components/ConnectProvider";
import { NotFoundPage } from "@/components/not-found/NotFoundPage";

export const Route = createRootRoute({
  component: RootComponent,
  notFoundComponent: NotFoundPage,
});

function RootComponent() {
  return (
    <ThemeProvider defaultTheme="dark">
      <ConnectProvider>
        <Outlet />
      </ConnectProvider>
    </ThemeProvider>
  );
}
