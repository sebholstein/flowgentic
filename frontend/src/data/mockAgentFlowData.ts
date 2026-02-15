import type { ChatMessage } from "@/lib/session-event-mapper";
import type { Task, TaskPlan } from "@/types/task";

// --- Demo Thread Types ---

export type DemoMode = "quick" | "plan";
export type DemoPhase = "completed" | "planning" | "execution" | "creation";

export interface DemoThread {
  id: string;
  topic: string;
  mode: DemoMode;
  phase: DemoPhase;
  agents: DemoAgent[];
}

export interface DemoAgent {
  id: string;
  name: string;
  model: string;
  role: "overseer" | "planner";
  color: string;
}

export interface DemoPlan {
  agentId: string;
  agentName: string;
  agentModel: string;
  summary: string;
  approach: string;
  tasks: DemoPlanTask[];
  considerations: string[];
  estimatedComplexity: "low" | "medium" | "high";
  estimatedDuration?: string;
  affectedFiles?: string[];
}

export interface DemoPlanTask {
  id: string;
  name: string;
  description: string;
  dependencies: string[];
  estimatedDuration?: string;
  agent?: string;
  subtasks?: string[];
}

// --- File tree & changes for quick mode ---

export interface DemoFileNode {
  name: string;
  path: string;
  type: "file" | "directory";
  children?: DemoFileNode[];
  status?: "added" | "modified" | "deleted";
}

export interface DemoComment {
  id: string;
  lineNumber: number;
  side: "additions" | "deletions";
  author: string;
  content: string;
  timestamp: string;
  file: string;
}

export const demoFileTree: DemoFileNode[] = [
  {
    name: "src",
    path: "src",
    type: "directory",
    children: [
      {
        name: "components",
        path: "src/components",
        type: "directory",
        children: [
          { name: "Header.tsx", path: "src/components/Header.tsx", type: "file" },
          { name: "Footer.tsx", path: "src/components/Footer.tsx", type: "file" },
        ],
      },
      { name: "App.tsx", path: "src/App.tsx", type: "file" },
    ],
  },
  { name: "README.md", path: "README.md", type: "file", status: "modified" },
  { name: "CONTRIBUTING.md", path: "CONTRIBUTING.md", type: "file", status: "modified" },
  { name: "package.json", path: "package.json", type: "file" },
];

export const demoComments: DemoComment[] = [
  {
    id: "comment-1",
    lineNumber: 5,
    side: "additions",
    author: "You",
    content: "Nice catch on the possessive form!",
    timestamp: "2026-02-15T10:02:00Z",
    file: "README.md",
  },
];

export const demoPatchData: Record<string, string> = {
  "README.md": `--- a/README.md
+++ b/README.md
@@ -10,9 +10,9 @@
 ## Getting Started

 To get started with the project, follow these steps:
-1. Clone the repository and recieve the latest code
+1. Clone the repository and receive the latest code
 2. Install dependencies with \`npm install\`
-3. It's important to configure it's environment variables
+3. It's important to configure its environment variables
 4. Run the development server

 ## Features
@@ -25,7 +25,7 @@

 ## Contributing

-For more details, see the [contributing guide](https://example.com/old-link/contributing).
+For more details, see the [contributing guide](https://github.com/project/CONTRIBUTING.md).

 We welcome all contributions! Please read our guidelines before submitting.
`,
  "CONTRIBUTING.md": `--- a/CONTRIBUTING.md
+++ b/CONTRIBUTING.md
@@ -1,6 +1,6 @@
 # Contributing

-Thank you for your intrest in contributing!
+Thank you for your interest in contributing!

 ## How to Contribute

`,
};

// --- Demo Project ---

export const demoProject = {
  id: "demo-project",
  name: "AgentFlow Demo",
  color: "text-cyan-400",
};

// --- Demo Agents ---

const claudeOverseer: DemoAgent = {
  id: "agent-claude",
  name: "Claude",
  model: "claude-opus-4-6",
  role: "overseer",
  color: "bg-violet-500",
};

const claudePlanner: DemoAgent = {
  id: "agent-claude-planner",
  name: "Claude",
  model: "claude-sonnet-4-5-20250929",
  role: "planner",
  color: "bg-violet-500",
};

const gpt5Planner: DemoAgent = {
  id: "agent-gpt5",
  name: "GPT-5",
  model: "gpt-5",
  role: "planner",
  color: "bg-emerald-500",
};

