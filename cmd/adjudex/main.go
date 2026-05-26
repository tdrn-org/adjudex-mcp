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

// Package main is the entry point for adjudex — Agent juDy eXchange.
// adjudex is an MCP server for tracking and analyzing stock prices,
// with an embedded SvelteKit web frontend.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("adjudex v0.1.0 — Agent juDy eXchange")
	fmt.Println("MCP server starting...")
	os.Exit(0)
}
