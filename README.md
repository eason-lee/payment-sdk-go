# Payment Go SDK

Payment 商户网关 Go SDK，用于第三方商户服务端接入代收、代付、PayPal 签约、订单查询、代收退款、争议申诉和商户通知验签。

## 安装

```bash
go get github.com/eason-lee/payment-sdk-go@v0.1.11
```

## 创建客户端

```go
client := payment.NewClient(payment.Production())
```

当前环境：

- `Production()`：正式环境，地址待确认
- `Sandbox()`：beta 商户 API，`https://api-beta.winzay.top`
- `EnvTest`：本机 `edge-proxy`，`http://127.0.0.1:8083`

收款走 `/api/payin/**`，出款走 `/api/payout/**`。不要把 Admin 或渠道 webhook 配进 SDK。

如果后续正式环境域名独立，接入方只需要切换初始化环境：

```go
client := payment.NewClient(payment.Sandbox())
```

## 商户信息

每次请求都需要传入商户 ID 和商户密钥：

```go
merchant := payment.Merchant{
	ID:     "1001",
	Secret: "merchant-secret",
}
```

`Merchant.Secret` 只能保存在商户服务端，不能暴露到浏览器、移动端或前端包里。

## 创建代收订单

```go
resp, err := client.CreatePayin(ctx, payment.CreatePayinReq{
	Merchant:        merchant,
	MerchantOrderID: "PAYIN-202606090001",
	Amount:          10000,
	Currency:        payment.CurrencyTpUSD,
	PayMethod:       payment.PayMethodPayPal,
	PayMode:         payment.PayModePayPalAgreement,
	User: &payment.User{
		ID:      "u_1001",
		AppName: "DemoApp",
		Name:    "John Doe",
		Email:   "john@example.com",
		Phone:   "+10000000000",
	},
	Address: &payment.Address{
		Country: "US",
	},
	PayPal: &payment.PayPal{
		Email: "buyer@example.com",
	},
})
if err != nil {
	return err
}
```

返回值：

```go
fmt.Println(resp.OrderID)
fmt.Println(resp.ChannelOrderID)
fmt.Println(resp.Link)
```

如果 `resp.Link` 不为空，商户可以把用户跳转到该链接完成支付。

### Checkout 信用卡 Payment Session

Checkout 信用卡 Flow 使用 `PayMethodCreditCard` + `PayModeCreditFlow` 创建订单。商户服务端创建订单后，把返回的 `PaymentSession` 交给前端 Checkout SDK 完成卡信息采集和 3DS 流程。

```go
resp, err := client.CreatePayin(ctx, payment.CreatePayinReq{
	Merchant:        merchant,
	MerchantOrderID: "PAYIN-202606090002",
	Amount:          10000,
	Currency:        payment.CurrencyTpUSD,
	PayMethod:       payment.PayMethodCreditCard,
	PayMode:         payment.PayModeCreditFlow,
	User: &payment.User{
		ID:      "u_1001",
		AppName: "DemoApp",
		Name:    "John Doe",
		Email:   "john@example.com",
	},
	Address: &payment.Address{
		Country: "US",
		Address: "100 Main St",
		State:   "NY",
		City:    "New York",
		Zip:     "10001",
	},
})
if err != nil {
	return err
}
if resp.PaymentSession == nil {
	return errors.New("checkout payment session is empty")
}
fmt.Println(resp.OrderID)
fmt.Println(resp.PaymentSession.ID)
fmt.Println(resp.PaymentSession.Token)
fmt.Println(resp.PaymentSession.Secret)
```

### 存卡 SourceID

`PayModeCreditSourceID` 使用 `credit_card.source_id`。地址走顶层 `address`。

```go
resp, err := client.CreatePayin(ctx, payment.CreatePayinReq{
	Merchant:        merchant,
	MerchantOrderID: "PAYIN-202606090003",
	Amount:          10000,
	Currency:        payment.CurrencyTpUSD,
	PayMethod:       payment.PayMethodCreditCard,
	PayMode:         payment.PayModeCreditSourceID,
	Address: &payment.Address{
		Country: "US",
	},
	CreditCard: &payment.CreditCard{
		SourceID: "src_xxx",
	},
})
```

CreditFlow 第二阶段提交时，卡片快照字段与网关一致：`card_name`、`bin`、`last4`、`expiry_month`、`expiry_year`、`scheme`、`country`。

