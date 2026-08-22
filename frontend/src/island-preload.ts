// The shared Island package still owns the allowlisted bridge projections.
// Forge bundles that bridge into Kennel's dedicated preload target so the
// renderer receives no Node or Electron capabilities directly.
import "../../packages/kennel-island/desktop/preload.cjs";
