// Package plane implements the isotope "plane" derivation. Given a charge
// assignment it builds, in one shot and with no hyperparameter search:
//
//   - a U(1)-covariant, block-diagonal edge tensor,
//   - a spanning-tree transport over the planes (never a star), and
//   - a three-point sort functional with holonomy and quarantine.
//
// The construction is deterministic: the same input yields bit-identical
// output across runs. No map iteration is used anywhere in the encoder or the
// tree builder, transports are cached per sort, and tree tie-breaks are stable.
//
// Anchor: exactly one anchor is fixed per run — 42+0i (proven consensus, the
// real axis) or 0+42i (working hypothesis, the imaginary axis). Anchors are
// never mixed inside a sort.
//
// Interpretation note. The tensor, tree, transport, holonomy, and three-point
// key are all defined over planes. A string encodes to a Node: a centered
// coordinate vector with exactly one complex value per plane. The tensor score
// compares two Nodes. The tree/transport/holonomy operate on planes for a given
// coordinate vector (the CLI calls it the probe node).
package plane
