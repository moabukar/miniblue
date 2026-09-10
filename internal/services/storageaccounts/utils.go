package storageaccounts

// src overwrites dst.
func DeepMerge(dst, src map[string]any) map[string]any {
	for k, vSrc := range src {
		if vDst, ok := dst[k]; ok {
			// If both values are maps, recursively merge them
			if mapDst, okDst := vDst.(map[string]any); okDst {
				if mapSrc, okSrc := vSrc.(map[string]any); okSrc {
					dst[k] = DeepMerge(mapDst, mapSrc)
					continue
				}
			}
		}
		// Base case: overwrite or add the value from src
		dst[k] = vSrc
	}
	return dst
}
