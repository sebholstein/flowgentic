import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { createFileRoute, Link } from "@tanstack/react-router";
import { Rocket } from "lucide-react";
import { useEffect, useRef } from "react";

export const Route = createFileRoute("/")({
  component: RootComponent,
});

function RootComponent() {

  const ref = useRef<HTMLVideoElement|null>(null);

  useEffect(() => {
    if (ref.current) {
      ref.current.playbackRate = 0.8;
    }
  }, []);

  return (<>
    <div className="w-full h-full min-h-0 overflow-hidden flex flex-col gap-4 items-center justify-center z-30">
    <FlowgenticLogo></FlowgenticLogo>
   <div className="flex gap-3 flex-col text-sm mt-8">
    <div className="flex gap-2 items-center text-muted-foreground"><Switch checked></Switch> Start Control Plane</div>
    <div className="flex gap-2 items-center text-muted-foreground"><Switch checked></Switch> Start Worker</div>
   </div>
    <Button size={"lg"} asChild className="mt-1"><Link to="/app"><Rocket className="size-4 mr-1" /> Lets go</Link></Button>
    </div>
    <video
      src="/bg-animated3.mp4"
      ref={ref}
      autoPlay
      muted
      loop
      className="fixed inset-0 -z-10 w-full h-full object-cover pointer-events-none blur-in-lg grayscale-100 opacity-10"
    />
  </>
  );
}

export function FlowgenticLogo() {
  return (
    <div className="flex items-center gap-8">

      <div className="flex flex-col">
        <span className="text-4xl font-semibold tracking-tight text-white">
          Flowgentic
        </span>
      </div>
    </div>
  );
}