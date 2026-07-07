package client

import (
	"github.com/user/kross/internal/license"
)

// ProcessRevocations checks the current license against newly received revocations.
// If the current license key is in the revoke list, it removes the stored license.
func (c *Client) ProcessRevocations(revokeKeyIDs []string) error {
	if len(revokeKeyIDs) == 0 {
		return nil
	}

	if err := c.store.AddToBlacklist(revokeKeyIDs); err != nil {
		return err
	}

	data, err := c.store.LoadLicense()
	if err != nil {
		return nil // No active license to revoke
	}

	signedLic, err := license.Decode(data.Key)
	if err != nil {
		return nil
	}

	for _, revokedID := range revokeKeyIDs {
		if signedLic.License.ID == revokedID {
			return c.store.RemoveLicense()
		}
	}

	return nil
}
