package claudeauth

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/tsai41/claude-account-manager/internal/paths"
)

// AccountMeta is what we extract from ~/.claude.json oauthAccount.
type AccountMeta struct {
	Email          string `json:"email,omitempty"`
	AccountUUID    string `json:"account_uuid,omitempty"`
	OrgUUID        string `json:"org_uuid,omitempty"`
	OrgName        string `json:"org_name,omitempty"`
	OrgRole        string `json:"org_role,omitempty"`
	WorkspaceRole  string `json:"workspace_role,omitempty"`
	OrgType        string `json:"org_type,omitempty"`
}

// ReadAccountMeta extracts oauthAccount info from ~/.claude.json. Returns zero-value meta if absent.
func ReadAccountMeta() (AccountMeta, error) {
	var meta AccountMeta
	b, err := os.ReadFile(paths.ClaudeJSON())
	if err != nil {
		if os.IsNotExist(err) {
			return meta, nil
		}
		return meta, fmt.Errorf("read %s: %w", paths.ClaudeJSON(), err)
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return meta, fmt.Errorf("parse %s: %w", paths.ClaudeJSON(), err)
	}
	oa, ok := raw["oauthAccount"].(map[string]any)
	if !ok {
		return meta, nil
	}
	getStr := func(k string) string {
		if v, ok := oa[k].(string); ok {
			return v
		}
		return ""
	}
	meta.Email = getStr("emailAddress")
	meta.AccountUUID = getStr("accountUuid")
	meta.OrgUUID = getStr("organizationUuid")
	meta.OrgName = getStr("organizationName")
	meta.OrgRole = getStr("organizationRole")
	meta.WorkspaceRole = getStr("workspaceRole")
	meta.OrgType = getStr("organizationType")
	return meta, nil
}