## 查询代收订单

```go
payin, err := client.GetPayin(ctx, payment.GetPayinReq{
	Merchant: merchant,
	OrderID:  "10201312003",
})
if err != nil {
	return err
}
if payin.Status == payment.PayinOrderStatusProcessing {
	// 处理中，请等待通知或继续查询。
}
if payin.HasRefund && payin.Refund != nil {
	// 关联退款见 payin.Refund
}
if payin.Dispute != nil {
	// 关联争议见 payin.Dispute
}
```

代收订单状态：

| 常量 | 值 | 说明 |
|---|---:|---|
| `PayinOrderStatusUnspecified` | `0` | 未知状态 |
| `PayinOrderStatusProcessing` | `1` | 处理中 |
| `PayinOrderStatusSuccess` | `2` | 支付成功 |
| `PayinOrderStatusFailed` | `3` | 支付失败 |
| `PayinOrderStatusRefunding` | `4` | 退款中 |
| `PayinOrderStatusRefunded` | `5` | 已退款 |
| `PayinOrderStatusCanceled` | `6` | 已取消 |

## 代收退款

```go
err := client.RefundPayin(ctx, payment.RefundPayinReq{
	Merchant: merchant,
	OrderID:  "10201312003",
	Amount:   10000,
})
if err != nil {
	return err
}
```

## 争议申诉

游戏或商户先自己生成申诉 PDF，再把可下载的 HTTPS 地址交给支付。支付会立刻拉取文件并提交渠道。渠道裁决仍走现有争议通知。

```go
err := client.AppealDispute(ctx, &payment.AppealDisputeReq{
	Merchant: merchant,
	OrderID:  "10201312003",
	FileURL:  "https://files.example/chargeback/packet.pdf",
	Notes:    "optional seller statement",
})
if err != nil {
	return err
}
```

## PayPal 签约

商户服务端先绑定协议，把返回的 `Link` 交给用户授权。授权完成后可调用 `ApprovePayPalAgreement`，或等 PayPal 回跳/通知。已生效协议用 `CancelPayPalAgreement` 取消。

```go
resp, err := client.BindPayPalAgreement(ctx, &payment.BindPayPalAgreementReq{
	Merchant: merchant,
	Currency: payment.CurrencyTpUSD,
	AppName:  "DemoApp",
	UserID:   "u_1001",
})
if err != nil {
	return err
}
fmt.Println(resp.Token)
fmt.Println(resp.Link)

err = client.ApprovePayPalAgreement(ctx, &payment.ApprovePayPalAgreementReq{
	Merchant: merchant,
	Token:    resp.Token,
})
if err != nil {
	return err
}

err = client.CancelPayPalAgreement(ctx, &payment.CancelPayPalAgreementReq{
	Merchant:    merchant,
	AppName:     "DemoApp",
	UserID:      "u_1001",
	PayPalEmail: "buyer@example.com",
})
if err != nil {
	return err
}
```

用户授权完成后，再用 `PayModePayPalAgreement` 建单扣款。签约结果也会走 `MerchantNotifyTpPayInAgreement` 通知。

## 创建代付订单

```go
resp, err := client.CreatePayout(ctx, payment.CreatePayoutReq{
	Merchant:        merchant,
	MerchantOrderID: "PAYOUT-202606090001",
	Amount:          50000,
	Currency:        payment.CurrencyTpUSD,
	PayMethod:       payment.PayMethodPayPal,
	Account:         "receiver@example.com",
	User: &payment.User{
		ID:      "u_1001",
		AppName: "DemoApp",
		Name:    "John Doe",
		Email:   "john@example.com",
		Phone:   "+10000000000",
	},
})
if err != nil {
	return err
}
```

返回值：

```go
fmt.Println(resp.OrderID)
```

## 查询代付订单

```go
payout, err := client.GetPayout(ctx, payment.GetPayoutReq{
	Merchant: merchant,
	OrderID:  "10201312010",
})
if err != nil {
	return err
}
if payout.Status == payment.PayoutOrderStatusProcessing {
	// 处理中，请等待通知或继续查询。
}
```

代付订单状态：

