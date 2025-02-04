// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package managedwriter

import (
	"context"
	"testing"

	"github.com/googleapis/gax-go/v2"
	"google.golang.org/grpc"
)

func TestTableParentFromStreamName(t *testing.T) {
	testCases := []struct {
		in   string
		want string
	}{
		{
			"bad",
			"bad",
		},
		{
			"projects/foo/datasets/bar/tables/baz",
			"projects/foo/datasets/bar/tables/baz",
		},
		{
			"projects/foo/datasets/bar/tables/baz/zip/zam/zoomie",
			"projects/foo/datasets/bar/tables/baz",
		},
		{
			"projects/foo/datasets/bar/tables/baz/_default",
			"projects/foo/datasets/bar/tables/baz",
		},
	}

	for _, tc := range testCases {
		got := TableParentFromStreamName(tc.in)
		if got != tc.want {
			t.Errorf("mismatch on %s: got %s want %s", tc.in, got, tc.want)
		}
	}
}

// TestCreatePool tests the result of calling createPool with different combinations
// of global configuration and per-writer configuration.
func TestCreatePool(t *testing.T) {
	testCases := []struct {
		desc            string
		cfg             *writerClientConfig
		settings        *streamSettings
		wantMaxBytes    int
		wantMaxRequests int
		wantCallOptions int
		wantErr         bool
	}{
		{
			desc:    "no config",
			wantErr: true,
		},
		{
			desc: "cfg, no settings",
			cfg: &writerClientConfig{
				defaultInflightRequests: 12,
				defaultInflightBytes:    2048,
			},
			wantMaxBytes:    2048,
			wantMaxRequests: 12,
		},
		{
			desc: "empty cfg, w/settings",
			cfg:  &writerClientConfig{},
			settings: &streamSettings{
				MaxInflightRequests: 99,
				MaxInflightBytes:    1024,
				appendCallOptions:   []gax.CallOption{gax.WithPath("foo")},
			},
			wantMaxBytes:    1024,
			wantMaxRequests: 99,
			wantCallOptions: 1,
		},
		{
			desc: "both cfg and settings",
			cfg: &writerClientConfig{
				defaultInflightRequests:      123,
				defaultInflightBytes:         456,
				defaultAppendRowsCallOptions: []gax.CallOption{gax.WithGRPCOptions(grpc.MaxCallRecvMsgSize(999))},
			},
			settings: &streamSettings{
				MaxInflightRequests: 99,
				MaxInflightBytes:    1024,
			},
			wantMaxBytes:    1024,
			wantMaxRequests: 99,
			wantCallOptions: 1,
		},
		{
			desc: "merge defaults and settings",
			cfg: &writerClientConfig{
				defaultInflightRequests:      123,
				defaultInflightBytes:         456,
				defaultAppendRowsCallOptions: []gax.CallOption{gax.WithGRPCOptions(grpc.MaxCallRecvMsgSize(999))},
			},
			settings: &streamSettings{
				MaxInflightBytes:  1024,
				appendCallOptions: []gax.CallOption{gax.WithPath("foo")},
			},
			wantMaxBytes:    1024,
			wantMaxRequests: 123,
			wantCallOptions: 2,
		},
	}

	for _, tc := range testCases {
		c := &Client{
			cfg: tc.cfg,
		}
		got, err := c.createPool(context.Background(), tc.settings, nil, newSimpleRouter(""), false)
		if err != nil {
			if !tc.wantErr {
				t.Errorf("case %q: createPool errored unexpectedly: %v", tc.desc, err)
			}
			continue
		}
		if err == nil && tc.wantErr {
			t.Errorf("case %q: expected createPool to error but it did not", tc.desc)
			continue
		}
		// too many go-cmp overrides needed to quickly diff here, look at the interesting fields explicitly.
		if gotVal := got.baseFlowController.maxInsertBytes; gotVal != tc.wantMaxBytes {
			t.Errorf("case %q: flowController maxInsertBytes mismatch, got %d want %d", tc.desc, gotVal, tc.wantMaxBytes)
		}
		if gotVal := got.baseFlowController.maxInsertCount; gotVal != tc.wantMaxRequests {
			t.Errorf("case %q: flowController maxInsertCount mismatch, got %d want %d", tc.desc, gotVal, tc.wantMaxRequests)
		}
		if gotVal := len(got.callOptions); gotVal != tc.wantCallOptions {
			t.Errorf("case %q: calloption count mismatch, got %d want %d", tc.desc, gotVal, tc.wantCallOptions)
		}
	}
}
