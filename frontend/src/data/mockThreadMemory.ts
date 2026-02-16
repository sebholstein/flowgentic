export const mockThreadMemory = {
  summary: `This thread coordinates the implementation of the authentication system across multiple tasks.

## Cross-Task Patterns

1. **Error Handling**: All tasks use the custom \`Result<T, E>\` type for consistent error propagation
2. **Testing**: Integration tests share a common \`TestDB\` fixture that seeds users and roles
3. **Naming**: API routes follow \`/api/v1/{resource}\` convention established in task 1

## Key Architectural Decisions

- **JWT over sessions**: Chose stateless JWT tokens for horizontal scalability
- **Refresh token rotation**: Each refresh generates a new refresh token, invalidating the old one
- **Password hashing**: bcrypt with cost factor 12 (balances security vs. login latency)

## Conflicts Resolved

Task 3 (User Model) and Task 4 (Login Endpoints) had overlapping changes to \`src/models/user.ts\`. Resolved by having Task 3 own the model definition and Task 4 extend it with auth-specific methods.`,

  learnings: [
    "The project uses a monorepo structure — shared types live in packages/shared, not duplicated per service",
    "CI pipeline requires all tasks to pass lint before merge, so each task branch must run prettier independently",
    "The existing auth middleware in /src/middleware/auth.ts is deprecated — new implementation should replace it entirely",
    "Team convention: all database migrations must be reversible (both up and down scripts required)",
  ],

  relatedDocs: [
    { title: "Authentication Architecture RFC", url: "https://docs.example.com/rfcs/auth-v2" },
    { title: "JWT Best Practices (OWASP)", url: "https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html" },
    { title: "Project API Conventions", path: "/docs/api-conventions.md" },
    { title: "Database Migration Guide", path: "/docs/migrations.md" },
  ],
};
