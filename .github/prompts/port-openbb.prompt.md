---
description: "Continue porting OpenBB features into stock_home from the parity backlog"
agent: openbb-porter
argument-hint: "next [N] | phase <N> | F###"
---
Run the OpenBB port loop for: ${input:scope:next 1}

Use [the backlog](../../stock_home/docs/openbb-parity/BACKLOG.md) as the source of truth and [stock_home/AGENTS.md](../../stock_home/AGENTS.md) for conventions. Implement, test, record and locally commit each feature before moving to the next.
