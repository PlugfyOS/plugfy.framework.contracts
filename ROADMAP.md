# Roadmap — plugfy-common

| Milestone | Goal | Items |
|---|---|---|
| **v1.0.0** | Stable, stdlib-only base ABI | CMN-01..06, CMN-08 |
| **v1.0.1** | Frozen ABI (golden surface test; no ABI change) | CMN-07 |
| **v1.1.0** | Wave-0 shared contracts: `api` route-contributions, `installed` manifest + layout (render-path/compatibility), `persistence` (`SQLDB`/`RegistryStore` over stdlib `database/sql`), `spi` Kind extension | CMN-11..14 |
| **v1.2.0** | First-class telemetry contract + distributed trace over the event bus | CMN-09, CMN-10 |

Tied to the master roadmap in
[`PlugfyOS/plugfy.platform.governance.spine`](https://github.com/PlugfyOS/plugfy.platform.governance.spine). As the
L1 baseplate, `plugfy-common` is the first unit of any release wave — every other
module pins `^1.x` of it, so its ABI must stay backward-compatible within a major.
