// Copyright 2026 The KYBa Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package kcp

import (
	"net/http"
	"strconv"

	kcp_model "code.gitea.io/gitea/models/kcp"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/context"
)

// RegisterRoutes registers KYBa KCP API endpoints under /api/v1/kcp.
func RegisterRoutes(m *web.Router) {
	m.Get("/capsules", ListCapsules)
	m.Get("/imports", ListImports)
	m.Get("/impact", Impact)
	m.Get("/export-plan", ExportPlan)
}

func ListCapsules(ctx *context.APIContext) {
	repoID, ok := requireRepoID(ctx)
	if !ok {
		return
	}
	items, err := kcp_model.ListRepositoryInterfaces(ctx, repoID)
	if err != nil {
		ctx.APIErrorInternal(err)
		return
	}
	ctx.JSON(http.StatusOK, items)
}

func ListImports(ctx *context.APIContext) {
	repoID, ok := requireRepoID(ctx)
	if !ok {
		return
	}
	items, err := kcp_model.ListImportsForRepo(ctx, repoID)
	if err != nil {
		ctx.APIErrorInternal(err)
		return
	}
	ctx.JSON(http.StatusOK, items)
}

func Impact(ctx *context.APIContext) {
	repoID, ok := requireRepoID(ctx)
	if !ok {
		return
	}
	items, err := kcp_model.ListImpactTasksForRepo(ctx, repoID)
	if err != nil {
		ctx.APIErrorInternal(err)
		return
	}
	ctx.JSON(http.StatusOK, items)
}

func ExportPlan(ctx *context.APIContext) {
	repoID, ok := requireRepoID(ctx)
	if !ok {
		return
	}
	interfaces, err := kcp_model.ListRepositoryInterfaces(ctx, repoID)
	if err != nil {
		ctx.APIErrorInternal(err)
		return
	}
	plans := make([]map[string]any, 0, len(interfaces))
	for _, item := range interfaces {
		files, err := kcp_model.ListRepositoryInterfaceFiles(ctx, repoID, item.InterfaceID)
		if err != nil {
			ctx.APIErrorInternal(err)
			return
		}
		plans = append(plans, map[string]any{
			"interface_id": item.InterfaceID,
			"kind":         item.Kind,
			"version":      item.Version,
			"visibility":   item.Visibility,
			"files":        files,
			"ready":        len(files) > 0,
		})
	}
	ctx.JSON(http.StatusOK, map[string]any{
		"ready": len(plans) > 0,
		"plans": plans,
	})
}

func requireRepoID(ctx *context.APIContext) (int64, bool) {
	raw := ctx.FormString("repo_id")
	if raw == "" {
		ctx.JSON(http.StatusBadRequest, map[string]string{"message": "repo_id query parameter is required"})
		return 0, false
	}
	repoID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || repoID <= 0 {
		ctx.JSON(http.StatusBadRequest, map[string]string{"message": "repo_id must be a positive integer"})
		return 0, false
	}
	return repoID, true
}
