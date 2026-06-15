package vm

import (
	"fmt"
	"strings"

	"github.com/moabukar/miniblue/internal/store"
)

// ---------------------------------------------------------------------------
// User-assigned managed identity block handling (VM side)
//
// The identities themselves are owned by the identity service
// (Microsoft.ManagedIdentity/userAssignedIdentities, store prefix
// "msi:identity:"); here we only validate references and resolve their
// principalId/clientId into the VM response.
// ---------------------------------------------------------------------------

// identityStoreKeyFromARMID converts a userAssignedIdentities ARM id into its
// store key. Returns an error for malformed ids.
func identityStoreKeyFromARMID(armID string) (string, error) {
	parts := strings.Split(strings.Trim(armID, "/"), "/")
	// subscriptions/<sub>/resourceGroups/<rg>/providers/Microsoft.ManagedIdentity/userAssignedIdentities/<name>
	if len(parts) != 8 ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "resourceGroups") ||
		!strings.EqualFold(parts[4], "providers") ||
		!strings.EqualFold(parts[5], "Microsoft.ManagedIdentity") ||
		!strings.EqualFold(parts[6], "userAssignedIdentities") {
		return "", fmt.Errorf("'%s' is not a valid userAssignedIdentities resource id", armID)
	}
	return "msi:identity:" + parts[1] + ":" + parts[3] + ":" + parts[7], nil
}

// resolveIdentityBlock validates the request's identity block against
// existing identities and returns the block to store, with principalId and
// clientId resolved per assignment. A nil return with nil error means the VM
// has no identity ("type": "None" or no block at all).
func resolveIdentityBlock(s *store.Store, input map[string]interface{}) (map[string]interface{}, error) {
	block, _ := input["identity"].(map[string]interface{})
	if block == nil {
		return nil, nil
	}
	idType, _ := block["type"].(string)
	if strings.EqualFold(idType, "None") || idType == "" {
		return nil, nil
	}
	if !strings.EqualFold(idType, "UserAssigned") {
		return nil, fmt.Errorf("identity type '%s' is not supported; only 'UserAssigned' (or 'None') is available", idType)
	}

	assignments, _ := block["userAssignedIdentities"].(map[string]interface{})
	if len(assignments) == 0 {
		return nil, fmt.Errorf("identity type 'UserAssigned' requires at least one entry in userAssignedIdentities")
	}

	resolved := make(map[string]interface{}, len(assignments))
	for armID := range assignments {
		key, err := identityStoreKeyFromARMID(armID)
		if err != nil {
			return nil, err
		}
		v, ok := s.Get(key)
		if !ok {
			return nil, fmt.Errorf("the managed identity '%s' could not be found", armID)
		}
		idRec, _ := v.(map[string]interface{})
		idProps, _ := idRec["properties"].(map[string]interface{})
		entry := map[string]interface{}{}
		if idProps != nil {
			entry["principalId"] = idProps["principalId"]
			entry["clientId"] = idProps["clientId"]
		}
		resolved[armID] = entry
	}

	return map[string]interface{}{
		"type":                   "UserAssigned",
		"userAssignedIdentities": resolved,
	}, nil
}
