package cart

import (
	"flamingo.me/dingo"
	"flamingo.me/flamingo/v3/framework/flamingo"
	"flamingo.me/flamingo/v3/framework/web"
	"flamingo.me/form"
	formDomain "flamingo.me/form/domain"
	flamingographql "flamingo.me/graphql"

	"flamingo.me/flamingo-commerce/v3/cart/application"
	"flamingo.me/flamingo-commerce/v3/cart/domain/cart"
	"flamingo.me/flamingo-commerce/v3/cart/domain/events"
	"flamingo.me/flamingo-commerce/v3/cart/domain/placeorder"
	"flamingo.me/flamingo-commerce/v3/cart/infrastructure"
	placeorderAdapter "flamingo.me/flamingo-commerce/v3/cart/infrastructure/placeorder"
	"flamingo.me/flamingo-commerce/v3/cart/interfaces/controller"
	"flamingo.me/flamingo-commerce/v3/cart/interfaces/controller/forms"
	"flamingo.me/flamingo-commerce/v3/cart/interfaces/graphql"
	"flamingo.me/flamingo-commerce/v3/cart/interfaces/templatefunctions"
	"flamingo.me/flamingo-commerce/v3/customer"
	"flamingo.me/flamingo-commerce/v3/product"
)

type (
	// Module registers our profiler
	Module struct {
		routerRegistry                *web.RouterRegistry
		enableDefaultCartAdapter      bool
		enablePlaceOrderLoggerAdapter bool
		enableCartCache               bool
	}
)

// Inject dependencies
func (m *Module) Inject(
	routerRegistry *web.RouterRegistry,
	config *struct {
		EnableDefaultCartAdapter      bool `inject:"config:commerce.cart.defaultCartAdapter.enabled,optional"`
		EnableCartCache               bool `inject:"config:commerce.cart.enableCartCache,optional"`
		EnablePlaceOrderLoggerAdapter bool `inject:"config:commerce.cart.placeOrderLogger.enabled,optional"`
	},
) {
	m.routerRegistry = routerRegistry
	if config != nil {
		m.enableDefaultCartAdapter = config.EnableDefaultCartAdapter
		m.enableCartCache = config.EnableCartCache
		m.enablePlaceOrderLoggerAdapter = config.EnablePlaceOrderLoggerAdapter
	}
}

// Configure module
func (m *Module) Configure(injector *dingo.Injector) {
	if m.enableDefaultCartAdapter {
		injector.Bind((*infrastructure.CartStorage)(nil)).To(infrastructure.InMemoryCartStorage{}).AsEagerSingleton()
		injector.Bind((*infrastructure.GiftCardHandler)(nil)).To(infrastructure.DefaultGiftCardHandler{})
		injector.Bind((*infrastructure.VoucherHandler)(nil)).To(infrastructure.DefaultVoucherHandler{})
		injector.Bind((*cart.GuestCartService)(nil)).To(infrastructure.DefaultGuestCartService{})
		injector.Bind((*cart.CustomerCartService)(nil)).To(infrastructure.DefaultCustomerCartService{})
	}
	if m.enablePlaceOrderLoggerAdapter {
		injector.Bind((*placeorder.Service)(nil)).To(placeorderAdapter.PlaceOrderLoggerAdapter{})
	}
	// Register Default EventPublisher
	injector.Bind((*events.EventPublisher)(nil)).To(events.DefaultEventPublisher{})

	// Event
	flamingo.BindEventSubscriber(injector).To(application.EventReceiver{})

	// TemplateFunction
	flamingo.BindTemplateFunc(injector, "getCart", new(templatefunctions.GetCart))
	flamingo.BindTemplateFunc(injector, "getDecoratedCart", new(templatefunctions.GetDecoratedCart))
	flamingo.BindTemplateFunc(injector, "getQuantityAdjustmentDeletedItemsMessages", new(templatefunctions.GetQuantityAdjustmentDeletedItemsMessages))
	flamingo.BindTemplateFunc(injector, "getQuantityAdjustmentUpdatedItemsMessages", new(templatefunctions.GetQuantityAdjustmentUpdatedItemsMessage))
	flamingo.BindTemplateFunc(injector, "getQuantityAdjustmentCouponCodesRemoved", new(templatefunctions.GetQuantityAdjustmentCouponCodesRemoved))
	flamingo.BindTemplateFunc(injector, "removeQuantityAdjustmentMessages", new(templatefunctions.RemoveQuantityAdjustmentMessages))

	injector.Bind((*cart.DeliveryInfoBuilder)(nil)).To(cart.DefaultDeliveryInfoBuilder{})

	if m.enableCartCache {
		injector.Bind((*application.CartCache)(nil)).To(application.CartSessionCache{})
	}

	// Register Form Data Provider
	injector.BindMap(new(formDomain.FormService), "commerce.cart.deliveryFormService").To(forms.DeliveryFormService{})
	injector.BindMap(new(formDomain.FormService), "commerce.cart.billingFormService").To(forms.BillingAddressFormService{})
	injector.BindMap(new(formDomain.FormService), "commerce.cart.personaldataFormService").To(forms.DefaultPersonalDataFormService{})

	web.BindRoutes(injector, new(routes))

	injector.BindMulti(new(flamingographql.Service)).To(graphql.Service{})
}

