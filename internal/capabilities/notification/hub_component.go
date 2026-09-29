package notification

import "context"

// HubComponent 把 WebSocket Hub 的消费循环纳入应用生命周期（contract.Component）。
type HubComponent struct {
	hub    *Hub
	cancel context.CancelFunc
}

// NewHubComponent 创建 Hub 生命周期组件。
func NewHubComponent(hub *Hub) *HubComponent { return &HubComponent{hub: hub} }

// Start 在后台运行 Hub 的消息分发循环。
func (c *HubComponent) Start(context.Context) error {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	go c.hub.Run(ctx)
	return nil
}

// Stop 停止 Hub 的消息分发循环。
func (c *HubComponent) Stop(context.Context) error {
	if c.cancel != nil {
		c.cancel()
	}
	return nil
}
