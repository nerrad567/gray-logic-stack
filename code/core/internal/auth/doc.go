// Package auth provides authentication and authorisation for Gray Logic Core.
//
// It implements a 4-tier role model (panel → user → admin → owner) with:
//   - Argon2id password hashing (OWASP 2025 recommendation)
//   - JWT access/refresh token rotation with family-based theft detection
//   - Multi-room panel device identity for frictionless wall panel access
//   - Explicit per-user room grants with per-room scene management control
//   - Static role-permission mapping (compile-time, no database lookup)
//
// Room scoping uses a "zero access by default, grant explicitly" model:
// a user with no room assignments cannot access anything. Admin must
// deliberately grant access to specific rooms via user_room_access.
// Admin and owner roles bypass room scoping entirely.
package auth