// --- Available agents for thread creation demo ---

export interface DemoAvailableAgent {
  id: string;
  name: string;
  model: string;
  provider: string;
  description: string;
  color: string;
}

export const demoAvailableAgents: DemoAvailableAgent[] = [
  {
    id: "agent-claude-opus",
    name: "Claude",
    model: "claude-opus-4-6",
    provider: "Anthropic",
    description: "Advanced reasoning and planning",
    color: "bg-violet-500",
  },
  {
    id: "agent-claude-sonnet",
    name: "Claude",
    model: "claude-sonnet-4-5",
    provider: "Anthropic",
    description: "Fast and capable for most tasks",
    color: "bg-violet-500",
  },
  {
    id: "agent-gpt5",
    name: "GPT-5",
    model: "gpt-5",
    provider: "OpenAI",
    description: "Strong general-purpose coding agent",
    color: "bg-emerald-500",
  },
  {
    id: "agent-gemini",
    name: "Gemini",
    model: "gemini-2.5-pro",
    provider: "Google",
    description: "Large context with broad knowledge",
    color: "bg-blue-500",
  },
];

export interface DemoPlanTemplate {
  id: string;
  name: string;
  description: string;
  source: "repo" | "user" | "builtin";
  tags: string[];
}

export const demoPlanTemplates: DemoPlanTemplate[] = [
  {
    id: "tmpl-default",
    name: "Default",
    description: "Standard project conventions — build, lint, test verification with per-task branches",
    source: "repo",
    tags: ["build", "lint", "test", "per-task branches"],
  },
  {
    id: "tmpl-hotfix",
    name: "Hotfix",
    description: "Minimal process for urgent fixes — 1-2 tasks max, shared branch",
    source: "repo",
    tags: ["urgent", "shared branch"],
  },
  {
    id: "tmpl-spike",
    name: "Spike",
    description: "Exploratory work with relaxed verification — for prototyping and research",
    source: "user",
    tags: ["exploration", "prototype"],
  },
];

// --- Demo Threads ---

export const demoThreads: DemoThread[] = [
  {
    id: "demo-new-thread",
    topic: "New Thread",
    mode: "plan",
    phase: "creation",
    agents: [],
  },
  {
    id: "demo-quick-completed",
    topic: "Fix README typos",
    mode: "quick",
    phase: "completed",
    agents: [claudeOverseer],
  },
  {
    id: "demo-plan-single",
    topic: "Add dark mode support",
    mode: "plan",
    phase: "planning",
    agents: [claudeOverseer],
  },
  {
    id: "demo-plan-multi",
    topic: "Build payment system",
    mode: "plan",
    phase: "planning",
    agents: [claudeOverseer, claudePlanner, gpt5Planner],
  },
  {
    id: "demo-plan-execution",
    topic: "User authentication",
    mode: "plan",
    phase: "execution",
    agents: [claudeOverseer],
  },
];

// --- Messages per scenario ---

