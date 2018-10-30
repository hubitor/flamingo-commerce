package application_test

import (
	"context"
	"testing"

	"reflect"

	cartApplication "flamingo.me/flamingo-commerce/cart/application"
	cartDomain "flamingo.me/flamingo-commerce/cart/domain/cart"
	cartInfrastructure "flamingo.me/flamingo-commerce/cart/infrastructure"
	productDomain "flamingo.me/flamingo-commerce/product/domain"
	authApplication "flamingo.me/flamingo/core/auth/application"
	"flamingo.me/flamingo/framework/flamingo"
	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
)

// MockGuestCartServiceAdapter

type (
	MockGuestCartServiceAdapter struct{}
)

func (m *MockGuestCartServiceAdapter) GetCart(ctx context.Context, cartId string) (*cartDomain.Cart, error) {
	return &cartDomain.Cart{
		ID: "mock_guest_cart",
	}, nil
}

func (m *MockGuestCartServiceAdapter) GetNewCart(ctx context.Context) (*cartDomain.Cart, error) {
	return &cartDomain.Cart{
		ID: "mock_guest_cart",
	}, nil
}

func (m *MockGuestCartServiceAdapter) GetCartOrderBehaviour(context.Context) (cartDomain.CartBehaviour, error) {
	cartBehaviour := new(cartInfrastructure.InMemoryCartOrderBehaviour)

	return cartBehaviour, nil
}

// MockGuestCartServiceAdapter with error on GetCart

type (
	MockGuestCartServiceAdapterError struct{}
)

func (m *MockGuestCartServiceAdapterError) GetCart(ctx context.Context, cartId string) (*cartDomain.Cart, error) {
	return nil, errors.New("defective")
}

func (m *MockGuestCartServiceAdapterError) GetNewCart(ctx context.Context) (*cartDomain.Cart, error) {
	return nil, errors.New("defective")
}

func (m *MockGuestCartServiceAdapterError) GetCartOrderBehaviour(context.Context) (cartDomain.CartBehaviour, error) {
	cartBehaviour := new(cartInfrastructure.InMemoryCartOrderBehaviour)

	return cartBehaviour, nil
}

// MockCustomerCartService

type (
	MockCustomerCartService struct{}
)

func (m *MockCustomerCartService) GetCartOrderBehaviour(context.Context, cartDomain.Auth) (cartDomain.CartBehaviour, error) {
	cartBehaviour := new(cartInfrastructure.InMemoryCartOrderBehaviour)

	return cartBehaviour, nil
}

func (m *MockCustomerCartService) GetCart(ctx context.Context, auth cartDomain.Auth, cartId string) (*cartDomain.Cart, error) {
	return &cartDomain.Cart{
		ID: "mock_customer_cart",
	}, nil
}

// MockProductService

type (
	MockProductService struct{}
)

func (m *MockProductService) Get(ctx context.Context, marketplaceCode string) (productDomain.BasicProduct, error) {
	mockProduct := new(productDomain.SimpleProduct)

	mockProduct.Identifier = "mock_product"

	return mockProduct, nil
}

// MockCartCache

type (
	MockCartCache struct{}
)

func (m *MockCartCache) GetCart(context.Context, *sessions.Session, cartApplication.CartCacheIdentifier) (*cartDomain.Cart, error) {
	return &cartDomain.Cart{}, nil
}

func (m *MockCartCache) CacheCart(context.Context, *sessions.Session, cartApplication.CartCacheIdentifier, *cartDomain.Cart) error {
	return nil
}

func (m *MockCartCache) Invalidate(context.Context, *sessions.Session, cartApplication.CartCacheIdentifier) error {
	return nil
}

func (m *MockCartCache) Delete(context.Context, *sessions.Session, cartApplication.CartCacheIdentifier) error {
	return nil
}

func (m *MockCartCache) DeleteAll(context.Context, *sessions.Session) error {
	return nil
}

// MockEventPublisher

type (
	MockEventPublisher struct{}
)

func (m *MockEventPublisher) PublishOrderPlacedEvent(ctx context.Context, cart *cartDomain.Cart, placedOrderInfos cartDomain.PlacedOrderInfos) {
}

func (m *MockEventPublisher) PublishAddToCartEvent(ctx context.Context, marketPlaceCode string, variantMarketPlaceCode string, qty int) {
}

func (m *MockEventPublisher) PublishChangedQtyInCartEvent(ctx context.Context, item *cartDomain.Item, qtyBefore int, qtyAfter int, cartID string) {
}

