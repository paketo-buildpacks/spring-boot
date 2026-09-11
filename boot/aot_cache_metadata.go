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

	// the sidecar may name the fields after the Java system properties instead
	JavaVersionProperty string `json:"java.version"`
	OsArchProperty      string `json:"os.arch"`
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
	if meta.JavaVersion == "" {
		meta.JavaVersion = meta.JavaVersionProperty
	}
	if meta.OsArch == "" {
		meta.OsArch = meta.OsArchProperty
	}
	// a sidecar that declares neither field is not something we can check anything against
	return meta, meta.JavaVersion != "" || meta.OsArch != "", nil
}

// validateAotCacheMetadata verifies that a pre-recorded AOT cache was produced for the given
// JRE. Only the fields the sidecar actually declared are compared. Callers only reach this
// with metadata that loadAotCacheMetadata reported as usable.
func validateAotCacheMetadata(meta AotCacheMetadata, jre JREProperties) (compatible bool, reason string) {
	if meta.JavaVersion != "" && meta.JavaVersion != jre.JavaVersion {
		return false,
			fmt.Sprintf("pre-recorded AOT cache was recorded with Java %s but the image JRE is Java %s", meta.JavaVersion, jre.JavaVersion)
	}
	if meta.OsArch != "" && meta.OsArch != jre.OsArch {
		return false,
			fmt.Sprintf("pre-recorded AOT cache was recorded for architecture %s but the image JRE architecture is %s", meta.OsArch, jre.OsArch)
	}
	return true, ""
}
