import { generateNameFromId } from "@/lib/names";
import type { Thread, ThreadOverseer } from "@/types/thread";
import type { Project } from "@/types/project";

// Define projects
export const projects: Project[] = [
  {
    id: "1",
    name: "Flowgentic",
    description: "Core product development",
    color: "text-violet-400",
  },
  {
    id: "2",
    name: "Project Alpha",
    description: "Internal tooling and infrastructure",
    color: "text-cyan-400",
  },
  {
    id: "3",
    name: "Marketing Campaign",
    description: "Marketing automation and analytics",
    color: "text-amber-400",
  },
];

// Generate overseer for each thread using deterministic name generation
function createThreadOverseer(threadId: string): ThreadOverseer {
  return {
    id: `overseer-${threadId}`,
    name: generateNameFromId(`thread-${threadId}`),
  };
}

export const threads: Thread[] = [
  // Flowgentic project threads
  {
    id: "1",
    projectId: "1",
    mode: "plan",
    controlPlaneId: "cp-embedded",
    title: "User Authentication Flow",
    description: `Implement a **complete user authentication system** with the following features:

### Core Features
- **Login** — Email/password + OAuth providers (*Google*, *GitHub*)
- **Signup** — Email verification and strong password requirements
- **Password Reset** — Secure email tokens with expiration
- **Session Management** — JWT access tokens + refresh tokens

### Security Requirements
> The system must follow security best practices

1. Rate limiting on all auth endpoints
2. Password hashing with \`bcrypt\` (cost factor 12)
3. HTTPS-only cookies for tokens
4. CSRF protection on forms

See also: [OWASP Auth Guidelines](https://owasp.org)`,
    status: "in_progress",
    taskCount: 8,
    completedTasks: 5,
    createdAt: "2 hours ago",
    updatedAt: "5 min ago",
    overseer: createThreadOverseer("1"),
    memory: `## Decision Log

**2024-01-15** — Chose \`bcrypt\` over \`argon2\` for password hashing due to better library support in Node.js ecosystem.

**2024-01-14** — User requested OAuth support. Added Google and GitHub as initial providers. Apple Sign-In deferred to v2.

## Learnings
The login UI task had two competing implementations. GPT-4's version had better error handling but Claude's had cleaner code structure. Merged best of both.

## Links
- Design mockups: \`/docs/auth-designs.fig\`
- API spec: \`/docs/api/auth.yaml\``,
    resources: [
      {
        resourceId: "res-1",
        name: "Auth System PRD",
        type: "markdown",
        status: "approved",
        sharedWithAllTasks: true,
      },
      {
        resourceId: "res-2",
        name: "API Specification",
        type: "data",
        status: "approved",
        sharedWithAllTasks: true,
      },
      {
        resourceId: "res-3",
        name: "UI Mockups",
        type: "image",
        status: "approved",
        sharedWithAllTasks: false,
        restrictedToTaskIds: ["t5", "t6"],
      },
      {
        resourceId: "res-4",
        name: "User Model Schema",
        type: "code",
        status: "approved",
        sharedWithAllTasks: true,
      },
    ],
    vcs: {
      strategy: "stacked_branches",
      rootBranch: "main",
      totalBranches: 8,
      mergedBranches: 5,
      conflictedBranches: 0,
    },
  },
  {
    id: "2",
    projectId: "1",
    mode: "plan",
    controlPlaneId: "cp-embedded",
    title: "Dashboard Analytics",
    description: `Build a comprehensive analytics dashboard for monitoring key business metrics.

**Key Components:**
- Real-time visitor tracking with WebSocket updates
- Interactive charts (line, bar, pie) using **Recharts**
- Customizable date range filters
- Export functionality (PDF, CSV)

**Metrics to display:**
1. Daily/weekly/monthly active users
2. Conversion rates and funnel analysis
3. Revenue trends and projections
4. Geographic distribution of users`,
    status: "pending",
    taskCount: 12,
    completedTasks: 0,
    createdAt: "1 day ago",
    updatedAt: "1 day ago",
    overseer: createThreadOverseer("2"),
  },
  // Project Alpha threads
  {
    id: "3",
    projectId: "2",
    mode: "plan",
    controlPlaneId: "cp-remote-1",
    title: "Payment Integration",
    description: `Integrate **Stripe** for payment processing with full subscription support.

Features include:
- One-time payments and recurring subscriptions
- Multiple pricing tiers (Free, Pro, Enterprise)
- Automatic invoice generation
- Webhook handling for payment events
- Proration for plan upgrades/downgrades

> Note: All payment logic should be handled server-side for security.`,
    status: "completed",
    taskCount: 6,
    completedTasks: 6,
    createdAt: "3 days ago",
    updatedAt: "Yesterday",
    overseer: createThreadOverseer("3"),
    memory: `## Completion Summary

All 6 tasks completed successfully. Stripe integration is live in production.

### Key Decisions
1. Used Stripe Checkout for hosted payment pages (faster to implement)
2. Webhooks deployed to \`/api/webhooks/stripe\` with signature verification
3. Subscription data synced to local DB for faster queries

### Test Cards Used
- Success: \`4242 4242 4242 4242\`
- Decline: \`4000 0000 0000 0002\`

### Post-Launch Notes
- Monitor webhook failures in Stripe dashboard
- Set up alerts for failed payments > 3 retries`,
  },
  {
    id: "4",
    projectId: "2",
    mode: "plan",
    controlPlaneId: "cp-embedded",
    title: "Email Notification System",
    description: `Set up a transactional email system using **SendGrid** or **AWS SES**.

**Email Types:**
- Welcome emails for new signups
- Password reset confirmations
- Payment receipts and invoices
- Weekly digest notifications

Technical requirements:
- Template engine with \`handlebars\` for dynamic content
- Queue system (Redis/Bull) for reliable delivery
- Bounce and complaint handling`,
    status: "failed",
    taskCount: 5,
    completedTasks: 3,
    createdAt: "4 days ago",
    updatedAt: "2 days ago",
    overseer: createThreadOverseer("4"),
  },
  // Marketing Campaign threads
  {
    id: "5",
    projectId: "3",
    mode: "plan",
    controlPlaneId: "cp-embedded",
    title: "API Rate Limiting",
    description: `Implement robust rate limiting to protect API endpoints from abuse.

**Algorithm:** Token bucket with Redis backend

**Configuration:**
- \`100 requests/minute\` for authenticated users
- \`20 requests/minute\` for anonymous users
- Custom limits for specific endpoints (e.g., login: 5/min)

Response headers should include \`X-RateLimit-Remaining\` and \`X-RateLimit-Reset\`.`,
    status: "in_progress",
    taskCount: 4,
    completedTasks: 2,
    createdAt: "5 days ago",
    updatedAt: "3 hours ago",
    overseer: createThreadOverseer("5"),
    vcs: {
      strategy: "worktree",
      rootBranch: "develop",
      worktreeBaseDir: "../rate-limit-worktrees",
      totalBranches: 4,
      mergedBranches: 2,
      conflictedBranches: 0,
    },
  },
  {
    id: "6",
    projectId: "3",
    mode: "build",
    controlPlaneId: "cp-embedded",
    title: "User Settings Page",
    description: `Create a comprehensive user settings page with multiple sections:

- **Profile** - Name, avatar, bio
- **Account** - Email, password change, 2FA setup
- **Notifications** - Email preferences, push settings
- **Privacy** - Data export, account deletion
- **Billing** - Current plan, payment methods, invoices

All changes should save automatically with optimistic UI updates.`,
    status: "draft",
    taskCount: 0,
    completedTasks: 0,
    createdAt: "10 min ago",
    updatedAt: "10 min ago",
    overseer: createThreadOverseer("6"),
  },
];

// Legacy export for backwards compatibility
export const issues = threads;