export const demoMessages: Record<string, ChatMessage[]> = {
  "demo-quick-completed": [
    {
      id: "msg-1",
      type: "user",
      content: "Fix the typos in README.md — there are a few grammatical errors and a broken link.",
      timestamp: "2026-02-15T10:00:00Z",
    },
    {
      id: "msg-2",
      type: "agent",
      content:
        "I'll fix the README typos now. Let me read the file and make corrections.\n\nI found 3 issues:\n1. \"recieve\" → \"receive\" (line 12)\n2. \"it's\" → \"its\" when used as possessive (line 28)\n3. Broken link to contributing guide — updated URL\n\nAll fixes have been applied and committed.",
      timestamp: "2026-02-15T10:00:15Z",
    },
    {
      id: "msg-3",
      type: "user",
      content: "Thanks, looks good!",
      timestamp: "2026-02-15T10:01:00Z",
    },
  ],

  "demo-plan-single": [
    {
      id: "msg-ps-1",
      type: "user",
      content:
        "Add dark mode support to the app. It should respect the system preference and allow manual toggle.",
      timestamp: "2026-02-15T10:00:00Z",
    },
    {
      id: "msg-ps-2",
      type: "agent",
      content:
        "I'll analyze the codebase and create a plan for adding dark mode support. Let me look at the current styling setup and component structure.\n\nI've drafted a plan — you can review it in the panel on the right. The approach uses CSS custom properties with a ThemeProvider context, and I've identified 4 tasks to complete the work.",
      timestamp: "2026-02-15T10:00:30Z",
    },
  ],

  // Multi-planner: each planner gets their own messages
  "demo-plan-multi-overseer": [
    {
      id: "msg-pm-o1",
      type: "user",
      content:
        "Build a payment processing system with Stripe integration, subscription management, and invoice generation.",
      timestamp: "2026-02-15T10:00:00Z",
    },
    {
      id: "msg-pm-o2",
      type: "agent",
      content:
        "This is a complex feature. I've assigned two planners to draft independent approaches — Claude and GPT-5. You can switch between their tabs to see each plan, or compare them side-by-side in the panel on the right.\n\nOnce you're happy with an approach, select the plan you'd like to proceed with.",
      timestamp: "2026-02-15T10:00:20Z",
    },
  ],
  "demo-plan-multi-claude": [
    {
      id: "msg-pm-c1",
      type: "agent",
      content:
        "I've analyzed the requirements. My approach focuses on a **modular service layer** — each concern (payments, subscriptions, invoices) gets its own service with a shared Stripe client. This keeps things testable and allows independent deployment.\n\nSee my detailed plan in the comparison panel.",
      timestamp: "2026-02-15T10:01:00Z",
    },
  ],
  "demo-plan-multi-gpt5": [
    {
      id: "msg-pm-g1",
      type: "agent",
      content:
        "I've reviewed the codebase and existing patterns. My plan uses an **event-driven architecture** with a central payment gateway that dispatches to handlers. This approach scales better for webhooks and async operations like invoice generation.\n\nCheck the comparison panel for the full breakdown.",
      timestamp: "2026-02-15T10:01:10Z",
    },
  ],

  "demo-plan-execution": [
    {
      id: "msg-pe-1",
      type: "user",
      content:
        "Implement user authentication with JWT tokens, refresh flow, and role-based access control.",
      timestamp: "2026-02-15T09:00:00Z",
    },
    {
      id: "msg-pe-2",
      type: "agent",
      content:
        "Plan approved! I'm now coordinating execution across the 5 tasks. Here's the current status:\n\n- **Auth middleware** is complete\n- **JWT token service** is complete\n- **User model & registration** is running now\n- **Login & refresh endpoints** are blocked on the user model\n- **RBAC middleware** is pending\n\nYou can track progress in the panel on the right.",
      timestamp: "2026-02-15T09:30:00Z",
    },
  ],
};

// --- Plans ---