// MockCartValidator

type (
	MockCartValidator struct{}
)

func (m *MockCartValidator) Validate(ctx context.Context, session *sessions.Session, cart *cartDomain.DecoratedCart) cartDomain.CartValidationResult {
	return cartDomain.CartValidationResult{}
}

// MockItemValidator

type (
	MockItemValidator struct{}
)

func (m *MockItemValidator) Validate(ctx context.Context, session *sessions.Session, deliveryCode string, request cartDomain.AddRequest, product productDomain.BasicProduct) error {
	return nil
}

// MockDeliveryInfoBuilder

type (
	MockDeliveryInfoBuilder struct{}
)

func (m *MockDeliveryInfoBuilder) BuildByDeliveryCode(deliveryCode string) (cartDomain.DeliveryInfo, error) {
	return cartDomain.DeliveryInfo{}, nil
}

func TestCartReceiverService_ShouldHaveGuestCart(t *testing.T) {
	type fields struct {
		GuestCartService     cartDomain.GuestCartService
		CustomerCartService  cartDomain.CustomerCartService
		CartDecoratorFactory *cartDomain.DecoratedCartFactory
		AuthManager          *authApplication.AuthManager
		UserService          *authApplication.UserService
		Logger               flamingo.Logger
		CartCache            cartApplication.CartCache
	}
	type args struct {
		session *sessions.Session
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "has session key",
			fields: fields{
				GuestCartService:    new(MockGuestCartServiceAdapter),
				CustomerCartService: new(MockCustomerCartService),
				CartDecoratorFactory: &cartDomain.DecoratedCartFactory{
					ProductService: &MockProductService{},
				},
				AuthManager: &authApplication.AuthManager{},
				UserService: &authApplication.UserService{},
				Logger:      flamingo.NullLogger{},
				CartCache:   new(MockCartCache),
			},
			args: args{
				session: &sessions.Session{
					Values: map[interface{}]interface{}{
						cartApplication.GuestCartSessionKey: struct{}{},
					},
				},
			},
			want: true,
		}, {
			name: "doesn't have session key",
			fields: fields{
				GuestCartService:    new(MockGuestCartServiceAdapter),
				CustomerCartService: new(MockCustomerCartService),
				CartDecoratorFactory: &cartDomain.DecoratedCartFactory{
					ProductService: &MockProductService{},
				},
				AuthManager: &authApplication.AuthManager{},
				UserService: &authApplication.UserService{},
				Logger:      flamingo.NullLogger{},
				CartCache:   new(MockCartCache),
			},
			args: args{
				session: &sessions.Session{
					Values: map[interface{}]interface{}{
						"arbitrary_and_wrong_key": struct{}{},
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &cartApplication.CartReceiverService{
				GuestCartService:     tt.fields.GuestCartService,
				CustomerCartService:  tt.fields.CustomerCartService,
				CartDecoratorFactory: tt.fields.CartDecoratorFactory,
				AuthManager:          tt.fields.AuthManager,
				UserService:          tt.fields.UserService,
				Logger:               tt.fields.Logger,
				CartCache:            tt.fields.CartCache,
			}

			got := cs.ShouldHaveGuestCart(tt.args.session)

			if got != tt.want {
				t.Errorf("CartReceiverService.ShouldHaveGuestCart() = %v, wantType0 %v", got, tt.want)
			}
		})
	}
}

func TestCartReceiverService_ViewGuestCart(t *testing.T) {
	type fields struct {
		GuestCartService     cartDomain.GuestCartService
		CustomerCartService  cartDomain.CustomerCartService
		CartDecoratorFactory *cartDomain.DecoratedCartFactory
		AuthManager          *authApplication.AuthManager
		UserService          *authApplication.UserService
		Logger               flamingo.Logger
		CartCache            cartApplication.CartCache
	}
	type args struct {
		ctx     context.Context
		session *sessions.Session
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		want           *cartDomain.Cart
		wantErr        bool
		wantMessageErr string
	}{
		{
			name: "has no guest cart",
			fields: fields{
				GuestCartService:    new(MockGuestCartServiceAdapter),
				CustomerCartService: new(MockCustomerCartService),
				CartDecoratorFactory: &cartDomain.DecoratedCartFactory{
					ProductService: &MockProductService{},
				},
				AuthManager: &authApplication.AuthManager{},
				UserService: &authApplication.UserService{},
				Logger:      flamingo.NullLogger{},
				CartCache:   new(MockCartCache),
			},
			args: args{
				ctx: context.Background(),
				session: &sessions.Session{
					Values: map[interface{}]interface{}{
						"stuff": "some_malformed_id",
					},
				},
			},
			want:           &cartDomain.Cart{},
			wantErr:        false,
			wantMessageErr: "",
		}, {
			name: "failed guest cart get",
			fields: fields{
				GuestCartService:    new(MockGuestCartServiceAdapterError),
				CustomerCartService: new(MockCustomerCartService),
				CartDecoratorFactory: &cartDomain.DecoratedCartFactory{
					ProductService: &MockProductService{},
				},
				AuthManager: &authApplication.AuthManager{},
				UserService: &authApplication.UserService{},
				Logger:      flamingo.NullLogger{},
				CartCache:   new(MockCartCache),
			},
			args: args{
				ctx: context.Background(),
				session: &sessions.Session{
					Values: map[interface{}]interface{}{
						cartApplication.GuestCartSessionKey: "some_guest_id",
					},
				},
			},
			want:           nil,
			wantErr:        true,
			wantMessageErr: cartApplication.ErrTemporaryCartService.Error(),
		}, {
			name: "guest cart get without error",
			fields: fields{
				GuestCartService:    new(MockGuestCartServiceAdapter),
				CustomerCartService: new(MockCustomerCartService),
				CartDecoratorFactory: &cartDomain.DecoratedCartFactory{
					ProductService: &MockProductService{},
				},
				AuthManager: &authApplication.AuthManager{},
				UserService: &authApplication.UserService{},
				Logger:      flamingo.NullLogger{},
				CartCache:   new(MockCartCache),
			},
			args: args{
				ctx: context.Background(),
				session: &sessions.Session{
					Values: map[interface{}]interface{}{
						cartApplication.GuestCartSessionKey: "some_guest_id",
					},
				},
			},
			want: &cartDomain.Cart{
				ID: "mock_guest_cart",
			},
			wantErr:        false,
			wantMessageErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &cartApplication.CartReceiverService{
				GuestCartService:     tt.fields.GuestCartService,
				CustomerCartService:  tt.fields.CustomerCartService,
				CartDecoratorFactory: tt.fields.CartDecoratorFactory,
				AuthManager:          tt.fields.AuthManager,
				UserService:          tt.fields.UserService,
				Logger:               tt.fields.Logger,
				CartCache:            tt.fields.CartCache,
			}

			got, err := cs.ViewGuestCart(tt.args.ctx, tt.args.session)
			if (err != nil) != tt.wantErr {
				t.Errorf("CartReceiverService.ViewGuestCart() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if err != nil && err.Error() != tt.wantMessageErr {
				t.Errorf("Error doesn't match - error = %v, wantMessageErr %v", err, tt.wantMessageErr)

				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CartReceiverService.ViewGuestCart() = %v, wantType0 %v", got, tt.want)
			}
		})
	}
}

func TestCartService_DeleteSavedSessionGuestCartID(t *testing.T) {
	type fields struct {
		CartReceiverService *cartApplication.CartReceiverService
		ProductService      productDomain.ProductService
		Logger              flamingo.Logger
		CartValidator       cartDomain.CartValidator
		ItemValidator       cartDomain.ItemValidator
		EventPublisher      cartApplication.EventPublisher
		DefaultDeliveryCode string
		DeliveryInfoBuilder cartDomain.DeliveryInfoBuilder
		CartCache           cartApplication.CartCache
	}
	type args struct {
		session *sessions.Session
	}
	tests := []struct {
		name          string
		fields        fields
		args          args
		wantErr       bool
		valuesCleared bool
	}{
		{
			name: "basic clearing of guest cart session value",
			fields: fields{
				CartReceiverService: &cartApplication.CartReceiverService{
					GuestCartService:    new(MockGuestCartServiceAdapter),
					CustomerCartService: new(MockCustomerCartService),
					CartDecoratorFactory: &cartDomain.DecoratedCartFactory{
						ProductService: &MockProductService{},
					},
					AuthManager: &authApplication.AuthManager{},
					UserService: &authApplication.UserService{},
					Logger:      flamingo.NullLogger{},
					CartCache:   new(MockCartCache),
				},
				ProductService:      &MockProductService{},
				Logger:              flamingo.NullLogger{},
				CartValidator:       new(MockCartValidator),
				ItemValidator:       new(MockItemValidator),
				EventPublisher:      new(MockEventPublisher),
				DefaultDeliveryCode: "default_delivery_code",
				DeliveryInfoBuilder: new(MockDeliveryInfoBuilder),
				CartCache:           new(MockCartCache),
			},
			args: args{
				session: &sessions.Session{
					Values: map[interface{}]interface{}{
						cartApplication.GuestCartSessionKey: "some_guest_id",
					},
				},
			},
			wantErr:       false,
			valuesCleared: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &cartApplication.CartService{
				CartReceiverService: tt.fields.CartReceiverService,
				ProductService:      tt.fields.ProductService,
				Logger:              tt.fields.Logger,
				CartValidator:       tt.fields.CartValidator,
				ItemValidator:       tt.fields.ItemValidator,
				EventPublisher:      tt.fields.EventPublisher,
				DefaultDeliveryCode: tt.fields.DefaultDeliveryCode,
				DeliveryInfoBuilder: tt.fields.DeliveryInfoBuilder,
				CartCache:           tt.fields.CartCache,
			}

			err := cs.DeleteSavedSessionGuestCartID(tt.args.session)

			if (err != nil) != tt.wantErr {
				t.Errorf("CartService.DeleteSavedSessionGuestCartID() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if tt.valuesCleared == true {
				if len(tt.args.session.Values) > 0 {
					t.Error("Session Values should be empty, but aren't")
				}
			}
		})
	}
}

func TestCartReceiverService_DecorateCart(t *testing.T) {
	type fields struct {
		GuestCartService     cartDomain.GuestCartService
		CustomerCartService  cartDomain.CustomerCartService
		CartDecoratorFactory *cartDomain.DecoratedCartFactory
		AuthManager          *authApplication.AuthManager
		UserService          *authApplication.UserService
		Logger               flamingo.Logger
		CartCache            cartApplication.CartCache
	}
	type args struct {
		ctx  context.Context
		cart *cartDomain.Cart
	}
	tests := []struct {
		name                     string
		fields                   fields
		args                     args
		want                     *cartDomain.DecoratedCart
		wantErr                  bool
		wantMessageErr           string
		wantDecoratedItemsLength int
	}{
		{
			name: "error b/c no cart supplied",
			fields: fields{
				GuestCartService:    new(MockGuestCartServiceAdapter),
				CustomerCartService: new(MockCustomerCartService),
				CartDecoratorFactory: &cartDomain.DecoratedCartFactory{
					ProductService: &MockProductService{},
				},
				AuthManager: &authApplication.AuthManager{},
				UserService: &authApplication.UserService{},
				Logger:      flamingo.NullLogger{},
				CartCache:   new(MockCartCache),
			},
			args: args{
				ctx:  context.Background(),
				cart: nil,
			},
			wantErr:                  true,
			wantMessageErr:           "no cart given",
			wantDecoratedItemsLength: 0,
		}, {
			name: "basic decoration of cart",
			fields: fields{
				GuestCartService:    new(MockGuestCartServiceAdapter),
				CustomerCartService: new(MockCustomerCartService),
				CartDecoratorFactory: &cartDomain.DecoratedCartFactory{
					ProductService: &MockProductService{},
				},
				AuthManager: &authApplication.AuthManager{},
				UserService: &authApplication.UserService{},
				Logger:      flamingo.NullLogger{},
				CartCache:   new(MockCartCache),
			},
			args: args{
				ctx: context.Background(),
				cart: &cartDomain.Cart{
					ID: "some_test_cart",
					Deliveries: []cartDomain.Delivery{
						{
							Cartitems: []cartDomain.Item{
								{
									ID: "test_id",
								},
							},
						},
					},
				},
			},
			wantErr:                  false,
			wantMessageErr:           "",
			wantDecoratedItemsLength: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &cartApplication.CartReceiverService{
				GuestCartService:     tt.fields.GuestCartService,
				CustomerCartService:  tt.fields.CustomerCartService,
				CartDecoratorFactory: tt.fields.CartDecoratorFactory,
				AuthManager:          tt.fields.AuthManager,
				UserService:          tt.fields.UserService,
				Logger:               tt.fields.Logger,
				CartCache:            tt.fields.CartCache,
			}

			got, err := cs.DecorateCart(tt.args.ctx, tt.args.cart)

			if (err != nil) != tt.wantErr {
				t.Errorf("CartReceiverService.DecorateCart() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if err != nil && err.Error() != tt.wantMessageErr {
				t.Errorf("Error doesn't match - error = %v, wantMessageErr %v", err, tt.wantMessageErr)

				return
			}

			if tt.wantDecoratedItemsLength > 0 {
				for _, decoratedDeliveryItem := range got.DecoratedDeliveries {
					if len(decoratedDeliveryItem.DecoratedItems) != tt.wantDecoratedItemsLength {
						t.Errorf("Mismatch of expected Decorated Items, got %d, expected %d", len(decoratedDeliveryItem.DecoratedItems), tt.wantDecoratedItemsLength)
					}
				}
			}
		})
	}
}

func TestCartReceiverService_GetDecoratedCart(t *testing.T) {
	type fields struct {
		GuestCartService     cartDomain.GuestCartService
		CustomerCartService  cartDomain.CustomerCartService
		CartDecoratorFactory *cartDomain.DecoratedCartFactory
		AuthManager          *authApplication.AuthManager
		UserService          *authApplication.UserService
		Logger               flamingo.Logger
		CartCache            cartApplication.CartCache
	}
	type args struct {
		ctx     context.Context
		session *sessions.Session
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantType0 *cartDomain.DecoratedCart
		wantType1 cartDomain.CartBehaviour
		wantErr   bool
	}{
		{
			name: "decorated cart not found",
			fields: fields{
				GuestCartService:    new(MockGuestCartServiceAdapterError),
				CustomerCartService: new(MockCustomerCartService),
				CartDecoratorFactory: &cartDomain.DecoratedCartFactory{
					ProductService: &MockProductService{},
				},
				AuthManager: &authApplication.AuthManager{},
				UserService: &authApplication.UserService{},
				Logger:      flamingo.NullLogger{},
				CartCache:   new(MockCartCache),
			},
			args: args{
				ctx: context.Background(),
				session: &sessions.Session{
					Values: map[interface{}]interface{}{
						"some_nonvalid_key": "some_guest_id",
					},
				},
			},
			wantType0: nil,
			wantType1: nil,
			wantErr:   true,
		}, {
			name: "decorated cart found",
			fields: fields{
				GuestCartService:    new(MockGuestCartServiceAdapter),
				CustomerCartService: new(MockCustomerCartService),
				CartDecoratorFactory: &cartDomain.DecoratedCartFactory{
					ProductService: &MockProductService{},
				},
				AuthManager: &authApplication.AuthManager{},
				UserService: &authApplication.UserService{},
				Logger:      flamingo.NullLogger{},
				CartCache:   new(MockCartCache),
			},
			args: args{
				ctx: context.Background(),
				session: &sessions.Session{
					Values: map[interface{}]interface{}{
						"some_nonvalid_key": "some_guest_id",
					},
				},
			},
			wantType0: &cartDomain.DecoratedCart{},
			wantType1: &cartInfrastructure.InMemoryCartOrderBehaviour{},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &cartApplication.CartReceiverService{
				GuestCartService:     tt.fields.GuestCartService,
				CustomerCartService:  tt.fields.CustomerCartService,
				CartDecoratorFactory: tt.fields.CartDecoratorFactory,
				AuthManager:          tt.fields.AuthManager,
				UserService:          tt.fields.UserService,
				Logger:               tt.fields.Logger,
				CartCache:            tt.fields.CartCache,
			}

			got, got1, err := cs.GetDecoratedCart(tt.args.ctx, tt.args.session)

			if (err != nil) != tt.wantErr {
				t.Errorf("CartReceiverService.GetDecoratedCart() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if tt.wantType0 == nil {
				if !reflect.DeepEqual(got, tt.wantType0) {
					t.Errorf("CartReceiverService.GetDecoratedCart() got = %v, wantType0 %v", got, tt.wantType0)

					return
				}
			} else {
				gotType := reflect.TypeOf(got).Elem()
				wantType := reflect.TypeOf(tt.wantType0).Elem()
				if wantType != gotType {
					t.Error("Return Type for wantType0 doesn't match")

					return
				}
			}

			if tt.wantType1 == nil {
				if !reflect.DeepEqual(got1, tt.wantType1) {
					t.Errorf("CartReceiverService.GetDecoratedCart() got = %v, wantType0 %v", got1, tt.wantType1)

					return
				}
			} else {
				gotType1 := reflect.TypeOf(got1).Elem()
				wantType1 := reflect.TypeOf(tt.wantType1).Elem()
				if wantType1 != gotType1 {
					t.Error("Return Type for wantType0 doesn't match")

					return
				}
			}

			if !reflect.DeepEqual(got1, tt.wantType1) {
				t.Errorf("CartReceiverService.GetDecoratedCart() got1 = %v, wantType0 %v", got1, tt.wantType1)

				return
			}
		})
	}
}