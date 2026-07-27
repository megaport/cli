package mcr

import (
	"context"

	megaport "github.com/megaport/megaportgo"
)

// Wrapper functions for Looking Glass service methods to enable testability

var listIPRoutesFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) ([]*megaport.LookingGlassIPRoute, error) {
	return client.MCRLookingGlassService.ListIPRoutes(ctx, mcrUID)
}

var listIPRoutesWithFilterFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ListIPRoutesRequest) ([]*megaport.LookingGlassIPRoute, error) {
	return client.MCRLookingGlassService.ListIPRoutesWithFilter(ctx, req)
}

var listBGPRoutesFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) ([]*megaport.LookingGlassBGPRoute, error) {
	return client.MCRLookingGlassService.ListBGPRoutes(ctx, mcrUID)
}

var listBGPRoutesWithFilterFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ListBGPRoutesRequest) ([]*megaport.LookingGlassBGPRoute, error) {
	return client.MCRLookingGlassService.ListBGPRoutesWithFilter(ctx, req)
}

var listBGPSessionsFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) ([]*megaport.LookingGlassBGPSession, error) {
	return client.MCRLookingGlassService.ListBGPSessions(ctx, mcrUID)
}

var listBGPNeighborRoutesFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ListBGPNeighborRoutesRequest) ([]*megaport.LookingGlassBGPNeighborRoute, error) {
	return client.MCRLookingGlassService.ListBGPNeighborRoutes(ctx, req)
}

var pingMCRFunc = func(ctx context.Context, client *megaport.Client, req *megaport.MCRPingRequest) (string, error) {
	return client.MCRLookingGlassService.PingMCR(ctx, req)
}

var waitForMCRPingFunc = func(ctx context.Context, client *megaport.Client, mcrUID, operationID string) (*megaport.LookingGlassPingResult, error) {
	return client.MCRLookingGlassService.WaitForMCRPing(ctx, mcrUID, operationID)
}

var tracerouteMCRFunc = func(ctx context.Context, client *megaport.Client, req *megaport.MCRTracerouteRequest) (string, error) {
	return client.MCRLookingGlassService.TracerouteMCR(ctx, req)
}

var waitForMCRTracerouteFunc = func(ctx context.Context, client *megaport.Client, mcrUID, operationID string) (*megaport.LookingGlassTracerouteResult, error) {
	return client.MCRLookingGlassService.WaitForMCRTraceroute(ctx, mcrUID, operationID)
}
