// Mock test results for demo
export const mockTestResults = [
  {
    name: "stripe.test.ts › initializes Stripe client correctly",
    status: "passed",
    duration: "12ms",
  },
  {
    name: "stripe.test.ts › handles missing API key gracefully",
    status: "passed",
    duration: "8ms",
  },
  { name: "stripe.test.ts › validates API version format", status: "passed", duration: "5ms" },
  {
    name: "stripe-client.test.ts › loads Stripe.js asynchronously",
    status: "passed",
    duration: "45ms",
  },
  { name: "stripe-client.test.ts › memoizes Stripe promise", status: "passed", duration: "3ms" },
];

// Mock output log for demo
export const mockOutputLog = `[14:32:05] Starting task: Stripe SDK Setup
[14:32:05] Agent: claude-opus-4 (claude-opus-4-5-20251101)
[14:32:06] Reading package.json...
[14:32:06] Found existing dependencies: react, @tanstack/react-router
[14:32:07] Installing stripe and @stripe/stripe-js...
[14:32:15] Successfully installed dependencies
[14:32:16] Creating src/lib/stripe.ts
[14:32:18] Creating src/lib/stripe-client.ts
[14:32:20] Updating .env.example with Stripe variables
[14:32:22] Running type check...
[14:32:28] Type check passed
[14:32:29] Running tests...
[14:32:35] All tests passed (5/5)
[14:33:10] Task completed successfully
[14:33:10] Duration: 1m 05s
[14:33:10] Tokens used: 12,450 (input: 8,200, output: 4,250)`;

export interface ProgressStep {
  id: string;
  name: string;
  status: "completed" | "running" | "pending";
  detail: string;
}

// Mock progress steps for the agent
export const mockProgressSteps: ProgressStep[] = [
  {
    id: "1",
    name: "Analyze task requirements",
    status: "completed",
    detail: "Read task description and dependencies",
  },
  {
    id: "2",
    name: "Check existing codebase",
    status: "completed",
    detail: "Scanned package.json and project structure",
  },
  {
    id: "3",
    name: "Install dependencies",
    status: "completed",
    detail: "Added stripe and @stripe/stripe-js packages",
  },
  {
    id: "4",
    name: "Create server-side client",
    status: "completed",
    detail: "Created src/lib/stripe.ts with API configuration",
  },
  {
    id: "5",
    name: "Create client-side loader",
    status: "completed",
    detail: "Created src/lib/stripe-client.ts for browser",
  },
  {
    id: "6",
    name: "Update environment config",
    status: "completed",
    detail: "Added Stripe keys to .env.example",
  },
  {
    id: "7",
    name: "Run type checking",
    status: "completed",
    detail: "Verified TypeScript compiles without errors",
  },
  { id: "8", name: "Run tests", status: "completed", detail: "All 5 tests passed" },
];

// For running tasks, show partial progress
export const mockRunningProgressSteps: ProgressStep[] = [
  {
    id: "1",
    name: "Analyze task requirements",
    status: "completed",
    detail: "Read task description and dependencies",
  },
  {
    id: "2",
    name: "Check existing codebase",
    status: "completed",
    detail: "Scanned package.json and project structure",
  },
  { id: "3", name: "Install dependencies", status: "completed", detail: "Added required packages" },
  {
    id: "4",
    name: "Create form component",
    status: "running",
    detail: "Building signup form with validation...",
  },
  {
    id: "5",
    name: "Add form validation",
    status: "pending",
    detail: "Implement Zod schema validation",
  },
  { id: "6", name: "Style components", status: "pending", detail: "Apply Tailwind CSS styles" },
  { id: "7", name: "Run tests", status: "pending", detail: "Execute component tests" },
];
