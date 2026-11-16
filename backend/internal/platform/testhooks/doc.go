// Package testhooks exposes the non-production test-control surface (R19, 20 §QA-16): clock
// advance, running due River jobs, reading the fake OTP/SMS/push outboxes, and reset/seed. It is
// compiled only with the `testhooks` build tag and served on a separate listener guarded by
// ROVO_TESTHOOKS_SECRET. The production image is built without the tag; a CI test asserts the
// routes and symbols are absent (charter §4).
//
// The build-tagged implementation lives in handler_testhooks.go; handler_prod.go provides a
// no-op Mount so the composition root compiles in every build.
package testhooks