// CueConfig defines the cart module configuration
func (*Module) CueConfig() string {
	return `
commerce: {
	cart: {
		defaultCartAdapter: {
			enabled: bool | *true
			storage: "inmemory"
			defaultTaxRate?: number
		}
		placeOrderLogger: {
			enabled: bool | *true
			useFlamingoLog: bool | *true
			logAsFile: bool | *true
			logDirectory: string | *"./orders/"
		}
		enableCartCache: bool | *true
		cacheLifetime: number | *1200
		defaultUseBillingAddress: bool | *false
		defaultDeliveryCode: string | *"delivery"
		deleteEmptyDelivery: bool | *false
		showEmptyCartPageIfNoItems?: bool
		adjustItemsToRestrictedQty?: bool
		personalDataForm: {
			additionalFormFields: [...string] | *[]
			dateOfBirthRequired: bool | *false
			passportCountryRequired: bool | *false
			passportNumberRequired: bool | *false
			minAge?: number
		}
		simplePaymentForm: {
			giftCardPaymentMethod: string | *"voucher"
		}
	}
}`
}

// FlamingoLegacyConfigAlias mapping
func (*Module) FlamingoLegacyConfigAlias() map[string]string {
	return map[string]string{
		"commerce.cart.useEmailPlaceOrderAdapter":                 "commerce.cart.placeOrderLogger.enabled",
		"commerce.cart.useInMemoryCartServiceAdapters":            "commerce.cart.defaultCartAdapter.enabled",
		"commerce.cart.inMemoryCartServiceAdapter.defaultTaxRate": "commerce.cart.defaultCartAdapter.defaultTaxRate",
	}
}

// Depends on other modules
func (m *Module) Depends() []dingo.Module {
	return []dingo.Module{
		new(product.Module),
		new(form.Module),
		new(customer.Module),
	}
}

type routes struct {
	viewController *controller.CartViewController
	apiController  *controller.CartAPIController
}

func (r *routes) Inject(viewController *controller.CartViewController, apiController *controller.CartAPIController) {
	r.viewController = viewController
	r.apiController = apiController
}

func (r *routes) Routes(registry *web.RouterRegistry) {
	registry.HandleAny("cart.view", r.viewController.ViewAction)
	registry.Route("/cart", "cart.view")

	registry.HandleAny("cart.add", r.viewController.AddAndViewAction)
	registry.Route("/cart/add/:marketplaceCode", `cart.add(marketplaceCode,variantMarketplaceCode?="",qty?="1",deliveryCode?="")`)

	registry.HandleAny("cart.updateQty", r.viewController.UpdateQtyAndViewAction)
	registry.Route("/cart/update/:id", `cart.updateQty(id,qty?="1",deliveryCode?="")`)

	registry.HandleAny("cart.deleteAllItems", r.viewController.DeleteAllAndViewAction)
	registry.Route("/cart/delete/all", "cart.deleteAllItems")

	registry.HandleAny("cart.clean", r.viewController.CleanAndViewAction)
	registry.Route("/cart/clean", "cart.clean")

	registry.HandleDelete("cart.clean", r.viewController.CleanAndViewAction)
	registry.Route("/cart/delivery/:deliveryCode", "cart.clean")

	registry.HandleAny("cart.cleanDelivery", r.viewController.CleanDeliveryAndViewAction)
	registry.Route("/cart/delete/delivery/:deliveryCode", `cart.cleanDelivery(deliveryCode?="")`)

	registry.HandleAny("cart.deleteItem", r.viewController.DeleteAndViewAction)
	registry.Route("/cart/delete/:id", `cart.deleteItem(id,deliveryCode?="")`)
	r.apiRoutes(registry)
}

