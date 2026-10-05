package product

import (
	"context"
	"fmt"

	megaport "github.com/megaport/megaportgo"
)

type MockProductService struct {
	ListProductsErr    error
	ListProductsResult []megaport.Product

	GetProductTypeErr    error
	GetProductTypeResult string

	ExecuteOrderErr    error
	ExecuteOrderResult *[]byte

	ModifyProductErr    error
	ModifyProductResult *megaport.ModifyProductResponse

	DeleteProductErr    error
	DeleteProductResult *megaport.DeleteProductResponse

	ManageProductLockErr    error
	ManageProductLockResult *megaport.ManageProductLockResponse

	ValidateProductOrderErr error

	ListProductResourceTagsErr    error
	ListProductResourceTagsResult []megaport.ResourceTag

	UpdateProductResourceTagsErr error

	GetProductPricingErr    error
	GetProductPricingResult *megaport.PriceBookDTO

	GetProductPricingForCompanyErr    error
	GetProductPricingForCompanyResult *megaport.PriceBookDTO
}

func (m *MockProductService) ListProducts(ctx context.Context) ([]megaport.Product, error) {
	return m.ListProductsResult, m.ListProductsErr
}

func (m *MockProductService) GetProductType(ctx context.Context, productUID string) (string, error) {
	return m.GetProductTypeResult, m.GetProductTypeErr
}

func (m *MockProductService) ExecuteOrder(ctx context.Context, requestBody interface{}) (*[]byte, error) {
	return m.ExecuteOrderResult, m.ExecuteOrderErr
}

func (m *MockProductService) ModifyProduct(ctx context.Context, req *megaport.ModifyProductRequest) (*megaport.ModifyProductResponse, error) {
	return m.ModifyProductResult, m.ModifyProductErr
}

func (m *MockProductService) DeleteProduct(ctx context.Context, req *megaport.DeleteProductRequest) (*megaport.DeleteProductResponse, error) {
	return m.DeleteProductResult, m.DeleteProductErr
}

func (m *MockProductService) RestoreProduct(ctx context.Context, productId string) (*megaport.RestoreProductResponse, error) {
	return nil, fmt.Errorf("mock: RestoreProduct not configured")
}

func (m *MockProductService) ManageProductLock(ctx context.Context, req *megaport.ManageProductLockRequest) (*megaport.ManageProductLockResponse, error) {
	return m.ManageProductLockResult, m.ManageProductLockErr
}

func (m *MockProductService) ValidateProductOrder(ctx context.Context, requestBody interface{}) error {
	return m.ValidateProductOrderErr
}

func (m *MockProductService) ListProductResourceTags(ctx context.Context, productID string) ([]megaport.ResourceTag, error) {
	return m.ListProductResourceTagsResult, m.ListProductResourceTagsErr
}

func (m *MockProductService) UpdateProductResourceTags(ctx context.Context, productUID string, tagsReq *megaport.UpdateProductResourceTagsRequest) error {
	return m.UpdateProductResourceTagsErr
}

func (m *MockProductService) GetProductPricing(ctx context.Context, req megaport.PriceBookRequest) (*megaport.PriceBookDTO, error) {
	return m.GetProductPricingResult, m.GetProductPricingErr
}

func (m *MockProductService) GetProductPricingForCompany(ctx context.Context, req *megaport.GetProductPricingRequest) (*megaport.PriceBookDTO, error) {
	return m.GetProductPricingForCompanyResult, m.GetProductPricingForCompanyErr
}
