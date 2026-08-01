// Package aliyun implements provider.CloudProvider for Alibaba Cloud.
//
// This is a stub. It exists to remind us that iac-server is not coupled
// to HuaweiCloud — adding a cloud means implementing this interface +
// embedding a skills directory, nothing in server core or iac/ changes.
package aliyun

import (
	"io/fs"
	"os"

	"github.com/yusheng-g/openagent-go/cmd/mcp/iac-server/provider"
)

// Aliyun implements provider.CloudProvider.
type Aliyun struct {
	region string
}

// New creates an Aliyun provider for the given region.
// Credentials are read from the environment on demand via Env().
func New(region string) *Aliyun {
	return &Aliyun{region: region}
}

// Compile-time interface check.
var _ provider.CloudProvider = (*Aliyun)(nil)

// Name returns the cloud identifier.
func (a *Aliyun) Name() string { return "aliyun" }

// Env returns Aliyun credential environment variables.
// Reads from the process environment at call time so secrets never
// persist in the struct.
func (a *Aliyun) Env() map[string]string {
	return map[string]string{
		"ALICLOUD_ACCESS_KEY": os.Getenv("ALICLOUD_ACCESS_KEY"),
		"ALICLOUD_SECRET_KEY": os.Getenv("ALICLOUD_SECRET_KEY"),
		"ALICLOUD_REGION":     a.region,
	}
}

// Skills returns the embedded skills directory.
// nil until skills are embedded for this cloud.
func (a *Aliyun) Skills() fs.FS { return nil }
