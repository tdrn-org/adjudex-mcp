/*
 * Copyright 2026 Holger de Carne
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package rest

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/adjudex-mcp/internal/data"
)

// TestSafeErrorMessage pins the whitelist: only failures that describe the
// request itself reach the client. Everything else must stay opaque, otherwise a
// driver or provider error would leak internal state through the API.
func TestSafeErrorMessage(t *testing.T) {
	// Safe: the message is built from the request alone.
	unknown := fmt.Errorf("unknown listing for %q: %w", "demo:symbol=X", data.ErrUnknownListing)
	message, ok := safeErrorMessage(unknown)
	require.True(t, ok)
	require.Contains(t, message, "unknown listing")

	malformed := fmt.Errorf("%w: %q cannot be read", ErrInvalidListing, "demo")
	message, ok = safeErrorMessage(malformed)
	require.True(t, ok)
	require.Contains(t, message, "invalid listing")

	// Not safe: internal state and driver errors never leave the server.
	for _, cause := range []error{
		nil,
		errors.New("sql: no such table: position"),
		fmt.Errorf("querying provider: %w", errors.New("401 unauthorized")),
	} {
		message, ok = safeErrorMessage(cause)
		require.False(t, ok, "expected %v to be withheld", cause)
		require.Empty(t, message)
	}
}
