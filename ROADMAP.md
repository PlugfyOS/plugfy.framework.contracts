# Roadmap — plugfy-common

| Milestone | Goal | Items |
|---|---|---|
| **v1.0.0** | Stable, stdlib-only base ABI | CMN-01..06, CMN-08 |
| **v1.1.0** | First-class telemetry contract + frozen ABI | CMN-07, CMN-09 |
| **v1.2.0** | Distributed trace over the event bus | CMN-10 |

Tied to the master roadmap in
[`PlugfyOS/plugfy-platform`](https://github.com/PlugfyOS/plugfy-platform). As the
L1 baseplate, `plugfy-common` is the first unit of any release wave — every other
module pins `^1.x` of it, so its ABI must stay backward-compatible within a major.
