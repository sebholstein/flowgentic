import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useNavigate } from "@tanstack/react-router";

interface LoginFormProps {
  heading?: string;
  logo?: {
    url?: string;
    src: string;
    alt: string;
    title?: string;
  };
  buttonText?: string;
  signupText?: string;
  signupUrl?: string;
  className?: string;
}

const LoginForm = ({
  heading = "Login",
  logo = {
    src: "/logo192.png",
    alt: "Flowgentic",
    title: "Flowgentic",
  },
  buttonText = "Login",
  signupText = "Need an account?",
  signupUrl = "#",
  className,
}: LoginFormProps) => {
  const navigate = useNavigate();

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    navigate({ to: "/app/threads" });
  };

  return (
    <section className={cn("h-screen bg-background", className)}>
      <div className="flex h-full items-center justify-center">
        <div className="flex flex-col items-center gap-6 lg:justify-start">
          <form
            onSubmit={handleSubmit}
            className="flex w-full max-w-sm min-w-sm flex-col items-center gap-y-4 px-6 py-12"
          >
            {logo && (
              <a href={logo.url}>
                <img src={logo.src} alt={logo.alt} title={logo.title} className="h-10" />
              </a>
            )}
            {heading && <h1 className="text-2xl font-semibold">{heading}</h1>}
            <div className="flex w-full flex-col gap-2">
              <Label htmlFor="email">Email</Label>
              <Input id="email" type="text" placeholder="Email" className="text-sm" />
            </div>
            <div className="flex w-full flex-col gap-2">
              <Label htmlFor="password">Password</Label>
              <Input id="password" type="password" placeholder="Password" className="text-sm" />
            </div>
            <Button type="submit" className="w-full">
              {buttonText}
            </Button>
          </form>
          <div className="flex justify-center gap-1 text-sm text-muted-foreground">
            <p>{signupText}</p>
            <a href={signupUrl} className="font-medium text-primary hover:underline">
              Sign up
            </a>
          </div>
        </div>
      </div>
    </section>
  );
};

export { LoginForm };
