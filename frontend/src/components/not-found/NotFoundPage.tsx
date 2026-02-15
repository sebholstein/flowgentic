import { Link, useNavigate, useRouterState } from "@tanstack/react-router";
import { ArrowLeft, LayoutGrid, MessageCircle, MessageSquarePlus, Settings } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

export function NotFoundPage() {
  const navigate = useNavigate();
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const pathLabel = pathname ?? "/";

  const handleBack = () => {
    if (window.history.length > 1) {
      window.history.back();
      return;
    }
    navigate({ to: "/app/threads" });
  };

  return (
    <main className="relative min-h-svh overflow-hidden bg-background text-foreground">
      <div className="pointer-events-none absolute inset-0">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_15%_10%,_var(--chart-2)_0%,_transparent_55%)] opacity-30" />
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_85%_15%,_var(--primary)_0%,_transparent_60%)] opacity-25" />
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_50%_90%,_var(--chart-4)_0%,_transparent_60%)] opacity-25" />
        <div
          className="absolute inset-0 bg-[linear-gradient(90deg,_var(--border)_1px,_transparent_1px),_linear-gradient(180deg,_var(--border)_1px,_transparent_1px)] opacity-15"
          style={{ backgroundSize: "48px 48px" }}
        />
      </div>

      <div className="relative mx-auto flex min-h-svh w-full max-w-5xl flex-col justify-center gap-10 px-6 py-16">
        <div className="flex items-center gap-3">
          <Badge variant="outline" className="uppercase tracking-[0.35em] text-[0.55rem] px-3">
            Signal Lost
          </Badge>
          <span className="text-xs text-muted-foreground">404</span>
        </div>

        <div className="grid gap-10 lg:grid-cols-[1.2fr_0.8fr] lg:items-center">
          <div className="space-y-6">
            <div className="space-y-4">
              <p className="text-[clamp(3.5rem,16vw,9rem)] font-black leading-none tracking-[-0.04em] text-transparent bg-clip-text bg-[linear-gradient(130deg,_var(--foreground)_0%,_var(--primary)_45%,_var(--chart-2)_100%)]">
                404
              </p>
              <div className="space-y-2">
                <h1 className="text-2xl font-semibold sm:text-3xl">
                  We could not locate that route.
                </h1>
                <p className="text-sm text-muted-foreground max-w-xl">
                  The path
                  <span className="mx-2 inline-block max-w-full break-all rounded-full border border-border/60 bg-card/80 px-2 py-0.5 font-mono text-[0.65rem] text-foreground/90">
                    {pathLabel}
                  </span>
                  does not exist in this build. Try one of the destinations below or return to the
                  previous view.
                </p>
              </div>
            </div>

            <div className="flex flex-wrap gap-3">
              <Button onClick={handleBack} variant="secondary" size="lg" className="gap-2">
                <ArrowLeft className="size-4" />
                Go Back
              </Button>
              <Button asChild size="lg" className="gap-2">
                <Link to="/app/threads">
                  <MessageCircle className="size-4" />
                  Threads
                </Link>
              </Button>
              <Button asChild size="lg" variant="outline" className="gap-2">
                <Link to="/app/tasks">
                  <LayoutGrid className="size-4" />
                  Tasks
                </Link>
              </Button>
            </div>
          </div>

          <div className="space-y-4">
            <div className="rounded-2xl border border-border/70 bg-card/80 p-4 shadow-sm backdrop-blur">
              <div className="flex items-start justify-between gap-3">
                <div className="space-y-1">
                  <p className="text-xs uppercase tracking-[0.3em] text-muted-foreground">Status</p>
                  <p className="text-lg font-semibold">Route not found</p>
                </div>
                <Badge variant="secondary">404</Badge>
              </div>
              <div className="mt-4 grid gap-2 text-xs text-muted-foreground">
                <div className="flex items-center justify-between border-b border-border/50 pb-2">
                  <span>Requested</span>
                  <span className="max-w-[60%] truncate font-mono text-[0.65rem] text-foreground/80">
                    {pathLabel}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span>Next step</span>
                  <span className="text-foreground/80">Choose a destination</span>
                </div>
              </div>
            </div>

            <div className="grid gap-3 sm:grid-cols-2">
              <Link
                to="/app/settings"
                className="group rounded-2xl border border-border/70 bg-background/70 p-4 transition hover:-translate-y-0.5 hover:bg-card/90"
              >
                <div className="flex items-center gap-3">
                  <span className="flex size-9 items-center justify-center rounded-xl border border-border/60 bg-muted/50">
                    <Settings className="size-4 text-muted-foreground group-hover:text-foreground" />
                  </span>
                  <div className="space-y-1">
                    <p className="text-sm font-semibold">Settings</p>
                    <p className="text-xs text-muted-foreground">
                      Check configuration and routing.
                    </p>
                  </div>
                </div>
              </Link>
              <Link
                to="/app/threads"
                className="group rounded-2xl border border-border/70 bg-background/70 p-4 transition hover:-translate-y-0.5 hover:bg-card/90"
              >
                <div className="flex items-center gap-3">
                  <span className="flex size-9 items-center justify-center rounded-xl border border-border/60 bg-muted/50">
                    <MessageSquarePlus className="size-4 text-muted-foreground group-hover:text-foreground" />
                  </span>
                  <div className="space-y-1">
                    <p className="text-sm font-semibold">Start a thread</p>
                    <p className="text-xs text-muted-foreground">Kick off a new workflow.</p>
                  </div>
                </div>
              </Link>
            </div>
          </div>
        </div>
      </div>
    </main>
  );
}
