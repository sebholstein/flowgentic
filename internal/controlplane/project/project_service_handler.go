package project

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	controlplanev1 "github.com/sebastianm/flowgentic/internal/proto/gen/controlplane/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// projectServiceHandler implements controlplanev1connect.ProjectServiceHandler.
type projectServiceHandler struct {
	log *slog.Logger
	svc *ProjectService
}

func (h *projectServiceHandler) ListProjects(
	ctx context.Context,
	_ *connect.Request[controlplanev1.ListProjectsRequest],
) (*connect.Response[controlplanev1.ListProjectsResponse], error) {
	projects, err := h.svc.ListProjects(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	pbProjects := make([]*controlplanev1.ProjectConfig, len(projects))
	for i, p := range projects {
		pbProjects[i] = projectToProto(p)
	}

	return connect.NewResponse(&controlplanev1.ListProjectsResponse{
		Projects: pbProjects,
	}), nil
}

func (h *projectServiceHandler) GetProject(
	ctx context.Context,
	req *connect.Request[controlplanev1.GetProjectRequest],
) (*connect.Response[controlplanev1.GetProjectResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	p, err := h.svc.GetProject(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&controlplanev1.GetProjectResponse{
		Project: projectToProto(p),
	}), nil
}

func (h *projectServiceHandler) CreateProject(
	ctx context.Context,
	req *connect.Request[controlplanev1.CreateProjectRequest],
) (*connect.Response[controlplanev1.CreateProjectResponse], error) {
	p := Project{
		ID:                  req.Msg.Id,
		Name:                req.Msg.Name,
		DefaultPlannerAgent: req.Msg.DefaultPlannerAgent,
		DefaultPlannerModel: req.Msg.DefaultPlannerModel,
		EmbeddedWorkerPath:  req.Msg.EmbeddedWorkerPath,
		WorkerPaths:         req.Msg.WorkerPaths,
		SortIndex:           req.Msg.SortIndex,
	}

	created, err := h.svc.CreateProject(ctx, p)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&controlplanev1.CreateProjectResponse{
		Project: projectToProto(created),
	}), nil
}

func (h *projectServiceHandler) UpdateProject(
	ctx context.Context,
	req *connect.Request[controlplanev1.UpdateProjectRequest],
) (*connect.Response[controlplanev1.UpdateProjectResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	p := Project{
		ID:                  req.Msg.Id,
		Name:                req.Msg.Name,
		DefaultPlannerAgent: req.Msg.DefaultPlannerAgent,
		DefaultPlannerModel: req.Msg.DefaultPlannerModel,
		EmbeddedWorkerPath:  req.Msg.EmbeddedWorkerPath,
		WorkerPaths:         req.Msg.WorkerPaths,
		SortIndex:           req.Msg.SortIndex,
	}

	updated, err := h.svc.UpdateProject(ctx, p)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&controlplanev1.UpdateProjectResponse{
		Project: projectToProto(updated),
	}), nil
}

func (h *projectServiceHandler) DeleteProject(
	ctx context.Context,
	req *connect.Request[controlplanev1.DeleteProjectRequest],
) (*connect.Response[controlplanev1.DeleteProjectResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	if err := h.svc.DeleteProject(ctx, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&controlplanev1.DeleteProjectResponse{}), nil
}

func (h *projectServiceHandler) ReorderProjects(
	ctx context.Context,
	req *connect.Request[controlplanev1.ReorderProjectsRequest],
) (*connect.Response[controlplanev1.ReorderProjectsResponse], error) {
	entries := make([]SortEntry, len(req.Msg.Entries))
	for i, e := range req.Msg.Entries {
		entries[i] = SortEntry{ID: e.Id, SortIndex: e.SortIndex}
	}

	if err := h.svc.ReorderProjects(ctx, entries); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&controlplanev1.ReorderProjectsResponse{}), nil
}

func projectToProto(p Project) *controlplanev1.ProjectConfig {
	return &controlplanev1.ProjectConfig{
		Id:                  p.ID,
		Name:                p.Name,
		DefaultPlannerAgent: p.DefaultPlannerAgent,
		DefaultPlannerModel: p.DefaultPlannerModel,
		EmbeddedWorkerPath:  p.EmbeddedWorkerPath,
		WorkerPaths:         p.WorkerPaths,
		CreatedAt:           timestamppb.New(p.CreatedAt),
		UpdatedAt:           timestamppb.New(p.UpdatedAt),
		SortIndex:           p.SortIndex,
	}
}
