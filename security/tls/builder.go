package tls

import (
	"crypto/tls"
	"fmt"

	tlspb "github.com/Servora-Kit/servora/api/gen/go/servora/security/tls/v1"
)

// BuildServerTLS 从 TLS proto 构造服务端 *tls.Config。
// 当 c 为 nil 或 enable=false 时返回 (nil, nil)，调用方据此决定是否启用 TLS。
func BuildServerTLS(c *tlspb.TLS) (*tls.Config, error) {
	if c == nil || !c.GetEnable() {
		return nil, nil
	}
	return NewServerConfig(ServerConfigOptions{
		CertPath: c.GetCertPath(),
		KeyPath:  c.GetKeyPath(),
	})
}

// BuildClientTLS 从 TLS proto 构造客户端 *tls.Config。
// 当 c 为 nil 或 enable=false 时返回 (nil, nil)，调用方据此决定是否启用 TLS。
func BuildClientTLS(c *tlspb.TLS) (*tls.Config, error) {
	return BuildClientTLSForServer(c, "")
}

// BuildClientTLSForServer 构造客户端 TLS，并显式绑定证书校验使用的 server name。
func BuildClientTLSForServer(c *tlspb.TLS, serverName string) (*tls.Config, error) {
	if c == nil || !c.GetEnable() {
		return nil, nil
	}
	return NewClientConfig(ClientConfigOptions{
		CAPath:     c.GetCaPath(),
		CertPath:   c.GetCertPath(),
		KeyPath:    c.GetKeyPath(),
		ServerName: serverName,
	})
}

// MustBuildServerTLS 是 BuildServerTLS 的 panic 版本，TLS 配置非法时直接 panic。
func MustBuildServerTLS(c *tlspb.TLS) *tls.Config {
	tlsCfg, err := BuildServerTLS(c)
	if err != nil {
		panic(fmt.Sprintf("build server TLS config: %v", err))
	}
	return tlsCfg
}