| 常量 | 值 | 说明 |
|---|---:|---|
| `PayoutOrderStatusUnspecified` | `0` | 未知状态 |
| `PayoutOrderStatusProcessing` | `1` | 处理中 |
| `PayoutOrderStatusSuccess` | `2` | 代付成功 |
| `PayoutOrderStatusFailed` | `3` | 代付失败 |

## 接收商户通知

通知验签必须使用 HTTP 请求的原始 body，不能先反序列化再重新序列化。

```go
func notifyHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}

	client := payment.NewClient(payment.Production())

	notify := &payment.Notify{
		Header: r.Header,
		Body:   body,
	}

	identity, err := client.GetNotifyIdentity(notify)
	if err != nil {
		client.NotifyFailed(w)
		return
	}

	secret, err := loadMerchantSecret(r.Context(), identity.MerchantID, identity.OrderID)
	if err != nil {
		client.NotifyFailed(w)
		return
	}

	event, err := client.ParseNotify(r.Context(), secret, notify)
	if err != nil {
		client.NotifyFailed(w)
		return
	}

	switch event.Tp {
	case payment.MerchantNotifyTpPayIn:
		// 按 event.PayinOrder.OrderID 或 MerchantOrderId 做幂等处理。
	case payment.MerchantNotifyTpPayOut:
		// 按 event.PayoutOrder.OrderID 或 MerchantOrderId 做幂等处理。
	case payment.MerchantNotifyTpPayInRefund:
		// 处理代收退款通知。
	case payment.MerchantNotifyTpPayInDispute:
		// 处理代收争议通知。
	case payment.MerchantNotifyTpPayInAgreement:
		// 处理代收协议签约通知。
	case payment.MerchantNotifyTpPayInCardBind:
		// 处理信用卡绑卡通知。
	case payment.MerchantNotifyTpPayInFraud:
		// 处理代收欺诈通知。
	}

	client.NotifySuccess(w)
}
```

通知类型：

| 类型 | 值 | 说明 |
|---|---:|---|
| `MerchantNotifyTpPayIn` | `1` | 代收订单通知 |
| `MerchantNotifyTpPayInRefund` | `2` | 代收退款通知 |
| `MerchantNotifyTpPayInDispute` | `3` | 代收争议通知 |
| `MerchantNotifyTpPayOut` | `4` | 代付订单通知 |
| `MerchantNotifyTpPayInAgreement` | `5` | 代收协议签约通知 |
| `MerchantNotifyTpPayInCardBind` | `6` | 信用卡绑卡通知 |
| `MerchantNotifyTpPayInFraud` | `7` | 代收欺诈通知 |

退款通知 `RefundOrder.Status`：

| 常量 | 值 | 说明 |
|---|---:|---|
| `RefundStatusPending` | `1` | 待处理 |
| `RefundStatusSuccess` | `2` | 退款成功 |
| `RefundStatusFailed` | `3` | 失败（含拒绝） |

争议通知 `DisputeOrder.Status`：

| 常量 | 值 | 说明 |
|---|---:|---|
| `DisputeStatusReceived` | `1` | 收到争议 |
| `DisputeStatusWin` | `2` | 商户赢 |
| `DisputeStatusLose` | `3` | 商户输 |
| `DisputeStatusCanceled` | `4` | 争议取消 |

当前 Payment 可能仍把欺诈折成类型 `1` 的失败订单通知。接入方应同时处理 `MerchantNotifyTpPayIn` 失败和 `MerchantNotifyTpPayInFraud`。

`NotifySuccess` 会返回 HTTP 200，表示商户已经成功处理通知。

`NotifyFailed` 会返回 HTTP 500，表示商户处理失败，Payment 可以按重试策略再次通知。

## 错误处理

接口返回非 2xx 时，SDK 会返回 `*payment.APIError`：

```go
var apiErr *payment.APIError
if errors.As(err, &apiErr) {
	fmt.Println(apiErr.StatusCode)
	fmt.Println(apiErr.Message)
	fmt.Println(string(apiErr.Body))
}
```

## 接入注意事项

- 所有金额单位都是分。
- `MerchantOrderID` 由商户生成，建议保证唯一。
- 写接口不要无脑自动重试，除非业务侧已经做好幂等。
- 通知处理必须先验签，再处理业务。
- 通知业务落库成功后再返回 `NotifySuccess`。
- 商户密钥只能放在服务端，不能传给前端。
