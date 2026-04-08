This file provides additional guidance to AI coding assistants working under `./frontend/`.

## Inheritance

- Follow the repository-wide guidance in `../AGENTS.md`.
- Treat this file as frontend-specific additions, not a replacement for the root instructions.

## React Migration

- For Vue-to-React migrations in `frontend/`, read and follow `../docs/plans/2026-04-08-react-migration-playbook.md`.
- Use that playbook to decide migration order, safe Vue deletions, state/data boundaries, testing expectations, and CI pitfalls.

## Frontend Reminder

- All new UI code should be written in React unless an existing Vue surface must be preserved temporarily for compatibility.
- Do not delete Vue counterparts until you verify they have no remaining live callers.
