package kiro

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sys/windows/registry"
)

var (
	kiroClientProductOnce sync.Once
	kiroClientProduct     string
)

// ClientUserAgentProduct returns the same stable custom product identifier used
// by the installed Kiro client: KiroIDE, its installed version, and the hashed
// Windows machine ID. No synthetic device attributes are generated.
func ClientUserAgentProduct() string {
	kiroClientProductOnce.Do(func() {
		kiroClientProduct = "KiroIDE-" + installedKiroVersion() + "-" + hashedWindowsMachineID()
	})
	return kiroClientProduct
}

func installedKiroVersion() string {
	localAppData := strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
	if localAppData == "" {
		return "0.0.0"
	}
	data, err := os.ReadFile(filepath.Join(localAppData, "Programs", "Kiro", "resources", "app", "package.json"))
	if err != nil {
		return "0.0.0"
	}
	var manifest struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(data, &manifest) != nil || strings.TrimSpace(manifest.Version) == "" {
		return "0.0.0"
	}
	return strings.TrimSpace(manifest.Version)
}

func hashedWindowsMachineID() string {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE)
	if err != nil {
		return "UNDETERMINED_MACHINE_ID"
	}
	defer key.Close()
	machineGUID, _, err := key.GetStringValue("MachineGuid")
	if err != nil || strings.TrimSpace(machineGUID) == "" {
		return "UNDETERMINED_MACHINE_ID"
	}
	digest := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(machineGUID))))
	return hex.EncodeToString(digest[:])
}
