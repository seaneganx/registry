// Copyright 2020 Google LLC. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/apigee/registry/rpc"
	"github.com/apigee/registry/server/dao"
	"github.com/apigee/registry/server/models"
	"github.com/apigee/registry/server/names"
	"github.com/golang/protobuf/ptypes/empty"
)

type artifactParent interface {
	Artifact(string) names.Artifact
}

func parseArtifactParent(name string) (artifactParent, error) {
	if s, err := names.ParseSpec(name); err == nil {
		return s, nil
	} else if v, err := names.ParseVersion(name); err == nil {
		return v, nil
	} else if a, err := names.ParseApi(name); err == nil {
		return a, nil
	} else if p, err := names.ParseProject(name); err == nil {
		return p, nil
	}

	return nil, fmt.Errorf("invalid artifact parent %q", name)
}

// CreateArtifact handles the corresponding API request.
func (s *RegistryServer) CreateArtifact(ctx context.Context, req *rpc.CreateArtifactRequest) (*rpc.Artifact, error) {
	client, err := s.getStorageClient(ctx)
	if err != nil {
		return nil, unavailableError(err)
	}
	defer s.releaseStorageClient(client)
	db := dao.NewDAO(client)

	if req.GetArtifact() == nil {
		return nil, invalidArgumentError(fmt.Errorf("invalid artifact %+v: body must be provided", req.GetArtifact()))
	}

	parent, err := parseArtifactParent(req.GetParent())
	if err != nil {
		return nil, invalidArgumentError(err)
	}

	name := parent.Artifact(names.GenerateID())
	if req.GetArtifactId() != "" {
		name = parent.Artifact(req.GetArtifactId())
	}

	if _, err := db.GetArtifact(ctx, name); err == nil {
		return nil, alreadyExistsError(fmt.Errorf("artifact %q already exists", name))
	} else if !isNotFound(err) {
		return nil, err
	}

	if err := name.Validate(); err != nil {
		return nil, invalidArgumentError(err)
	}

	// Creation should only succeed when the parent exists.
	switch parent := parent.(type) {
	case names.Project:
		if _, err := db.GetProject(ctx, parent); err != nil {
			return nil, err
		}
	case names.Api:
		if _, err := db.GetApi(ctx, parent); err != nil {
			return nil, err
		}
	case names.Version:
		if _, err := db.GetVersion(ctx, parent); err != nil {
			return nil, err
		}
	case names.Spec:
		if _, err := db.GetSpec(ctx, parent); err != nil {
			return nil, err
		}
	}

	artifact := models.NewArtifact(name, req.GetArtifact())
	if err := db.SaveArtifact(ctx, artifact); err != nil {
		return nil, err
	}

	if err := db.SaveArtifactContents(ctx, artifact, req.Artifact.GetContents()); err != nil {
		return nil, err
	}

	message, err := artifact.BasicMessage()
	if err != nil {
		return nil, internalError(err)
	}

	s.notify(rpc.Notification_CREATED, name.String())
	return message, nil
}

// DeleteArtifact handles the corresponding API request.
func (s *RegistryServer) DeleteArtifact(ctx context.Context, req *rpc.DeleteArtifactRequest) (*empty.Empty, error) {
	client, err := s.getStorageClient(ctx)
	if err != nil {
		return nil, unavailableError(err)
	}
	defer s.releaseStorageClient(client)
	db := dao.NewDAO(client)

	name, err := names.ParseArtifact(req.GetName())
	if err != nil {
		return nil, invalidArgumentError(err)
	}

	// Deletion should only succeed on artifacts that currently exist.
	if _, err := db.GetArtifact(ctx, name); err != nil {
		return nil, err
	}

	if err := db.DeleteArtifact(ctx, name); err != nil {
		return nil, err
	}

	s.notify(rpc.Notification_DELETED, name.String())
	return &empty.Empty{}, nil
}

// GetArtifact handles the corresponding API request.
func (s *RegistryServer) GetArtifact(ctx context.Context, req *rpc.GetArtifactRequest) (*rpc.Artifact, error) {
	client, err := s.getStorageClient(ctx)
	if err != nil {
		return nil, unavailableError(err)
	}
	defer s.releaseStorageClient(client)
	db := dao.NewDAO(client)

	name, err := names.ParseArtifact(req.GetName())
	if err != nil {
		return nil, invalidArgumentError(err)
	}

	artifact, err := db.GetArtifact(ctx, name)
	if err != nil {
		return nil, err
	}

	var message *rpc.Artifact
	switch req.GetView() {
	case rpc.View_FULL:
		blob, err := db.GetArtifactContents(ctx, name)
		if err != nil {
			return nil, err
		}

		message, err = artifact.FullMessage(blob)
		if err != nil {
			return nil, internalError(err)
		}

	case rpc.View_BASIC, rpc.View_VIEW_UNSPECIFIED:
		message, err = artifact.BasicMessage()
		if err != nil {
			return nil, internalError(err)
		}

	default:
		return nil, invalidArgumentError(fmt.Errorf("unknown view type %v", req.GetView()))
	}

	return message, nil
}

