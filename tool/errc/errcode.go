// Copyright (c) 2025 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package errc

import (
	"runtime/debug"
)

type PlentifulError struct {
	Reason  string
	Cause   string
	Details map[string]string
}

func (e *PlentifulError) Error() string {
	return e.Reason + "\n" + e.Cause
}

func New(message string) *PlentifulError {
	e := &PlentifulError{
		Reason:  message,
		Details: make(map[string]string),
	}
	stackTrace := debug.Stack()
	e.Cause = string(stackTrace)
	return e
}

func (pe *PlentifulError) With(key, value string) *PlentifulError {
	pe.Details[key] = value
	return pe
}

func Adhere(err error, key, value string) error {
	if perr, ok := err.(*PlentifulError); ok {
		perr.Details[key] = value
		return perr
	}
	return err
}
