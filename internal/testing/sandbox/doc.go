// Package sandbox provides a secure JavaScript execution environment using
// the goja runtime. It blocks dangerous APIs (require, eval, filesystem,
// network) while exposing a curated stdlib (atob/btoa, crypto, console
// capture) for test script execution.
package sandbox
