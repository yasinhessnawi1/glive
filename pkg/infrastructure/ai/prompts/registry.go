package prompts

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// PromptRegistry manages prompt versions and rollback
type PromptRegistry struct {
	manager *PromptManager
}

// NewPromptRegistry creates a new prompt registry
func NewPromptRegistry(manager *PromptManager) *PromptRegistry {
	return &PromptRegistry{
		manager: manager,
	}
}

// GetLatestVersion returns the latest version of a template
func (pr *PromptRegistry) GetLatestVersion(name string) (*PromptTemplate, error) {
	versions := pr.manager.ListVersions(name)
	if len(versions) == 0 {
		return nil, fmt.Errorf("template %s not found", name)
	}

	// Sort versions numerically
	sortedVersions := make([]int, 0, len(versions))
	versionMap := make(map[int]string)
	for _, v := range versions {
		if num, err := strconv.Atoi(v); err == nil {
			sortedVersions = append(sortedVersions, num)
			versionMap[num] = v
		}
	}

	if len(sortedVersions) == 0 {
		// Fallback to string comparison
		sort.Strings(versions)
		latest := versions[len(versions)-1]
		tmpl, ok := pr.manager.GetTemplateVersion(name, latest)
		if !ok {
			return nil, fmt.Errorf("template %s version %s not found", name, latest)
		}
		return tmpl, nil
	}

	sort.Ints(sortedVersions)
	latestNum := sortedVersions[len(sortedVersions)-1]
	latest := versionMap[latestNum]

	tmpl, ok := pr.manager.GetTemplateVersion(name, latest)
	if !ok {
		return nil, fmt.Errorf("template %s version %s not found", name, latest)
	}
	return tmpl, nil
}

// RollbackToVersion rolls back to a specific version
func (pr *PromptRegistry) RollbackToVersion(name, version string) error {
	tmpl, ok := pr.manager.GetTemplateVersion(name, version)
	if !ok {
		return fmt.Errorf("template %s version %s not found", name, version)
	}

	// Register as latest version
	pr.manager.RegisterTemplate(tmpl)

	return nil
}

// ListAllVersions returns all versions of a template sorted
func (pr *PromptRegistry) ListAllVersions(name string) ([]*PromptTemplate, error) {
	versions := pr.manager.ListVersions(name)
	if len(versions) == 0 {
		return nil, fmt.Errorf("template %s not found", name)
	}

	// Sort versions
	sortedVersions := make([]int, 0, len(versions))
	versionMap := make(map[int]string)
	for _, v := range versions {
		if num, err := strconv.Atoi(v); err == nil {
			sortedVersions = append(sortedVersions, num)
			versionMap[num] = v
		}
	}

	sort.Ints(sortedVersions)

	result := make([]*PromptTemplate, 0, len(sortedVersions))
	for _, num := range sortedVersions {
		v := versionMap[num]
		if tmpl, ok := pr.manager.GetTemplateVersion(name, v); ok {
			result = append(result, tmpl)
		}
	}

	return result, nil
}

// CompareVersions compares two versions and returns which is newer
func (pr *PromptRegistry) CompareVersions(v1, v2 string) (int, error) {
	// Remove "v" prefix if present
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	num1, err1 := strconv.Atoi(v1)
	num2, err2 := strconv.Atoi(v2)

	if err1 != nil || err2 != nil {
		// String comparison fallback
		if v1 < v2 {
			return -1, nil
		} else if v1 > v2 {
			return 1, nil
		}
		return 0, nil
	}

	if num1 < num2 {
		return -1, nil
	} else if num1 > num2 {
		return 1, nil
	}
	return 0, nil
}

// GetVersionHistory returns version history for a template
func (pr *PromptRegistry) GetVersionHistory(name string) ([]VersionInfo, error) {
	versions, err := pr.ListAllVersions(name)
	if err != nil {
		return nil, err
	}

	history := make([]VersionInfo, 0, len(versions))
	for _, tmpl := range versions {
		history = append(history, VersionInfo{
			Version:      tmpl.Version,
			LastModified: tmpl.LastModified.Format("2006-01-02 15:04:05"),
			Category:     tmpl.Category,
			FilePath:     tmpl.FilePath,
		})
	}

	return history, nil
}

// VersionInfo contains information about a template version
type VersionInfo struct {
	Version      string
	LastModified string
	Category     string
	FilePath     string
}
