package delivery

import (
	"context"
	"net"
	"net/url"
	"strings"

	"markpost/internal/domain/delivery"
	"markpost/internal/service"
)

// I.3 SSRF 防护（关键安全）：
// webhook_url 是用户可控的服务器端请求入口。创建/更新渠道时拒绝：
//   - 非 http/https scheme
//   - host 为私有/保留 IP 段（含云元数据 169.254.0.0/16）
//   - DNS 解析结果落在私有/保留段（防 DNS rebinding 到内网）
// 命中一律返回 ErrWebhookURLForbidden（422，webhook_url_forbidden）。

var blockedCIDRs = []*net.IPNet{
	// IPv4
	mustCIDR("10.0.0.0/8"),
	mustCIDR("172.16.0.0/12"),
	mustCIDR("192.168.0.0/16"),
	mustCIDR("127.0.0.0/8"),
	mustCIDR("169.254.0.0/16"), // link-local，含云元数据
	// IPv6
	mustCIDR("::1/128"),
	mustCIDR("fc00::/7"), // ULA
}

func mustCIDR(s string) *net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		panic("invalid cidr " + s)
	}
	return n
}

func isBlockedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	for _, cidr := range blockedCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// validateWebhookURLSSRF enforces the SSRF policy for a webhook URL (I.3):
// http(s) scheme only; the host must not resolve to private/reserved addresses.
func validateWebhookURLSSRF(ctx context.Context, rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return service.New(ErrWebhookURLForbidden, "webhook URL is not allowed")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return service.New(ErrWebhookURLForbidden, "webhook URL is not allowed")
	}
	host := u.Hostname()
	if host == "" {
		return service.New(ErrWebhookURLForbidden, "webhook URL is not allowed")
	}

	// Host 本身是 IP 字面量：直接校验。
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return service.New(ErrWebhookURLForbidden, "webhook URL is not allowed")
		}
		return nil
	}

	// 域名：DNS 解析后校验所有解析结果（防 DNS rebinding）。
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return service.New(ErrWebhookURLForbidden, "webhook URL is not allowed")
	}
	if len(addrs) == 0 {
		return service.New(ErrWebhookURLForbidden, "webhook URL is not allowed")
	}
	for _, a := range addrs {
		if isBlockedIP(a.IP) {
			return service.New(ErrWebhookURLForbidden, "webhook URL is not allowed")
		}
	}
	return nil
}

// isPublicWebhookURL is a small helper mirroring the scheme+host checks used in
// validateConfiguration so the SSRF gate stays colocated with the syntax check.
func isAllowedWebhookURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return strings.TrimSpace(u.Host) != ""
}

// ValidateChannelSSRF enforces the I.3 SSRF policy for a channel's webhook URL
// on top of an already-parsed configuration map. The user-facing path runs the
// same checks inside validateConfiguration; this export exists so the admin
// channel creation path (I.3 applies unconditionally to "创建/更新渠道") cannot
// bypass the gate.
func ValidateChannelSSRF(ctx context.Context, kind delivery.ChannelKind, config delivery.ChannelConfiguration) error {
	if kind != delivery.ChannelKindFeishu {
		return service.New(ErrWebhookURLForbidden, "unsupported channel kind")
	}
	webhookURL := strings.TrimSpace(config.Feishu().WebhookURL)
	if webhookURL == "" {
		return service.New(service.ErrValidation, "webhook URL is required")
	}
	if !isAllowedWebhookURL(webhookURL) {
		return service.New(service.ErrValidation, "invalid webhook URL: must be a valid HTTP or HTTPS URL")
	}
	return validateWebhookURLSSRF(ctx, webhookURL)
}