func (r *routes) apiRoutes(registry *web.RouterRegistry) {

	registry.Route("/api/cart", "cart.api.get")
	registry.Route("/api/v1/cart", "cart.api.get")
	registry.HandleDelete("cart.api.get", r.apiController.DeleteCartAction)
	registry.HandleGet("cart.api.get", r.apiController.GetAction)

	// add command under the delivery:
	registry.Route("/api/cart/delivery/:deliveryCode/additem", `cart.api.add(marketplaceCode,variantMarketplaceCode?="",qty?="1",deliveryCode?="")`)
	registry.Route("/api/v1/cart/delivery/:deliveryCode/additem", `cart.api.add(marketplaceCode,variantMarketplaceCode?="",qty?="1",deliveryCode?="")`)

	registry.HandlePost("cart.api.add", r.apiController.AddAction)

	registry.Route("/api/cart/applyvoucher", `cart.api.applyVoucher(couponCode)`)
	registry.Route("/api/v1/cart/applyvoucher", `cart.api.applyVoucher(couponCode)`)

	registry.HandlePost("cart.api.applyVoucher", r.apiController.ApplyVoucherAndGetAction)
	registry.HandlePut("cart.api.applyVoucher", r.apiController.ApplyVoucherAndGetAction)

	registry.Route("/api/cart/removevoucher", `cart.api.removeVoucher(couponCode)`)
	registry.Route("/api/v1/cart/removevoucher", `cart.api.removeVoucher(couponCode)`)
	registry.HandlePost("cart.api.removeVoucher", r.apiController.RemoveVoucherAndGetAction)
	registry.HandleDelete("cart.api.removeVoucher", r.apiController.RemoveVoucherAndGetAction)

	registry.Route("/api/cart/applygiftcard", `cart.api.applyGiftCard(couponCode)`)
	registry.Route("/api/v1/cart/applygiftcard", `cart.api.applyGiftCard(couponCode)`)
	registry.HandlePost("cart.api.applyGiftCard", r.apiController.ApplyGiftCardAndGetAction)
	registry.HandlePut("cart.api.applyGiftCard", r.apiController.ApplyGiftCardAndGetAction)

	registry.Route("/api/cart/removegiftcard", `cart.api.removeGiftCard(couponCode)`)
	registry.Route("/api/v1/cart/removegiftcard", `cart.api.removeGiftCard(couponCode)`)
	registry.HandlePost("cart.api.removeGiftCard", r.apiController.RemoveGiftCardAndGetAction)
	registry.HandleDelete("cart.api.removeGiftCard", r.apiController.RemoveGiftCardAndGetAction)

	registry.Route("/api/cart/applycombinedvouchergift", `cart.api.applyCombinedVoucherGift(couponCode)`)
	registry.Route("/api/v1/cart/applycombinedvouchergift", `cart.api.applyCombinedVoucherGift(couponCode)`)
	registry.HandlePost("cart.api.applyCombinedVoucherGift", r.apiController.ApplyCombinedVoucherGift)

	registry.Route("/api/cart/billing", `cart.api.billing`)
	registry.Route("/api/v1/cart/billing", `cart.api.billing`)
	registry.HandlePost("cart.api.billing", r.apiController.BillingAction)

	registry.Route("/api/cart/delivery/:deliveryCode", `cart.api.delivery.delete`)
	registry.Route("/api/v1/cart/delivery/:deliveryCode", `cart.api.delivery.delete`)
	registry.HandleDelete("cart.api.delivery.delete", r.apiController.DeleteDelivery)

	registry.Route("/api/cart/delivery/:deliveryCode/deliveryinfo", `cart.api.delivery.update`)
	registry.Route("/api/v1/cart/delivery/:deliveryCode/deliveryinfo", `cart.api.delivery.update`)
	registry.HandlePost("cart.api.delivery.update", r.apiController.UpdateDeliveryInfoAction)

	registry.Route("/api/v1/cart/updatepaymentselection", `cart.api.updatepaymentselection`)
	registry.HandlePut("cart.api.updatepaymentselection", r.apiController.UpdatePaymentSelectionAction)

	// registry.Route("/api/cart/delivery/:shipping", `cart.api.shipping(deliveryCode?="")`)
	// TODO registry.HandleDelete("cart.api.delivery", r.apiController.DeleteDelivery)
}