export const demoPlans: Record<string, DemoPlan> = {
  "demo-plan-single": {
    agentId: "agent-claude",
    agentName: "Claude",
    agentModel: "claude-opus-4-6",
    summary:
      "Add system-aware dark mode with manual toggle using CSS custom properties and a React context provider.",
    approach:
      "Use CSS custom properties for theming, a ThemeProvider context for state management, and prefers-color-scheme media query for system detection. Store preference in localStorage.",
    tasks: [
      {
        id: "dm-task-1",
        name: "Create ThemeProvider context",
        description:
          "Build a React context that manages theme state (light/dark/system), persists to localStorage, and listens to prefers-color-scheme changes.",
        dependencies: [],
        estimatedDuration: "15 min",
        subtasks: [
          "Create ThemeContext with light/dark/system values",
          "Add localStorage persistence for theme preference",
          "Listen to prefers-color-scheme media query changes",
          "Export useTheme hook for consumers",
        ],
      },
      {
        id: "dm-task-2",
        name: "Define CSS custom properties",
        description:
          "Create CSS variables for all colors used across the app. Define light and dark value sets.",
        dependencies: [],
        estimatedDuration: "20 min",
        subtasks: [
          "Audit all hardcoded color values in the codebase",
          "Define --color-* CSS custom properties",
          "Create :root (light) and .dark (dark) variable sets",
          "Add transition property for smooth theme switching",
        ],
      },
      {
        id: "dm-task-3",
        name: "Update components to use theme variables",
        description:
          "Replace hardcoded colors in all components with CSS custom property references.",
        dependencies: ["dm-task-2"],
        estimatedDuration: "30 min",
        subtasks: [
          "Update Header component colors",
          "Update Sidebar component colors",
          "Update Card/Panel background colors",
          "Update text and border colors globally",
          "Verify contrast ratios in both themes",
        ],
      },
      {
        id: "dm-task-4",
        name: "Add theme toggle UI",
        description:
          "Create a toggle button in the header/settings that cycles between light, dark, and system themes.",
        dependencies: ["dm-task-1"],
        estimatedDuration: "10 min",
        subtasks: [
          "Create ThemeToggle component with sun/moon/monitor icons",
          "Wire up to ThemeProvider context",
          "Add to header navigation bar",
        ],
      },
    ],
    considerations: [
      "Ensure smooth transitions when switching themes",
      "Handle SSR/hydration mismatch for system preference",
      "Test with forced-colors and high-contrast modes",
    ],
    estimatedComplexity: "medium",
    estimatedDuration: "~1.5 hours",
    affectedFiles: [
      "src/contexts/ThemeContext.tsx",
      "src/styles/variables.css",
      "src/components/Header.tsx",
      "src/components/Sidebar.tsx",
      "src/components/ThemeToggle.tsx",
    ],
  },

  "demo-plan-multi-claude": {
    agentId: "agent-claude-planner",
    agentName: "Claude",
    agentModel: "claude-sonnet-4-5-20250929",
    summary:
      "Modular service architecture with dedicated services for payments, subscriptions, and invoices sharing a Stripe client.",
    approach:
      "Create independent service modules (PaymentService, SubscriptionService, InvoiceService) that share a configured Stripe client. Each service handles its own webhook events. Use a repository pattern for data persistence.\n\nThis approach prioritizes testability and separation of concerns. Each service can be developed and tested independently. The shared Stripe client avoids redundant initialization and ensures consistent configuration.",
    tasks: [
      {
        id: "pm-c-task-1",
        name: "Stripe client & config",
        description: "Set up Stripe SDK, environment config, and webhook signature verification.",
        dependencies: [],
        estimatedDuration: "20 min",
        agent: "Claude",
        subtasks: [
          "Install and configure Stripe SDK",
          "Create StripeClient singleton with env-based config",
          "Implement webhook signature verification middleware",
          "Add Stripe-related environment variables to .env.example",
        ],
      },
      {
        id: "pm-c-task-2",
        name: "Payment service",
        description:
          "Create payment intents, handle confirmations, manage refunds. Includes webhook handler for payment events.",
        dependencies: ["pm-c-task-1"],
        estimatedDuration: "45 min",
        agent: "Claude",
        subtasks: [
          "Create PaymentService class with createIntent, confirm, refund methods",
          "Implement PaymentRepository for persistence",
          "Add payment webhook handler (payment_intent.succeeded, failed, etc.)",
          "Write unit tests for PaymentService",
        ],
      },
      {
        id: "pm-c-task-3",
        name: "Subscription service",
        description:
          "Manage subscription lifecycle — create, upgrade, downgrade, cancel. Handle trial periods.",
        dependencies: ["pm-c-task-1"],
        estimatedDuration: "45 min",
        agent: "Claude",
        subtasks: [
          "Create SubscriptionService with CRUD + lifecycle methods",
          "Implement plan change logic (upgrade/downgrade proration)",
          "Handle trial period management",
          "Add subscription webhook handlers",
          "Write unit tests",
        ],
      },
      {
        id: "pm-c-task-4",
        name: "Invoice service",
        description:
          "Generate invoices from payment/subscription events. PDF generation and email delivery.",
        dependencies: ["pm-c-task-2", "pm-c-task-3"],
        estimatedDuration: "30 min",
        agent: "Claude",
        subtasks: [
          "Create InvoiceService with generate, send, void methods",
          "Implement PDF generation with template",
          "Add email delivery integration",
          "Listen to payment/subscription completion events",
        ],
      },
      {
        id: "pm-c-task-5",
        name: "API routes & middleware",
        description:
          "REST endpoints for all payment operations. Auth middleware for Stripe webhook validation.",
        dependencies: ["pm-c-task-2", "pm-c-task-3"],
        estimatedDuration: "25 min",
        agent: "Claude",
        subtasks: [
          "Create POST /payments/intents endpoint",
          "Create GET/POST /subscriptions endpoints",
          "Create GET /invoices endpoint",
          "Create POST /webhooks/stripe endpoint",
          "Add request validation middleware",
          "Write integration tests",
        ],
      },
    ],
    considerations: [
      "Idempotency keys for payment operations — use client-provided keys stored in DB",
      "Webhook retry handling — deduplicate by event ID, process idempotently",
      "PCI compliance — never store raw card data, use Stripe tokens exclusively",
      "Error handling — distinguish between retriable and terminal payment failures",
      "Testing — use Stripe test mode and mock webhooks for CI",
    ],
    estimatedComplexity: "high",
    estimatedDuration: "~3 hours",
    affectedFiles: [
      "src/services/stripe/StripeClient.ts",
      "src/services/payment/PaymentService.ts",
      "src/services/subscription/SubscriptionService.ts",
      "src/services/invoice/InvoiceService.ts",
      "src/routes/payments.ts",
      "src/routes/subscriptions.ts",
      "src/routes/webhooks.ts",
      "src/middleware/stripeWebhook.ts",
    ],
  },

  "demo-plan-multi-gpt5": {
    agentId: "agent-gpt5",
    agentName: "GPT-5",
    agentModel: "gpt-5",
    summary:
      "Event-driven payment gateway with centralized event bus dispatching to specialized handlers.",
    approach:
      "Build a PaymentGateway that normalizes all Stripe interactions behind an event bus. Handlers subscribe to events (payment.created, subscription.updated, etc.) and react independently. This decouples concerns and makes webhook handling natural.\n\nThe event-driven approach naturally maps to Stripe's webhook model — incoming webhooks become internal events that flow through the system. This makes the architecture inherently scalable and easier to extend with new payment providers.",
    tasks: [
      {
        id: "pm-g-task-1",
        name: "Event bus infrastructure",
        description:
          "Create an in-process event bus with typed events, subscriber registration, and async dispatch.",
        dependencies: [],
        estimatedDuration: "30 min",
        agent: "GPT-5",
        subtasks: [
          "Define PaymentEvent union type with all event variants",
          "Create EventBus class with subscribe/publish/unsubscribe",
          "Add async dispatch with error isolation per handler",
          "Implement event logging and replay capability",
          "Write comprehensive unit tests",
        ],
      },
      {
        id: "pm-g-task-2",
        name: "Stripe gateway adapter",
        description:
          "Wrap Stripe SDK behind a gateway interface. Convert Stripe webhooks to internal events.",
        dependencies: ["pm-g-task-1"],
        estimatedDuration: "35 min",
        agent: "GPT-5",
        subtasks: [
          "Define PaymentGateway interface (provider-agnostic)",
          "Implement StripeGateway adapter",
          "Create webhook-to-event mapper",
          "Add signature verification",
          "Support future payment provider swaps",
        ],
      },
      {
        id: "pm-g-task-3",
        name: "Payment handler",
        description:
          "Subscribe to payment events. Process charges, refunds, disputes. Emit completion events.",
        dependencies: ["pm-g-task-1"],
        estimatedDuration: "40 min",
        agent: "GPT-5",
        subtasks: [
          "Create PaymentHandler subscribing to payment.* events",
          "Implement charge processing with idempotency",
          "Handle refund flow with partial refund support",
          "Process dispute notifications",
          "Emit payment.completed / payment.failed events",
        ],
      },
      {
        id: "pm-g-task-4",
        name: "Subscription handler",
        description:
          "Subscribe to subscription events. Manage lifecycle, billing cycles, and plan changes.",
        dependencies: ["pm-g-task-1"],
        estimatedDuration: "40 min",
        agent: "GPT-5",
        subtasks: [
          "Create SubscriptionHandler subscribing to subscription.* events",
          "Handle create, update, cancel lifecycle",
          "Manage billing cycle transitions",
          "Support plan upgrades/downgrades with proration",
          "Emit subscription.activated / subscription.cancelled events",
        ],
      },
      {
        id: "pm-g-task-5",
        name: "Invoice handler",
        description:
          "Subscribe to payment and subscription completion events. Auto-generate and deliver invoices.",
        dependencies: ["pm-g-task-3", "pm-g-task-4"],
        estimatedDuration: "25 min",
        agent: "GPT-5",
        subtasks: [
          "Create InvoiceHandler subscribing to *.completed events",
          "Auto-generate invoice from event payload",
          "Generate PDF with line items",
          "Queue email delivery",
        ],
      },
      {
        id: "pm-g-task-6",
        name: "API layer",
        description:
          "REST endpoints that dispatch commands through the gateway. Webhook endpoint for Stripe.",
        dependencies: ["pm-g-task-2"],
        estimatedDuration: "20 min",
        agent: "GPT-5",
        subtasks: [
          "Create thin API routes that dispatch to gateway",
          "Implement POST /webhooks/stripe with event conversion",
          "Add request validation",
          "Write end-to-end tests",
        ],
      },
    ],
    considerations: [
      "Event ordering guarantees — use sequence numbers for dependent operations",
      "Dead letter queue for failed event processing — store and retry with backoff",
      "Saga pattern for multi-step transactions — compensating actions on failure",
      "Observability — structured logging for every event publish/handle",
      "Extensibility — new handlers can be added without modifying existing code",
    ],
    estimatedComplexity: "high",
    estimatedDuration: "~3.5 hours",
    affectedFiles: [
      "src/events/EventBus.ts",
      "src/events/types.ts",
      "src/gateway/PaymentGateway.ts",
      "src/gateway/StripeGateway.ts",
      "src/handlers/PaymentHandler.ts",
      "src/handlers/SubscriptionHandler.ts",
      "src/handlers/InvoiceHandler.ts",
      "src/routes/api.ts",
      "src/routes/webhooks.ts",
    ],
  },
};

