package errors

import (
	"errors"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name   string
		appErr *AppError
		want   string
	}{
		{
			name:   "with wrapped error",
			appErr: New("NOT_FOUND", "resource missing", ErrNotFound),
			want:   "resource missing: not found",
		},
		{
			name:   "without wrapped error",
			appErr: New("INTERNAL", "something went wrong", nil),
			want:   "something went wrong",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.appErr.Error(); got != tc.want {
				t.Errorf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	tests := []struct {
		name   string
		appErr *AppError
		target error
		wantIs bool
	}{
		{
			name:   "unwraps to sentinel ErrNotFound",
			appErr: New("NOT_FOUND", "not found", ErrNotFound),
			target: ErrNotFound,
			wantIs: true,
		},
		{
			name:   "does not unwrap to unrelated sentinel",
			appErr: New("NOT_FOUND", "not found", ErrNotFound),
			target: ErrUnauthorized,
			wantIs: false,
		},
		{
			name:   "nil wrapped error",
			appErr: New("INTERNAL", "oops", nil),
			target: ErrInternal,
			wantIs: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := errors.Is(tc.appErr, tc.target); got != tc.wantIs {
				t.Errorf("errors.Is() = %v, want %v", got, tc.wantIs)
			}
		})
	}
}
