package errors_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	gliveerrors "github.com/glive/domain/errors"
)

// TestGliveError_WithContext_DoesNotMutateSharedSentinel is the regression test for
// AUDIT-F-19. WithContext and WithCause used to mutate the receiver and return it,
// so calling them on a package-level sentinel such as ErrInvalidURL rewrote shared
// global state. Under concurrency that is a data race on the sentinel's Context map;
// single-threaded it silently leaked one caller's context into every later caller's
// error. Run with -race: this test fails on the pre-fix code.
func TestGliveError_WithContext_DoesNotMutateSharedSentinel(t *testing.T) {
	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()
			// Every goroutine decorates the SAME shared sentinel.
			derived := gliveerrors.ErrInvalidURL.WithContext("reason", fmt.Sprintf("caller-%d", i))
			if got := derived.Context["reason"]; got != fmt.Sprintf("caller-%d", i) {
				t.Errorf("goroutine %d saw context %q, want its own value", i, got)
			}
		}(i)
	}
	wg.Wait()

	// The sentinel itself must be untouched by any of the 50 calls.
	if gliveerrors.ErrInvalidURL.Context != nil {
		t.Errorf("shared sentinel was mutated: Context = %v, want nil", gliveerrors.ErrInvalidURL.Context)
	}
}

// TestGliveError_WithCause_DoesNotMutateSharedSentinel covers the same defect on
// the WithCause path.
func TestGliveError_WithCause_DoesNotMutateSharedSentinel(t *testing.T) {
	cause := errors.New("underlying failure")

	derived := gliveerrors.ErrInvalidURL.WithCause(cause)

	if !errors.Is(derived, cause) {
		t.Errorf("derived error does not wrap the cause")
	}
	if gliveerrors.ErrInvalidURL.Cause != nil {
		t.Errorf("shared sentinel was mutated: Cause = %v, want nil", gliveerrors.ErrInvalidURL.Cause)
	}
}

// TestGliveError_WithContext_CopyIsIndependent proves the copy does not share the
// original's Context map — a shallow struct copy alone would still alias it.
func TestGliveError_WithContext_CopyIsIndependent(t *testing.T) {
	base := gliveerrors.NewUserError("TEST_CODE", "test message").WithContext("first", "1")

	derived := base.WithContext("second", "2")

	if _, found := base.Context["second"]; found {
		t.Errorf("writing to the copy mutated the original's context map")
	}
	if derived.Context["first"] != "1" {
		t.Errorf("the copy lost context inherited from the original")
	}
}

// TestGliveError_SentinelsAreUnmodifiedAfterDecoration guards every package-level
// sentinel, not just ErrInvalidURL, so a future sentinel added without a copy is
// caught here.
func TestGliveError_SentinelsAreUnmodifiedAfterDecoration(t *testing.T) {
	sentinels := map[string]*gliveerrors.GliveError{
		"ErrInvalidURL":           gliveerrors.ErrInvalidURL,
		"ErrMissingAPIKey":        gliveerrors.ErrMissingAPIKey,
		"ErrNetworkTimeout":       gliveerrors.ErrNetworkTimeout,
		"ErrAPIUnavailable":       gliveerrors.ErrAPIUnavailable,
		"ErrCloneFailed":          gliveerrors.ErrCloneFailed,
		"ErrRepoNotFound":         gliveerrors.ErrRepoNotFound,
		"ErrCommandFailed":        gliveerrors.ErrCommandFailed,
		"ErrCommandTimeout":       gliveerrors.ErrCommandTimeout,
		"ErrSecurityViolation":    gliveerrors.ErrSecurityViolation,
		"ErrSuspiciousRepository": gliveerrors.ErrSuspiciousRepository,
		"ErrAIResponseInvalid":    gliveerrors.ErrAIResponseInvalid,
		"ErrDiskFull":             gliveerrors.ErrDiskFull,
		"ErrInternalPanic":        gliveerrors.ErrInternalPanic,
	}

	for name, sentinel := range sentinels {
		t.Run(name, func(t *testing.T) {
			_ = sentinel.WithContext("probe", "value")
			_ = sentinel.WithCause(errors.New("probe cause"))

			if sentinel.Context != nil {
				t.Errorf("%s.Context was mutated: %v", name, sentinel.Context)
			}
			if sentinel.Cause != nil {
				t.Errorf("%s.Cause was mutated: %v", name, sentinel.Cause)
			}
		})
	}
}
