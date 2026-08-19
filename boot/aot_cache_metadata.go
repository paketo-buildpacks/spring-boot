package boot

import (
	"encoding/json"
	"fmt"
	"os"
)

// AotCacheMetadata describes the JDK and platform that produced a pre-recorded AOT cache.
// It is recorded by the Spring Boot build plugin alongside the cache file so the buildpack
// can verify the cache is compatible with the JRE baked into the image at build time.
type AotCacheMetadata struct {
	JavaVersion string `json:"javaVersion"`
	OsArch      string `json:"osArch"`
}

// loadAotCacheMetadata reads the metadata sidecar for a pre-recorded AOT cache. The sidecar
// lives next to the cache file as `application.aot.meta`. If the sidecar is absent, ok is
// false, which lets callers decide whether to warn or fail depending on their posture.
func loadAotCacheMetadata(cacheFile string) (AotCacheMetadata, bool, error) {
	metaFile := cacheFile + ".meta"
	data, err := os.ReadFile(metaFile)
	if err != nil {
		if os.IsNotExist(err) {
			return AotCacheMetadata{}, false, nil
		}
		return AotCacheMetadata{}, false, fmt.Errorf("error reading AOT cache metadata at %s\n%w", metaFile, err)
	}

	var meta AotCacheMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return AotCacheMetadata{}, false, fmt.Errorf("error parsing AOT cache metadata at %s\n%w", metaFile, err)
	}
	return meta, true, nil
}

// validateAotCacheMetadata verifies that a pre-recorded AOT cache was produced for the given
// JRE. When metadata is absent (ok false), it returns an error only if strict is true.
// Otherwise it returns a boolean indicating whether the cache is compatible.
func validateAotCacheMetadata(meta AotCacheMetadata, present bool, jre JREProperties, strict bool) (compatible bool, reason string, err error) {
	if !present {
		if strict {
			return false, "", fmt.Errorf("AOT cache metadata is missing; the cache cannot be verified as compatible with the runtime JRE")
		}
		return true, "", nil
	}

	if meta.JavaVersion != "" && meta.JavaVersion != jre.JavaVersion {
		return false,
			fmt.Sprintf("pre-recorded AOT cache was recorded with Java %s but the image JRE is Java %s", meta.JavaVersion, jre.JavaVersion),
			nil
	}
	if meta.OsArch != "" && meta.OsArch != jre.OsArch {
		return false,
			fmt.Sprintf("pre-recorded AOT cache was recorded for architecture %s but the image JRE architecture is %s", meta.OsArch, jre.OsArch),
			nil
	}
	return true, "", nil
}
