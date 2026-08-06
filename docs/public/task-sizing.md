# Task Sizing Guide

When is spec-driven development with autospec the right choice? This guide helps you decide whether a task is too small (just do it), optimal (use autospec), or too large (split first).

## Quick Decision Guide

```
Can you describe it in one sentence AND finish in < 30 min?
  → YES: Skip autospec, just do it
  → NO: Continue...

Does it involve 3+ design decisions AND touch 3+ files?
  → YES: Use autospec
  → NO: Probably skip autospec

Would it take > 2 days OR have 3+ independent features?
  → YES: Split first, then run autospec on each part
```

## Too Small — Just Do It

**Criteria**: Zero design decisions, implementation is obvious, < 30 minutes

These tasks have clear scope and obvious solutions. Running the full specify → plan → tasks → implement workflow adds more overhead than value.

| Example | Why skip autospec |
|---------|-------------------|
| Fix typo: `"recieve"` → `"receive"` | Single line, no decisions |
| Add missing `import "fmt"` | Mechanical fix |
| Bump dependency `v1.2.2` → `v1.2.3` | Config change |
| Increase timeout from 30s to 60s | Config tweak |
| Remove unused function | Deletion, no design |
| Add `--quiet` flag (just wraps existing prints) | Trivial logic, single file |
| Fix nil pointer by adding nil check | Obvious bug fix |
| Update regex pattern for validation | Pattern replacement |
| Add log statement for debugging | Single line |
| Rename variable via IDE refactor | Tool handles it better |

**Rule of thumb**: If you can describe the change in one sentence and implement it in one sitting without thinking about architecture, skip autospec.

## Just Right — Use autospec

**Criteria**: 2-5 design decisions, 3-10 files, 2-8 hours of work, 3-15 tasks

These tasks have clear boundaries but require design decisions. The spec workflow prevents scope creep and ensures nothing is missed.

| Example | Key decisions autospec helps with |
|---------|-----------------------------------|
| Add rate limiting to API | Algorithm (token bucket vs sliding window), storage, bypass rules, error format |
| Create `history` command with filtering | Flag design, output format, filter syntax, pagination |
| Add Redis caching layer | Key strategy, TTL policy, invalidation, cache miss handling |
| Implement CSV/JSON export | Format detection, streaming vs buffered, progress, large file handling |
| Add webhook delivery system | Signature scheme, retry policy, timeout handling, failure notifications |
| Build retry mechanism with backoff | Max attempts, backoff algorithm, circuit breaker, state persistence |
| Add pagination to list endpoints | Cursor vs offset, page size limits, sort options, total count |
| Create data migration v1→v2 | Transformation logic, rollback, validation, progress tracking |

**Sweet spot indicators**:
- 3-15 tasks when broken down
- Touches 3-10 files
- Has 2-5 non-trivial decisions to make
- Requires both implementation and test code
- Can be completed in one focused day

**The "decision density" test**: Count design decisions the task requires (algorithm choice, error handling strategy, API surface, code location, test approach, performance considerations). If you count 3+, autospec helps ensure you address them all.

## Too Large — Split First

**Warning signs**:
- Description has 3+ "and"s connecting distinct features
- Would take > 2 days
- Would generate > 20 tasks
- Multiple subsystems that could ship independently
- Different people would naturally own different parts

These are "projects within projects." Running them as a single spec leads to sprawling plans, incomplete coverage, and integration failures.

| Too Large | Split Into |
|-----------|------------|
| "Add OAuth with Google, GitHub, SAML, LDAP" | 1: OAuth core + Google, 2: GitHub, 3: SAML, 4: LDAP |
| "Build notification system with email, SMS, push, in-app" | 1: Core + in-app, 2: Email, 3: SMS, 4: Push |
| "Create admin dashboard" | 1: Layout + auth, 2: User management, 3: Analytics, 4: Settings |
| "Add multi-tenancy" | 1: Tenant model, 2: Data isolation, 3: Per-tenant config, 4: Billing |
| "Implement plugin system" | 1: Interface + loader, 2: Lifecycle hooks, 3: Sandbox, 4: Discovery |
| "Rewrite frontend in React" | One spec per feature area: auth, dashboard, settings, reports |

### Splitting Strategies

- **By layer**: Infrastructure → Core logic → UI → Polish
- **By feature slice**: One provider/channel/integration at a time
- **By user journey**: Sign up → Log in → Password reset → Account settings
- **By component**: API → Worker → Admin UI
- **MVP first**: Core feature → Enhancements → Edge cases

## Edge Cases

| Situation | Recommendation |
|-----------|----------------|
| **Unfamiliar codebase** | Use autospec even for smaller tasks—the spec helps you learn existing patterns |
| **Security-sensitive change** | Use autospec regardless of size—you need the review artifact |
| **Compliance/audit needs** | Use autospec—specs create a paper trail |
| **Prototyping/exploring** | Skip autospec until approach is validated |
| **Production hotfix** | Skip autospec for the emergency fix, but backfill spec if the change was significant |
| **Related small tasks** | Group into one optimal-sized spec rather than running separately |
| **Teaching/onboarding** | Use autospec—specs help new devs understand the "why" |

## Summary Table

| | Too Small | Optimal | Too Large |
|---|-----------|---------|-----------|
| **Time** | < 30 min | 2-8 hours | > 2 days |
| **Files** | 1-2 | 3-10 | 10+ |
| **Tasks** | 1-2 | 3-15 | 20+ |
| **Decisions** | 0-1 | 2-5 | 6+ concerns |
| **Action** | Just do it | `autospec prep` → `implement` | Split, then autospec each |

## See Also

- [Quick Start Guide](./quickstart.md) — Get started with autospec
- [FAQ](./faq.md) — Common questions about workflow choices
