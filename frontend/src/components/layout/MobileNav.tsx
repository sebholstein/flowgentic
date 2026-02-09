import { cn } from "@/lib/utils";
import { Link } from "@tanstack/react-router";
import { navModules } from "./nav-config";

export function MobileNav() {
  return (
    <nav className="fixed inset-x-0 bottom-0 z-40 border-t bg-background/95 backdrop-blur md:hidden">
      <div className="grid grid-cols-2">
        {navModules.map((moduleItem) => {
          const Icon = moduleItem.icon;
          return (
            <Link
              key={moduleItem.id}
              to={moduleItem.href}
              className={cn(
                "flex flex-col items-center gap-1 py-2 text-xs text-muted-foreground hover:text-foreground",
                "[&.active]:text-foreground",
              )}
              aria-label={moduleItem.label}
            >
              <Icon className="size-5" />
              <span>{moduleItem.label}</span>
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