// --- Execution tasks (for scenario 4) ---

export const demoExecutionTasks: Task[] = [
  {
    id: "exec-task-1",
    name: "Auth middleware",
    description: "Create Express middleware for JWT verification and route protection.",
    status: "completed",
    duration: "2m 15s",
    dependencies: [],
    agent: "Claude",
    planStatus: "approved",
    plan: {
      summary: "JWT verification middleware with role extraction",
      steps: [
        "Create verifyToken middleware",
        "Extract user roles from JWT payload",
        "Add to Express app",
      ],
      agentId: "agent-claude",
      agentName: "Claude",
      createdAt: "2026-02-15T09:05:00Z",
    },
  },
  {
    id: "exec-task-2",
    name: "JWT token service",
    description:
      "Implement token generation, verification, and refresh logic with configurable expiry.",
    status: "completed",
    duration: "3m 42s",
    dependencies: [],
    agent: "Claude",
    planStatus: "approved",
  },
  {
    id: "exec-task-3",
    name: "User model & registration",
    description: "Create user schema, password hashing, and registration endpoint.",
    status: "running",
    dependencies: [],
    agent: "Claude",
    subtasks: [
      { id: "st-1", name: "Define User schema with Prisma", completed: true },
      { id: "st-2", name: "Implement bcrypt password hashing", completed: true },
      { id: "st-3", name: "Create POST /auth/register endpoint", completed: false },
      { id: "st-4", name: "Add input validation", completed: false },
    ],
    planStatus: "approved",
  },
  {
    id: "exec-task-4",
    name: "Login & refresh endpoints",
    description:
      "Build login endpoint with credential verification, and token refresh endpoint with rotation.",
    status: "blocked",
    dependencies: ["exec-task-3"],
    agent: "Claude",
    planStatus: "approved",
  },
  {
    id: "exec-task-5",
    name: "RBAC middleware",
    description: "Role-based access control middleware with permission checking.",
    status: "pending",
    dependencies: ["exec-task-1", "exec-task-4"],
    agent: "Claude",
    planStatus: "approved",
  },
];

// --- Helpers ---

export function getDemoThread(scenarioId: string): DemoThread | undefined {
  return demoThreads.find((t) => t.id === scenarioId);
}

export function getDemoMessages(scenarioId: string): ChatMessage[] {
  return demoMessages[scenarioId] ?? [];
}

export function getDemoPlan(planKey: string): DemoPlan | undefined {
  return demoPlans[planKey];
}

export function getDemoPlanAsTaskPlan(planKey: string): TaskPlan | undefined {
  const plan = demoPlans[planKey];
  if (!plan) return undefined;
  return {
    summary: plan.summary,
    steps: plan.tasks.map((t) => t.name),
    approach: plan.approach,
    considerations: plan.considerations,
    estimatedComplexity: plan.estimatedComplexity,
    agentId: plan.agentId,
    agentName: plan.agentName,
    createdAt: "2026-02-15T10:00:00Z",
  };
}
