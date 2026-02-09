import { Bot, GitBranch } from "lucide-react";

export type NavModule = {
  id: string;
  label: string;
  icon: React.ComponentType<React.SVGProps<SVGSVGElement>>;
  href: string;
};

export const navModules: NavModule[] = [
  { id: "threads", label: "Threads", icon: GitBranch, href: "/app/threads" },
  { id: "overseer", label: "Project Overseer", icon: Bot, href: "/app/overseer" },
];

export const userData = {
  name: "Jordan Lee",
  email: "jordan@acme.io",
  avatar: "https://deifkwefumgah.cloudfront.net/shadcnblocks/block/avatar/avatar1.webp",
};

export function getInitials(name: string) {
  return (
    name
      .split(" ")
      .filter(Boolean)
      .slice(0, 2)
      .map((part) => part[0]?.toUpperCase())
      .join("") || "U"
  );
}
