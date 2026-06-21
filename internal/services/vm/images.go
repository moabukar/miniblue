package vm

import (
	"fmt"
	"strings"
)

const defaultImage = "ubuntu:24.04"

// resolveImage maps the ARM request properties to a container image.
//
// Priority:
//  1. properties.miniblue.image — explicit container image (escape hatch)
//  2. properties.storageProfile.imageReference — mapped via the alias table
//  3. no image reference at all — defaultImage
//
// An imageReference that is present but cannot be mapped is an error so that
// typos fail loudly instead of silently booting the wrong OS.
func resolveImage(props map[string]interface{}) (string, error) {
	if mb, ok := props["miniblue"].(map[string]interface{}); ok {
		if img, ok := mb["image"].(string); ok && img != "" {
			return img, nil
		}
	}

	sp, _ := props["storageProfile"].(map[string]interface{})
	if sp == nil {
		return defaultImage, nil
	}
	ref, _ := sp["imageReference"].(map[string]interface{})
	if ref == nil {
		return defaultImage, nil
	}

	str := func(k string) string {
		v, _ := ref[k].(string)
		return strings.ToLower(v)
	}
	// Concatenate every identifying field so aliases match wherever the
	// caller put them (publisher:offer:sku, community alias, or plain id).
	all := strings.Join([]string{str("publisher"), str("offer"), str("sku"), str("id"), str("communityGalleryImageId")}, " ")

	switch {
	case strings.Contains(all, "24_04") || strings.Contains(all, "2404") || strings.Contains(all, "24.04"):
		return "ubuntu:24.04", nil
	case strings.Contains(all, "22_04") || strings.Contains(all, "2204") || strings.Contains(all, "22.04"):
		return "ubuntu:22.04", nil
	case strings.Contains(all, "ubuntu"):
		return defaultImage, nil
	case strings.Contains(all, "debian"):
		return "debian:stable-slim", nil
	case strings.TrimSpace(all) == "":
		return defaultImage, nil
	}
	return "", fmt.Errorf("unsupported imageReference (publisher=%q offer=%q sku=%q); set properties.miniblue.image to use an explicit container image", str("publisher"), str("offer"), str("sku"))
}
