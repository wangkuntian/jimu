// internal/platform/oauth/wechat.go
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"jimu/internal/platform/httpclient"

	"golang.org/x/oauth2"
)

// WeChatConfig 微信 OAuth 配置
type WeChatConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// WeChatProvider 微信 OAuth 实现
type WeChatProvider struct {
	config      *oauth2.Config
	client      *httpclient.Client
	userInfoURL string // 未导出，测试可覆盖
}

// NewWeChatProvider 创建微信 Provider
func NewWeChatProvider(cfg WeChatConfig, client *httpclient.Client) *WeChatProvider {
	return &WeChatProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://open.weixin.qq.com/connect/qrconnect",
				TokenURL: "https://api.weixin.qq.com/sns/oauth2/access_token",
			},
		},
		client:      client,
		userInfoURL: "https://api.weixin.qq.com/sns/userinfo",
	}
}

// Name 返回提供商名称
func (p *WeChatProvider) Name() string { return "wechat" }

// AuthURL 构造授权跳转 URL
func (p *WeChatProvider) AuthURL(state string) string {
	return p.config.AuthCodeURL(state)
}

// Exchange 用授权码换取用户信息
func (p *WeChatProvider) Exchange(ctx context.Context, code string) (*UserInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, oauthTimeout)
	defer cancel()
	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("wechat exchange: %w", err)
	}
	openid, ok := token.Extra("openid").(string)
	if !ok || openid == "" {
		return nil, fmt.Errorf("wechat openid missing")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.userInfoURL+"?access_token="+token.AccessToken+"&openid="+openid, nil)
	if err != nil {
		return nil, fmt.Errorf("wechat userinfo req: %w", err)
	}
	resp, err := p.client.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("wechat userinfo: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read wechat userinfo: %w", err)
	}
	var info struct {
		OpenID   string `json:"openid"`
		UnionID  string `json:"unionid"`
		Nickname string `json:"nickname"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("unmarshal wechat userinfo: %w", err)
	}
	return &UserInfo{Subject: info.OpenID, Email: "", Name: info.Nickname}, nil
}

var _ Provider = (*WeChatProvider)(nil)