// ListArtifacts handles the corresponding API request.
func (s *RegistryServer) ListArtifacts(ctx context.Context, req *rpc.ListArtifactsRequest) (*rpc.ListArtifactsResponse, error) {
	client, err := s.getStorageClient(ctx)
	if err != nil {
		return nil, unavailableError(err)
	}
	defer s.releaseStorageClient(client)
	db := dao.NewDAO(client)

	if req.GetPageSize() < 0 {
		return nil, invalidArgumentError(fmt.Errorf("invalid page_size %d: must not be negative", req.GetPageSize()))
	} else if req.GetPageSize() > 1000 {
		req.PageSize = 1000
	} else if req.GetPageSize() == 0 {
		req.PageSize = 50
	}

	parent, err := parseArtifactParent(req.GetParent())
	if err != nil {
		return nil, invalidArgumentError(err)
	}

	var listing dao.ArtifactList
	switch parent := parent.(type) {
	case names.Project:
		listing, err = db.ListProjectArtifacts(ctx, parent, dao.PageOptions{
			Size:   req.GetPageSize(),
			Filter: req.GetFilter(),
			Token:  req.GetPageToken(),
		})
	case names.Api:
		listing, err = db.ListApiArtifacts(ctx, parent, dao.PageOptions{
			Size:   req.GetPageSize(),
			Filter: req.GetFilter(),
			Token:  req.GetPageToken(),
		})
	case names.Version:
		listing, err = db.ListVersionArtifacts(ctx, parent, dao.PageOptions{
			Size:   req.GetPageSize(),
			Filter: req.GetFilter(),
			Token:  req.GetPageToken(),
		})
	case names.Spec:
		listing, err = db.ListSpecArtifacts(ctx, parent, dao.PageOptions{
			Size:   req.GetPageSize(),
			Filter: req.GetFilter(),
			Token:  req.GetPageToken(),
		})
	}
	if err != nil {
		return nil, err
	}

	response := &rpc.ListArtifactsResponse{
		Artifacts:     make([]*rpc.Artifact, len(listing.Artifacts)),
		NextPageToken: listing.Token,
	}

	for i, artifact := range listing.Artifacts {
		switch req.GetView() {
		case rpc.View_FULL:
			name, err := names.ParseArtifact(artifact.Name())
			if err != nil {
				return nil, internalError(err)
			}

			blob, err := db.GetArtifactContents(ctx, name)
			if err != nil {
				return nil, internalError(err)
			}

			response.Artifacts[i], err = artifact.FullMessage(blob)
			if err != nil {
				return nil, internalError(err)
			}

		case rpc.View_BASIC, rpc.View_VIEW_UNSPECIFIED:
			response.Artifacts[i], err = artifact.BasicMessage()
			if err != nil {
				return nil, internalError(err)
			}

		default:
			return nil, invalidArgumentError(fmt.Errorf("unknown view type %v", req.GetView()))
		}
	}

	return response, nil
}

// ReplaceArtifact handles the corresponding API request.
func (s *RegistryServer) ReplaceArtifact(ctx context.Context, req *rpc.ReplaceArtifactRequest) (*rpc.Artifact, error) {
	client, err := s.getStorageClient(ctx)
	if err != nil {
		return nil, unavailableError(err)
	}
	defer s.releaseStorageClient(client)
	db := dao.NewDAO(client)

	name, err := names.ParseArtifact(req.Artifact.GetName())
	if err != nil {
		return nil, invalidArgumentError(err)
	}

	// Replacement should only succeed on artifacts that currently exist.
	if _, err := db.GetArtifact(ctx, name); err != nil {
		return nil, err
	}

	artifact := models.NewArtifact(name, req.GetArtifact())
	if err := db.SaveArtifact(ctx, artifact); err != nil {
		return nil, err
	}

	if err := db.SaveArtifactContents(ctx, artifact, req.Artifact.GetContents()); err != nil {
		return nil, internalError(err)
	}

	s.notify(rpc.Notification_UPDATED, name.String())
	return artifact.BasicMessage()
}
