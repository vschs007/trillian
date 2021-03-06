// Copyright 2016 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"fmt"
	"time"

	"github.com/google/trillian"
	"github.com/google/trillian/extension"
	"github.com/google/trillian/log"
	"github.com/google/trillian/trees"
)

// SequencerManager provides sequencing operations for a collection of Logs.
type SequencerManager struct {
	guardWindow time.Duration
	registry    extension.Registry
}

// NewSequencerManager creates a new SequencerManager instance based on the provided KeyManager instance
// and guard window.
func NewSequencerManager(registry extension.Registry, gw time.Duration) *SequencerManager {
	return &SequencerManager{
		guardWindow: gw,
		registry:    registry,
	}
}

// Name returns the name of the object.
func (s SequencerManager) Name() string {
	return "Sequencer"
}

// ExecutePass performs sequencing for the specified Log.
func (s SequencerManager) ExecutePass(ctx context.Context, logID int64, info *LogOperationInfo) (int, error) {
	// TODO(Martin2112): Honor the sequencing enabled in log parameters, needs an API change
	// so deferring it

	tree, err := trees.GetTree(
		ctx,
		s.registry.AdminStorage,
		logID,
		trees.GetOpts{TreeType: trillian.TreeType_LOG})
	if err != nil {
		return 0, fmt.Errorf("error retrieving log %v: %v", logID, err)
	}
	ctx = trees.NewContext(ctx, tree)

	hasher, err := trees.Hasher(tree)
	if err != nil {
		return 0, fmt.Errorf("error getting hasher for log %v: %v", logID, err)
	}

	signer, err := trees.Signer(ctx, s.registry.SignerFactory, tree)
	if err != nil {
		return 0, fmt.Errorf("error getting signer for log %v: %v", logID, err)
	}

	sequencer := log.NewSequencer(hasher, info.TimeSource, s.registry.LogStorage, signer)
	sequencer.SetGuardWindow(s.guardWindow)

	leaves, err := sequencer.SequenceBatch(ctx, logID, info.BatchSize)
	if err != nil {
		return 0, fmt.Errorf("failed to sequence batch for %v: %v", logID, err)
	}
	return leaves, nil
}
