// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fakeserver

import (
	"context"
	"testing"

	api "github.com/gshireesh/terraform-provider-shireesh/api/shireesh.com/config/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Test CRUD lifecycle against the in-memory server directly via gRPC.
func TestSimpleCRUD(t *testing.T) {
	ctx := context.Background()
	s := NewInMemoryServer()
	httpSrv, grpcTarget, err := s.Start(ctx)
	if err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer httpSrv.Close()

	conn, err := grpc.Dial(grpcTarget, grpc.WithTransportCredentials(insecure.NewCredentials())) //nolint:staticcheck // grpc.Dial is deprecated in favor of NewClient; acceptable in test
	if err != nil {
		t.Fatalf("dial grpc: %v", err)
	}
	defer conn.Close()
	cli := api.NewGrpcTerraformServiceClient(conn)

	// Create
	cResp, err := cli.SimpleCreate(ctx, &api.SimpleCreateRequest{Item: &api.SimpleCreateInput{Score: 5, Value: "val", Flags: []bool{true}, Counts: []float64{1.2}}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if cResp.Item == nil || cResp.Item.Id == "" {
		t.Fatalf("expected id on create")
	}
	id := cResp.Item.Id

	// Read
	rResp, err := cli.SimpleRead(ctx, &api.SimpleReadRequest{Id: id})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if rResp.Item == nil || rResp.Item.Score != 5 {
		t.Fatalf("unexpected read item: %#v", rResp.Item)
	}

	// Update
	uResp, err := cli.SimpleUpdate(ctx, &api.SimpleUpdateRequest{Id: id, Item: &api.SimpleUpdateInput{Score: 10, Value: "updated"}})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if uResp.Item.Score != 10 || uResp.Item.Value != "updated" {
		t.Fatalf("unexpected update item: %#v", uResp.Item)
	}

	// Delete
	dResp, err := cli.SimpleDelete(ctx, &api.SimpleDeleteRequest{Id: id})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !dResp.Success {
		t.Fatalf("expected delete success")
	}

	// Read missing after delete
	rMissing, err := cli.SimpleRead(ctx, &api.SimpleReadRequest{Id: id})
	if err != nil {
		t.Fatalf("read missing: %v", err)
	}
	if rMissing.Item != nil && rMissing.Item.Id != "" {
		t.Fatalf("expected no item after delete, got %#v", rMissing.Item)
	}
}
